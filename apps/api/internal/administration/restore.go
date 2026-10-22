package administration

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/adminprincipal"
)

func DiscardRestoredElevations(ctx context.Context, db *ent.Client) error {
	db = persistence.Client(ctx, db)
	if _, err := db.AdminElevation.Delete().Exec(ctx); err != nil {
		return err
	}
	if err := db.AdminPrincipal.Update().Where(adminprincipal.EnrollmentSessionNEQ("")).SetEnrollmentBlocked(true).Exec(ctx); err != nil {
		return err
	}
	return db.AdminPrincipal.Update().ClearPendingSecret().ClearPendingExpiresAt().SetPendingSession("").SetEnrollmentSession("").SetEmergencyHash("").ClearEmergencyExpiresAt().AddFactorVersion(1).Exec(ctx)
}
