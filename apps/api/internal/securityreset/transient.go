package securityreset

import (
	"context"
	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/sessions"
)

// The caller stops all issuers and owns the transaction before global invalidation.
func Transient(ctx context.Context, db *ent.Client) error {
	if err := administration.DiscardRestoredElevations(ctx, db); err != nil {
		return err
	}
	if err := sessions.DiscardRestoredSessions(ctx, db); err != nil {
		return err
	}
	if err := identity.DiscardRestoredChallenges(ctx, db); err != nil {
		return err
	}
	if err := authorization.DiscardRestoredCredentials(ctx, db); err != nil {
		return err
	}
	return notifications.DiscardRestoredDeliveries(ctx, db)
}
