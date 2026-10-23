package developer

import (
	"context"
	"strings"
	"unicode/utf8"

	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

func (s *Service) AdminDelete(ctx context.Context, actor, id string, revision int, reason string) error {
	reason = strings.TrimSpace(reason)
	if actor == "" || reason == "" || !utf8.ValidString(reason) || utf8.RuneCountInString(reason) > 500 {
		return ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	row, err := unit.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.Deleted(false)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ErrMissing
	}
	if err != nil {
		return err
	}
	if row.Revision != revision {
		return ErrConflict
	}
	if err := s.protocol.SetManagedClientEnabled(ctx, row.ClientID, row.OwnerID, false); err != nil {
		return err
	}
	if err := unit.DeveloperApplication.UpdateOneID(id).SetDeleted(true).SetEnabled(false).SetSuspended(true).SetSuspensionReason(reason).AddRevision(1).Exec(ctx); err != nil {
		return err
	}
	if err := audit.RecordReason(ctx, unit.Client(), actor, "application.admin.deleted", id, reason); err != nil {
		return err
	}
	if err := s.notify(ctx, row.OwnerID, id, "application.deleted", reason); err != nil {
		return err
	}
	return unit.Commit()
}
