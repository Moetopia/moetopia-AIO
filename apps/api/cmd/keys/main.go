package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"time"

	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/securityreset"
)

func main() {
	if err := run(); err != nil {
		slog.Error("signing key command failed; verify configuration, schema and operation; no credential values were displayed")
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 || os.Args[1] != "init" && os.Args[1] != "rotate" && os.Args[1] != "emergency" {
		return errors.New("usage: keys init|rotate|emergency")
	}
	flags := flag.NewFlagSet("keys", flag.ContinueOnError)
	reason := flags.String("reason", "", "incident reason; never include credentials")
	ack := flags.Bool("acknowledge-global-revocation", false, "all API instances are stopped; invalidate all sessions, grants and transient proofs")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || os.Args[1] != "emergency" && (*ack || *reason != "") || os.Args[1] == "emergency" && (!*ack || *reason == "") {
		return errors.New("explicit maintenance acknowledgment and reason required for emergency rotation")
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
	if err := db.Ready(ctx); err != nil {
		return err
	}
	if os.Args[1] == "emergency" {
		if err := authorization.EmergencyRotateKey(ctx, db, box, *reason, func(ctx context.Context) error { return securityreset.Transient(ctx, db.Ent) }); err != nil {
			return err
		}
		slog.Info("emergency rotation committed; old public keys excluded and all transient identity credentials invalidated; restart only upgraded issuers")
		return nil
	}
	if os.Args[1] == "rotate" {
		if err := authorization.RotateKey(ctx, db, box); err != nil {
			return err
		}
		slog.Info("signing key rotated; old public key retained for overlap; private key was not displayed")
		return nil
	}
	if err := authorization.InitializeKey(ctx, db, box); err != nil {
		return err
	}
	slog.Info("signing key initialized; private key was not displayed")
	return nil
}
