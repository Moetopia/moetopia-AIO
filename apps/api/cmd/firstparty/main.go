package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/relyingparty"
)

func main() {
	if err := run(); err != nil {
		slog.Error("first-party client initialization failed; configuration and credentials were not displayed")
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ready(ctx); err != nil {
		return err
	}
	if err := relyingparty.InitializeClient(ctx, db.Ent, cfg.Issuer, cfg.EncryptionKey); err != nil {
		return err
	}
	slog.Info("user-site OIDC client initialized or verified; no credentials were displayed")
	return nil
}
