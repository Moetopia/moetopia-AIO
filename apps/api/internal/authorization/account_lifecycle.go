package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"entgo.io/ent/dialect/sql"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
	"moetopia.local/aio/api/internal/persistence/ent/clienttestsubject"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

func (p *Provider) PurgeAccount(ctx context.Context, userID string) error {
	db := persistence.Client(ctx, p.store.client)
	legacy, err := p.legacySubjectGrants(ctx, userID)
	if err != nil {
		return err
	}
	owned, err := db.OAuthClient.Query().Where(oauthclient.OwnerID(userID)).Order(ent.Asc(oauthclient.FieldID)).IDs(ctx)
	if err != nil {
		return err
	}
	rows, err := db.AuthorizationGrant.Query().Where(authorizationgrant.Or(authorizationgrant.UserID(userID), authorizationgrant.IDIn(legacy...))).Select(authorizationgrant.FieldClientID).All(ctx)
	if err != nil {
		return err
	}
	clients := append([]string{}, owned...)
	memberships, err := db.ClientTestSubject.Query().Where(clienttestsubject.UserID(userID)).All(ctx)
	if err != nil {
		return err
	}
	for _, membership := range memberships {
		clients = append(clients, membership.ClientID)
	}
	for _, row := range rows {
		clients = append(clients, row.ClientID)
	}
	slices.Sort(clients)
	clients = slices.Compact(clients)
	for _, id := range clients {
		_, err := db.OAuthClient.Query().Where(oauthclient.ID(id)).ForUpdate().Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
	}
	if _, err := db.ClientTestSubject.Delete().Where(clienttestsubject.UserID(userID)).Exec(ctx); err != nil {
		return err
	}
	for {
		ids, err := db.AuthorizationGrant.Query().Where(authorizationgrant.Or(authorizationgrant.UserID(userID), authorizationgrant.ClientIDIn(owned...), authorizationgrant.IDIn(legacy...))).Order(ent.Asc(authorizationgrant.FieldID)).Limit(100).ForUpdate().IDs(ctx)
		if err != nil || len(ids) == 0 {
			if err != nil {
				return err
			}
			break
		}
		if _, err := db.ProtocolSession.Delete().Where(protocolsession.RequestIDIn(ids...)).Exec(ctx); err != nil {
			return err
		}
		if _, err := db.AuthorizationGrant.Delete().Where(authorizationgrant.IDIn(ids...)).Exec(ctx); err != nil {
			return err
		}
	}
	// Remove orphaned retained snapshots for managed clients as well as their grants.
	if _, err := db.ProtocolSession.Delete().Where(protocolsession.ClientIDIn(owned...)).Exec(ctx); err != nil {
		return err
	}
	_, err = db.OAuthClient.Delete().Where(oauthclient.OwnerID(userID)).Exec(ctx)
	return err
}

func (p *Provider) legacySubjectGrants(ctx context.Context, userID string) ([]string, error) {
	result := []string{}
	after := ""
	for {
		rows, err := persistence.Client(ctx, p.store.client).ProtocolSession.Query().Where(protocolsession.Kind("code"), protocolsession.IDGT(after), func(selector *sql.Selector) {
			grants := sql.Table("grants").Schema("authorization")
			selector.Where(sql.Exists(sql.Select(grants.C("id")).From(grants).Where(sql.And(sql.ColumnsEQ(grants.C("id"), selector.C("request_id")), sql.EQ(grants.C("user_id"), "")))))
		}).Order(ent.Asc(protocolsession.FieldID)).Limit(100).All(ctx)
		if err != nil || len(rows) == 0 {
			return result, err
		}
		for _, row := range rows {
			after = row.ID
			plain, err := p.store.box.Open(row.Payload, "protocol:"+row.ID)
			if err != nil {
				return nil, errors.New("legacy authorization ownership could not be recovered")
			}
			var snapshot requestSnapshot
			if err := json.Unmarshal(plain, &snapshot); err != nil || snapshot.ID != row.RequestID || snapshot.ClientID != row.ClientID {
				return nil, errors.New("legacy authorization ownership is invalid")
			}
			session := newSession("")
			if err := json.Unmarshal(snapshot.Session, session); err != nil {
				return nil, errors.New("legacy authorization identity is invalid")
			}
			if session.GetSubject() == userID {
				result = append(result, row.RequestID)
			}
		}
	}
}
