package authorization

import (
	"context"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/handler/pkce"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

func (s *Store) CreateAuthorizeCodeSession(ctx context.Context, signature string, request fosite.Requester) error {
	return s.save(ctx, "code", signature, request, fosite.AuthorizeCode)
}
func (s *Store) GetAuthorizeCodeSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.load(ctx, "code", signature, session)
}
func (s *Store) InvalidateAuthorizeCodeSession(ctx context.Context, signature string) error {
	n, err := s.database(ctx).ProtocolSession.Update().Where(protocolsession.ID(signatureID("code", signature)), protocolsession.Revoked(false), protocolsession.ExpiresAtGT(time.Now())).SetRevoked(true).Save(ctx)
	if err == nil && n != 1 {
		return fosite.ErrInvalidatedAuthorizeCode
	}
	return err
}
func (s *Store) CreatePKCERequestSession(ctx context.Context, signature string, request fosite.Requester) error {
	return s.save(ctx, "pkce", signature, request, fosite.AuthorizeCode)
}
func (s *Store) GetPKCERequestSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.load(ctx, "pkce", signature, session)
}
func (s *Store) DeletePKCERequestSession(ctx context.Context, signature string) error {
	return s.remove(ctx, "pkce", signature)
}
func (s *Store) CreateOpenIDConnectSession(ctx context.Context, code string, request fosite.Requester) error {
	return s.save(ctx, "oidc", code, request, fosite.AuthorizeCode)
}
func (s *Store) GetOpenIDConnectSession(ctx context.Context, code string, request fosite.Requester) (fosite.Requester, error) {
	return s.load(ctx, "oidc", code, request.GetSession())
}
func (s *Store) DeleteOpenIDConnectSession(ctx context.Context, code string) error {
	return s.remove(ctx, "oidc", code)
}
func (s *Store) remove(ctx context.Context, kind, signature string) error {
	n, err := s.database(ctx).ProtocolSession.Delete().Where(protocolsession.ID(signatureID(kind, signature))).Exec(ctx)
	if err == nil && n != 1 {
		return fosite.ErrNotFound
	}
	return err
}
func (s *Store) RevokeAccessToken(ctx context.Context, requestID string) error {
	_, err := s.database(ctx).ProtocolSession.Update().Where(protocolsession.RequestID(requestID), protocolsession.Kind("access")).SetRevoked(true).Save(ctx)
	if err == nil {
		if state, ok := ctx.Value(transactionKey{}).(transaction); ok {
			*state.replayed = true
		}
	}
	return err
}

var _ openid.OpenIDConnectRequestStorage = (*Store)(nil)
var _ pkce.PKCERequestStorage = (*Store)(nil)
