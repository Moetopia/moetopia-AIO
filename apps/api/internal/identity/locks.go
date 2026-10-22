package identity

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"slices"
)

func (s *Service) LockUsers(ctx context.Context, ids []string) error {
	ordered := slices.Clone(ids)
	slices.Sort(ordered)
	ordered = slices.Compact(ordered)
	rows, err := persistence.Client(ctx, s.store.db).AccountUser.Query().Where(accountuser.IDIn(ordered...)).Select(accountuser.FieldID).Order(ent.Asc(accountuser.FieldID)).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(rows) != len(ordered) {
		return ErrMissing
	}
	return nil
}
