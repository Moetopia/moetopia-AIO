package developer

import (
	"context"
	"errors"

	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/applicationversion"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

var ErrPublished = errors.New("published applications must be disabled before account deletion")

func (s *Service) DisableOwner(ctx context.Context, owner string) error {
	if active, err := s.HasPublishedApplications(ctx, owner); err != nil {
		return err
	} else if active {
		return ErrPublished
	}
	db := persistence.Client(ctx, s.db)
	rows, err := db.DeveloperApplication.Query().Where(developerapplication.OwnerID(owner), developerapplication.Deleted(false)).Order(ent.Asc(developerapplication.FieldID)).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.protocol.SetManagedClientEnabled(ctx, row.ClientID, owner, false); err != nil {
			return err
		}
		if row.Enabled {
			if err := db.DeveloperApplication.UpdateOneID(row.ID).SetEnabled(false).AddRevision(1).Exec(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) PurgeOwner(ctx context.Context, owner string) error {
	db := persistence.Client(ctx, s.db)
	for {
		ids, err := db.DeveloperApplication.Query().Where(developerapplication.OwnerID(owner)).Order(ent.Asc(developerapplication.FieldID)).Limit(50).ForUpdate().IDs(ctx)
		if err != nil || len(ids) == 0 {
			return err
		}
		if _, err := db.ApplicationVersion.Delete().Where(applicationversion.ApplicationIDIn(ids...)).Exec(ctx); err != nil {
			return err
		}
		if _, err := db.DeveloperApplication.Delete().Where(developerapplication.IDIn(ids...)).Exec(ctx); err != nil {
			return err
		}
	}
}
