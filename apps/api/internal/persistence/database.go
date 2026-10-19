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
