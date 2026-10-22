package administration

import (
	"context"
	"errors"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
)

func Bootstrap(ctx context.Context, db *ent.Client, users *identity.Service, email string) error {
	unit, err := persistence.Begin(ctx, db)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := lockAdministration(ctx, unit.Client()); err != nil {
		return err
	}
	user, err := users.VerifiedEmailUser(ctx, email)
	if err != nil {
		return errors.New("bootstrap requires an existing enabled account with a verified email")
	}
	if err := users.VerifyVersion(ctx, user.ID, user.SecurityVersion); err != nil {
		return err
	}
	if err := unit.AdminBootstrap.Create().SetID("initial-administrator").SetUserID(user.ID).Exec(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return errors.New("administrator bootstrap has already been consumed")
		}
		return err
	}
	if count, err := unit.AdminPrincipal.Query().Count(ctx); err != nil {
		return err
	} else if count != 0 {
		return errors.New("administrator principals already exist")
	}
	if err := unit.AdminPrincipal.Create().SetID(user.ID).Exec(ctx); err != nil {
		return err
	}
	if err := audit.RecordTarget(ctx, unit.Client(), user.ID, "admin.bootstrap", user.ID, "success"); err != nil {
		return err
	}
	return unit.Commit()
}
