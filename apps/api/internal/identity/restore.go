package identity

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/invitation"
)

func DiscardRestoredChallenges(ctx context.Context, db *ent.Client) error {
	db = persistence.Client(ctx, db)
	if _, err := db.AccountChallenge.Delete().Exec(ctx); err != nil {
		return err
	}
	if _, err := db.EmailVerification.Delete().Exec(ctx); err != nil {
		return err
	}
	if err := db.Invitation.Update().Where(invitation.Consumed(false)).SetRevoked(true).Exec(ctx); err != nil {
		return err
	}
	_, err := db.ExternalLogin.Delete().Exec(ctx)
	return err
}
