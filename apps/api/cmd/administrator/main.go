package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"time"

	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/relyingparty"
)

func main() {
	if err := run(); err != nil {
		slog.Error("administrator task failed; verify the operation, bootstrap state and configuration; no credentials were displayed")
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return errors.New("use init-client or bootstrap")
	}
	flags := flag.NewFlagSet("administrator", flag.ContinueOnError)
	email := flags.String("email", "", "existing verified account email")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.AdminOrigin == "" {
		return errors.New("MOETOPIA_ADMIN_ORIGIN is required")
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
	switch os.Args[1] {
	case "init-client":
		if *email != "" {
			return errors.New("email is only accepted for bootstrap")
		}
		if err := relyingparty.InitializeAdminClient(ctx, db.Ent, cfg.Issuer, cfg.AdminOrigin, cfg.EncryptionKey); err != nil {
			return err
		}
		slog.Info("administrator OIDC client initialized or verified; no credentials were displayed")
	case "bootstrap":
		users, err := identity.New(identity.NewStore(db.Ent, nil), cfg.EncryptionKey)
		if err != nil {
			return err
		}
		if err := administration.Bootstrap(ctx, db.Ent, users, *email); err != nil {
			return err
		}
		slog.Info("initial administrator assigned; OAuth login and MFA enrollment are required")
	default:
		return errors.New("unknown administrator task")
	}
	return nil
}
