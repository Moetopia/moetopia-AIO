package identity

import (
	"context"

	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
)

// Security notices must remain deliverable to disabled accounts.
func (s *Service) VerifiedEmail(ctx context.Context, id string) (string, error) {
	row, err := persistence.Client(ctx, s.store.db).AccountUser.Query().Where(accountuser.ID(id)).Select(accountuser.FieldEmail, accountuser.FieldEmailVerifiedAt).Only(ctx)
	if ent.IsNotFound(err) {
		return "", ErrMissing
	}
	if err != nil {
		return "", err
	}
	if row.EmailVerifiedAt.IsZero() {
		return "", ErrMissing
	}
	return row.Email, nil
}
