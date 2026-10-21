package identity

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
)

func (s *Service) VerifiedEmailUser(ctx context.Context, email string) (User, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		return User{}, ErrMissing
	}
	row, err := persistence.Client(ctx, s.store.db).AccountUser.Query().Where(accountuser.Email(email), accountuser.Enabled(true), accountuser.DeletionDueAtIsNil()).Select(userColumns...).Only(ctx)
	if ent.IsNotFound(err) {
		return User{}, ErrMissing
	}
	if err != nil {
		return User{}, err
	}
	if row.EmailVerifiedAt.IsZero() {
		return User{}, ErrMissing
	}
	return public(row), nil
}
