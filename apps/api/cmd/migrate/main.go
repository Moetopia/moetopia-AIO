package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pressly/goose/v3"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, db.SQL, migrations.Files)
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return err
	}
	slog.Info("migrations complete", "applied", len(results))
	return nil
}
