package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"moetopia.local/aio/api/internal/persistence/ent"
)

type Database struct {
	SQL *sql.DB
	Ent *ent.Client
}

func Open(ctx context.Context, dsn string) (*Database, error) {
	configuration, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("invalid MOETOPIA_DATABASE_URL")
	}
	pool := stdlib.OpenDB(*configuration)
	pool.SetMaxOpenConns(8)
	pool.SetMaxIdleConns(2)
	pool.SetConnMaxLifetime(30 * time.Minute)
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, errors.New("PostgreSQL connection failed; check server and credentials")
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, pool)))
	return &Database{SQL: pool, Ent: client}, nil
}

func (d *Database) Ready(ctx context.Context) error {
	var statistics bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT to_regclass('authorization.protocol_events') IS NOT NULL AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='authorization' AND table_name='clients' AND column_name='statistics_since')").Scan(&statistics); err != nil {
		return err
	}
	if !statistics {
		return errors.New("database migration required")
	}
	var testSubjects bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT to_regclass('authorization.client_test_subjects') IS NOT NULL").Scan(&testSubjects); err != nil {
		return err
	}
	if !testSubjects {
		return errors.New("database migration required")
	}
	var logoutTables bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT to_regclass('authorization.logout_interactions') IS NOT NULL AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='authorization' AND table_name='clients' AND column_name='post_logout_redirect_uris')").Scan(&logoutTables); err != nil {
		return err
	}
	if !logoutTables {
		return errors.New("database migration required")
	}
	var externalTables bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT to_regclass('identity.external_identities') IS NOT NULL AND to_regclass('identity.external_logins') IS NOT NULL AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='identity' AND table_name='users' AND column_name='has_password')").Scan(&externalTables); err != nil {
		return err
	}
	if !externalTables {
		return errors.New("database migration required")
	}
	var lifecycleColumns bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='identity' AND table_name='users' AND column_name='deletion_due_at') AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='notifications' AND table_name='email_deliveries' AND column_name='user_id')").Scan(&lifecycleColumns); err != nil {
		return err
	}
	if !lifecycleColumns {
		return errors.New("database migration required")
	}
	var administratorTables bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT to_regclass('administration.principals') IS NOT NULL AND to_regclass('administration.elevations') IS NOT NULL AND to_regclass('administration.bootstrap') IS NOT NULL AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='identity' AND table_name='users' AND column_name='disabled_reason') AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='identity' AND table_name='registration_policy' AND column_name='revision') AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='identity' AND table_name='invitations' AND column_name='revoked')").Scan(&administratorTables); err != nil {
		return err
	}
	if !administratorTables {
		return errors.New("database migration required")
	}
	var developerTables bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT to_regclass('developer.applications') IS NOT NULL AND to_regclass('developer.application_versions') IS NOT NULL").Scan(&developerTables); err != nil {
		return err
	}
	if !developerTables {
		return errors.New("database migration required")
	}
	var exists bool
	if err := d.SQL.QueryRowContext(ctx, "SELECT to_regclass('identity.account_challenges') IS NOT NULL AND to_regclass('audit.security_events') IS NOT NULL AND to_regclass('sessions.rp_logins') IS NOT NULL AND to_regclass('authorization.grants') IS NOT NULL AND to_regclass('authorization.interactions') IS NOT NULL AND to_regclass('authorization.protocol_sessions') IS NOT NULL AND to_regclass('authorization.signing_keys') IS NOT NULL AND to_regclass('authorization.clients') IS NOT NULL AND to_regclass('identity.users') IS NOT NULL AND to_regclass('identity.registration_policy') IS NOT NULL AND to_regclass('identity.email_verifications') IS NOT NULL AND to_regclass('identity.invitations') IS NOT NULL AND to_regclass('sessions.web_sessions') IS NOT NULL AND to_regclass('notifications.email_deliveries') IS NOT NULL").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("database migration required")
	}
	return nil
}

func (d *Database) Close() error { return d.Ent.Close() }
