package authorization

import (
	"context"
	"time"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
)

const refreshIdleLifetime = 24 * time.Hour
const grantLifetime = 7 * 24 * time.Hour

func (s *Store) ensureGrant(ctx context.Context, request fosite.Requester, tokenType fosite.TokenType) error {
	_, err := s.database(ctx).AuthorizationGrant.Get(ctx, request.GetID())
	if !ent.IsNotFound(err) {
		return err
	}
	expires := request.GetSession().GetExpiresAt(tokenType)
	if tokenType == fosite.AuthorizeCode {
		expires = expires.Add(10 * time.Minute)
	}
	if request.GetGrantedScopes().Has("offline_access") {
		expires = time.Now().Add(grantLifetime)
	}
	create := s.database(ctx).AuthorizationGrant.Create().SetID(request.GetID()).SetClientID(request.GetClient().GetID()).SetExpiresAt(expires).SetScopes(request.GetGrantedScopes()).SetAudiences(request.GetGrantedAudience())
	if tokenType == fosite.AuthorizeCode {
		create.SetUserID(request.GetSession().GetSubject())
		if session, ok := request.GetSession().(*Session); ok {
			create.SetSecurityVersion(session.SecurityVersion)
		}
	}
	_, err = create.Save(ctx)
	return err
}
func (s *Store) grant(ctx context.Context, id string) (*ent.AuthorizationGrant, error) {
	query := s.database(ctx).AuthorizationGrant.Query().Where(authorizationgrant.ID(id))
	if _, ok := ctx.Value(transactionKey{}).(transaction); ok {
		query.ForUpdate()
	}
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, fosite.ErrNotFound
	}
	return row, err
}
func (s *Store) revokeGrant(ctx context.Context, id string) error {
	if _, err := s.grant(ctx, id); err != nil {
		return err
	}
	_, err := s.database(ctx).AuthorizationGrant.UpdateOneID(id).SetRevoked(true).Save(ctx)
	return err
}
