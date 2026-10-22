package developer

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

func (s *Service) ManagementOwner(ctx context.Context, id string) (string, error) {
	row, err := persistence.Client(ctx, s.db).DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.Deleted(false)).Select(developerapplication.FieldOwnerID).Only(ctx)
	if ent.IsNotFound(err) {
		return "", ErrMissing
	}
	if err != nil {
		return "", err
	}
	return row.OwnerID, nil
}
