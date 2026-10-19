package developer

import (
	"context"
	"strings"
	"unicode/utf8"

	"entgo.io/ent/dialect/sql"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

func (s *Service) AdminList(ctx context.Context, pending bool, cursor string) (Applications, error) {
	query := persistence.Client(ctx, s.db).DeveloperApplication.Query().Where(developerapplication.Deleted(false))
	if pending {
		query.Where(func(selector *sql.Selector) {
			versions := sql.Table("application_versions").Schema("developer")
			selector.Where(sql.Exists(sql.Select(versions.C("id")).From(versions).Where(sql.And(sql.ColumnsEQ(versions.C("application_id"), selector.C("id")), sql.ColumnsEQ(versions.C("version"), selector.C("latest_version")), sql.EQ(versions.C("status"), "pending")))))
		})
	}
	return s.list(ctx, query, cursor)
}
func (s *Service) AdminDetail(ctx context.Context, id string) (Application, error) {
	db := persistence.Client(ctx, s.db)
	row, err := db.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.Deleted(false)).Only(ctx)
	if ent.IsNotFound(err) {
		return Application{}, ErrMissing
	}
	if err != nil {
		return Application{}, err
	}
	return s.view(ctx, db, row)
}
func (s *Service) Suspend(ctx context.Context, actor, id string, revision int, suspended bool, reason string) (Application, error) {
	reason = strings.TrimSpace(reason)
	if actor == "" || !utf8.ValidString(reason) || utf8.RuneCountInString(reason) > 500 || suspended && reason == "" {
		return Application{}, ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return Application{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	row, err := unit.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.Deleted(false)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return Application{}, ErrMissing
	}
	if err != nil {
		return Application{}, err
	}
	if row.Revision != revision {
		return Application{}, ErrConflict
	}
	if err := s.protocol.SetManagedClientEnabled(ctx, row.ClientID, row.OwnerID, row.Enabled && !suspended); err != nil {
		return Application{}, err
	}
	if !suspended {
		reason = ""
	}
	if err := unit.DeveloperApplication.UpdateOneID(id).SetSuspended(suspended).SetSuspensionReason(reason).AddRevision(1).Exec(ctx); err != nil {
		return Application{}, err
	}
	row.Suspended, row.SuspensionReason, row.Revision = suspended, reason, row.Revision+1
	if err := audit.RecordTarget(ctx, unit.Client(), actor, "application.suspension.changed", id, "success"); err != nil {
		return Application{}, err
	}
	result, err := s.view(ctx, unit.Client(), row)
	if err != nil {
		return Application{}, err
	}
	return result, unit.Commit()
}
