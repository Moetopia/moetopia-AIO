package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
)

func main() {
	if err := run(); err != nil {
		slog.Error("key initialization failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || os.Args[1] != "init" {
		return errors.New("usage: keys init")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	box, err := cryptobox.New(cfg.EncryptionKey)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := authorization.InitializeKey(ctx, db, box); err != nil {
		return err
	}
	slog.Info("signing key initialized; private key was not displayed")
	return nil
}
