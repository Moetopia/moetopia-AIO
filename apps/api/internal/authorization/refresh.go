package authorization

import (
	"context"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
	"time"
)

func (s *Store) CreateRefreshTokenSession(ctx context.Context, signature, _ string, request fosite.Requester) error {
	return s.save(ctx, "refresh", signature, request, fosite.RefreshToken)
}
func (s *Store) GetRefreshTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.load(ctx, "refresh", signature, session)
}
func (s *Store) DeleteRefreshTokenSession(ctx context.Context, signature string) error {
	_, err := s.database(ctx).ProtocolSession.Update().Where(protocolsession.ID(signatureID("refresh", signature))).SetRevoked(true).Save(ctx)
	return err
}
func (s *Store) RotateRefreshToken(ctx context.Context, requestID, signature string) error {
	grant, err := s.grant(ctx, requestID)
	if err != nil {
		return err
	}
	if grant.Revoked || !time.Now().Before(grant.ExpiresAt) {
		return fosite.ErrInactiveToken
	}
	n, err := s.database(ctx).ProtocolSession.Update().Where(protocolsession.ID(signatureID("refresh", signature)), protocolsession.RequestID(requestID), protocolsession.Revoked(false), protocolsession.ExpiresAtGT(time.Now())).SetRevoked(true).Save(ctx)
	if err == nil && n != 1 {
		return fosite.ErrInactiveToken
	}
	return err
}
func (s *Store) RevokeRefreshToken(ctx context.Context, requestID string) error {
	if err := s.revokeGrant(ctx, requestID); err != nil {
		return err
	}
	_, err := s.database(ctx).ProtocolSession.Update().Where(protocolsession.RequestID(requestID), protocolsession.Kind("refresh")).SetRevoked(true).Save(ctx)
	return err
}

var _ oauth2.CoreStorage = (*Store)(nil)
