package authorization

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/ory/fosite"
	"time"

	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

var ErrGrantMissing = errors.New("authorization grant unavailable")
var ErrGrantCursor = errors.New("invalid authorization cursor")

type UserGrant struct {
	ID         string    `json:"id"`
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	Scopes     []string  `json:"scopes"`
	Audiences  []string  `json:"audiences"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}
type UserGrants struct {
	Items      []UserGrant `json:"items"`
	NextCursor string      `json:"next_cursor"`
}
type grantCursor struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func (p *Provider) UserGrants(ctx context.Context, userID string, securityVersion int, cursor string) (UserGrants, error) {
	result := UserGrants{Items: []UserGrant{}}
	query := p.store.client.AuthorizationGrant.Query().Where(authorizationgrant.UserID(userID), authorizationgrant.SecurityVersion(securityVersion), authorizationgrant.Revoked(false), authorizationgrant.ExpiresAtGT(time.Now()))
	if cursor != "" {
		if len(cursor) > 300 {
			return result, ErrGrantCursor
		}
		value, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return result, ErrGrantCursor
		}
		var c grantCursor
		if json.Unmarshal(value, &c) != nil || c.At.IsZero() || len(c.ID) > 128 || c.ID == "" {
			return result, ErrGrantCursor
		}
		query.Where(authorizationgrant.Or(authorizationgrant.CreatedAtLT(c.At), authorizationgrant.And(authorizationgrant.CreatedAtEQ(c.At), authorizationgrant.IDLT(c.ID))))
	}
	rows, err := query.Order(ent.Desc(authorizationgrant.FieldCreatedAt), ent.Desc(authorizationgrant.FieldID)).Limit(51).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 50 {
		rows = rows[:50]
		last := rows[len(rows)-1]
		encoded, _ := json.Marshal(grantCursor{last.CreatedAt, last.ID})
		result.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ClientID)
	}
	clients, err := p.store.client.OAuthClient.Query().Where(oauthclient.IDIn(ids...)).All(ctx)
	if err != nil {
		return result, err
	}
	names := map[string]string{}
	for _, client := range clients {
		names[client.ID] = client.Name
	}
	for _, row := range rows {
		result.Items = append(result.Items, UserGrant{row.ID, row.ClientID, names[row.ClientID], row.Scopes, row.Audiences, row.CreatedAt, row.ExpiresAt})
	}
	return result, nil
}
func (p *Provider) RevokeUserGrant(ctx context.Context, userID, id string) error {
	ctx, err := p.store.BeginTX(ctx)
	if err != nil {
		return err
	}
	defer p.store.Rollback(ctx)
	row, err := p.store.grant(ctx, id)
	if err != nil {
		if errors.Is(err, fosite.ErrNotFound) {
			return ErrGrantMissing
		}
		return err
	}
	if row.UserID != userID {
		return ErrGrantMissing
	}
	if err := p.store.database(ctx).AuthorizationGrant.UpdateOneID(id).SetRevoked(true).Exec(ctx); err != nil {
		return err
	}
	if _, err := p.store.database(ctx).ProtocolSession.Update().Where(protocolsession.RequestID(id)).SetRevoked(true).Save(ctx); err != nil {
		return err
	}
	if err := audit.RecordTarget(ctx, p.store.database(ctx), userID, "authorization.revoked", id, "success"); err != nil {
		return err
	}
	return p.store.Commit(ctx)
}
func BackfillGrantSubjects(ctx context.Context, store *Store) (int, error) {
	total := 0
	after := ""
	for {
		rows, err := store.client.ProtocolSession.Query().Where(protocolsession.Kind("code"), protocolsession.IDGT(after)).Order(ent.Asc(protocolsession.FieldID)).Limit(100).All(ctx)
		if err != nil {
			return total, err
		}
		if len(rows) == 0 {
			return total, nil
		}
		for _, row := range rows {
			after = row.ID
			grant, err := store.client.AuthorizationGrant.Get(ctx, row.RequestID)
			if ent.IsNotFound(err) {
				continue
			}
			if err != nil {
				return total, err
			}
			if grant.UserID != "" || grant.Revoked || !time.Now().Before(grant.ExpiresAt) {
				continue
			}
			plain, err := store.box.Open(row.Payload, "protocol:"+row.ID)
			if err != nil {
				return total, errors.New("legacy authorization snapshot could not be decrypted")
			}
			var snapshot requestSnapshot
			if err := json.Unmarshal(plain, &snapshot); err != nil {
				return total, errors.New("legacy authorization snapshot is invalid")
			}
			session := newSession("")
			if err := json.Unmarshal(snapshot.Session, session); err != nil {
				return total, err
			}
			if session.GetSubject() == "" {
				continue
			}
			if snapshot.ID != row.RequestID || snapshot.ClientID != row.ClientID {
				return total, errors.New("legacy authorization snapshot ownership mismatch")
			}
			count, err := store.client.AuthorizationGrant.Update().Where(authorizationgrant.ID(row.RequestID), authorizationgrant.UserID("")).SetUserID(session.GetSubject()).SetSecurityVersion(session.SecurityVersion).SetScopes(snapshot.Scopes).SetAudiences(snapshot.Audience).Save(ctx)
			if err != nil {
				return total, err
			}
			total += count
		}
	}
}
