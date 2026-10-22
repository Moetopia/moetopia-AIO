package restoration

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/securityreset"
	"net/url"
	"regexp"
)

func SafeTarget(dsn string) bool {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" || !regexp.MustCompile(`^/moetopia_restore_[a-f0-9]{16}$`).MatchString(u.Path) {
		return false
	}
	parsed, err := pgx.ParseConfig(dsn)
	return err == nil && u.Path == "/"+parsed.Database
}

func Check(ctx context.Context, dsn, key string, invalidate bool) error {
	if !SafeTarget(dsn) {
		return errors.New("restore checks require a new dedicated moetopia_restore_<16 hex> database")
	}
	db, err := persistence.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ready(ctx); err != nil {
		return err
	}
	box, err := cryptobox.New(key)
	if err != nil {
		return err
	}
	unit, err := persistence.Begin(ctx, db.Ent)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := authorization.ValidateRestoredKeys(ctx, unit.Client(), box); err != nil {
		return err
	}
	if !invalidate {
		return nil
	}
	if err := securityreset.Transient(ctx, unit.Client()); err != nil {
		return err
	}
	u, _ := url.Parse(dsn)
	if err := audit.RecordTarget(ctx, unit.Client(), "system:restore", "restore.credentials_invalidated", u.Path[1:], "success"); err != nil {
		return err
	}
	return unit.Commit()
}
