package main

import (
	"context"
	"log/slog"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("grant metadata backfill failed; credentials were not displayed")
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	box, err := cryptobox.New(cfg.EncryptionKey)
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
	if err := db.Ready(ctx); err != nil {
		return err
	}
	count, err := authorization.BackfillGrantSubjects(ctx, authorization.NewStore(db.Ent, box))
	if err != nil {
		return err
	}
	slog.Info("authorization subject metadata backfilled", "updated", count)
	return nil
}
