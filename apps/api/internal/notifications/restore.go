package notifications

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
)

func DiscardRestoredDeliveries(ctx context.Context, db *ent.Client) error {
	_, err := persistence.Client(ctx, db).EmailDelivery.Delete().Exec(ctx)
	return err
}
