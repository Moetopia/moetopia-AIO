package sessions

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
)

func DiscardRestoredSessions(ctx context.Context, db *ent.Client) error {
	db = persistence.Client(ctx, db)
	if _, err := db.WebSession.Delete().Exec(ctx); err != nil {
		return err
	}
	_, err := db.RPLogin.Delete().Exec(ctx)
	return err
}
