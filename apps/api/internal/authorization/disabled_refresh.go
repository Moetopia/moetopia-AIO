package authorization

import (
	"context"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
)

func (*Store) CreateRefreshTokenSession(context.Context, string, string, fosite.Requester) error {
	return fosite.ErrUnsupportedGrantType
}
func (*Store) GetRefreshTokenSession(context.Context, string, fosite.Session) (fosite.Requester, error) {
	return nil, fosite.ErrNotFound
}
func (*Store) DeleteRefreshTokenSession(context.Context, string) error { return fosite.ErrNotFound }
func (*Store) RotateRefreshToken(context.Context, string, string) error {
	return fosite.ErrUnsupportedGrantType
}
func (*Store) RevokeRefreshToken(context.Context, string) error { return nil }

var _ oauth2.CoreStorage = (*Store)(nil)
