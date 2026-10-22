package authorization

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
	"moetopia.local/aio/api/internal/persistence/ent/clienttestsubject"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

const MaxTestSubjects = 20

var ErrTestSubjectConflict = errors.New("test subject state changed")
var ErrTestSubjectLimit = errors.New("test subject limit reached")

type TestSubject struct {
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) allowSubject(ctx context.Context, clientID, userID string) error {
	db := persistence.Client(ctx, s.database(ctx))
	row, err := db.OAuthClient.Query().Where(oauthclient.ID(clientID)).Select(oauthclient.FieldID, oauthclient.FieldOwnerID, oauthclient.FieldTestOnly, oauthclient.FieldEnabled).Only(ctx)
	if ent.IsNotFound(err) {
		return ErrGrantMissing
	}
	if err != nil {
		return err
	}
	if !row.Enabled {
		return ErrGrantMissing
	}
	if !row.TestOnly || row.OwnerID == userID {
		return nil
	}
	allowed, err := db.ClientTestSubject.Query().Where(clienttestsubject.ClientID(clientID), clienttestsubject.UserID(userID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrGrantMissing
	}
	return nil
}

func (p *Provider) ListTestSubjects(ctx context.Context, clientID, owner string) ([]TestSubject, error) {
	db := persistence.Client(ctx, p.store.client)
	owned, err := db.OAuthClient.Query().Where(oauthclient.ID(clientID), oauthclient.OwnerID(owner)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if owner == "" || !owned {
		return nil, ErrGrantMissing
	}
	rows, err := db.ClientTestSubject.Query().Where(clienttestsubject.ClientID(clientID)).Order(ent.Asc(clienttestsubject.FieldCreatedAt), ent.Asc(clienttestsubject.FieldID)).Limit(MaxTestSubjects).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]TestSubject, 0, len(rows))
	for _, row := range rows {
		result = append(result, TestSubject{row.UserID, row.CreatedAt})
	}
	return result, nil
}

// The caller holds the owner/application locks and supplies the shared transaction.
func (p *Provider) ChangeTestSubject(ctx context.Context, clientID, owner, userID string, add bool) error {
	db := persistence.Client(ctx, p.store.client)
	row, err := db.OAuthClient.Query().Where(oauthclient.ID(clientID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	if owner == "" || row.OwnerID != owner || userID == owner {
		return ErrGrantMissing
	}
	query := db.ClientTestSubject.Query().Where(clienttestsubject.ClientID(clientID), clienttestsubject.UserID(userID))
	exists, err := query.Exist(ctx)
	if err != nil {
		return err
	}
	if exists == add || add && !row.TestOnly {
		return ErrTestSubjectConflict
	}
	if add {
		count, err := db.ClientTestSubject.Query().Where(clienttestsubject.ClientID(clientID)).Count(ctx)
		if err != nil {
			return err
		}
		if count >= MaxTestSubjects {
			return ErrTestSubjectLimit
		}
		return db.ClientTestSubject.Create().SetID(uuid.NewString()).SetClientID(clientID).SetUserID(userID).Exec(ctx)
	}
	if _, err := db.ClientTestSubject.Delete().Where(clienttestsubject.ClientID(clientID), clienttestsubject.UserID(userID)).Exec(ctx); err != nil {
		return err
	}
	if !row.TestOnly {
		return nil
	}
	ids, err := db.AuthorizationGrant.Query().Where(authorizationgrant.ClientID(clientID), authorizationgrant.UserID(userID)).Order(ent.Asc(authorizationgrant.FieldID)).IDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := db.AuthorizationGrant.Query().Where(authorizationgrant.ID(id)).ForUpdate().Only(ctx); err != nil {
			return err
		}
		if err := db.AuthorizationGrant.UpdateOneID(id).SetRevoked(true).Exec(ctx); err != nil {
			return err
		}
		if _, err := db.ProtocolSession.Update().Where(protocolsession.RequestID(id)).SetRevoked(true).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
