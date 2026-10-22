package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
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
	target := flags.String("user-id", "", "existing administrator UUID")
	revision := flags.Int("expected-revision", -1, "current principal revision")
	reason := flags.String("reason", "", "operator recovery reason; never include credentials")
	output := flags.String("output-file", "", "new absolute private file for the one-time recovery credential")
	acknowledge := flags.Bool("acknowledge-mfa-reset", false, "revoke administrator sessions and replace all MFA recovery material")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if os.Args[1] != "emergency-reset" && (*revision != -1 || *reason != "" || *output != "" || *acknowledge) {
		return errors.New("recovery arguments are only accepted by emergency-reset")
	}
	if os.Args[1] != "inspect" && os.Args[1] != "emergency-reset" && *target != "" {
		return errors.New("user-id is only accepted by inspect and emergency-reset")
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
	case "inspect":
		if *email != "" || *target == "" {
			return errors.New("inspect requires only user-id")
		}
		service := administration.New(db.Ent, nil, nil, nil, nil)
		principal, err := service.PrincipalStatus(ctx, *target)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(principal)
	case "emergency-reset":
		if *email != "" || !*acknowledge || *target == "" || *revision < 0 || *reason == "" || !filepath.IsAbs(*output) {
			return errors.New("explicit target, revision, reason, acknowledgment and private output file required")
		}
		file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return errors.New("cannot create private recovery output file")
		}
		defer file.Close()
		box, err := cryptobox.New(cfg.EncryptionKey)
		if err != nil {
			return err
		}
		users, err := identity.New(identity.NewStore(db.Ent, nil), cfg.EncryptionKey)
		if err != nil {
			return err
		}
		web, err := sessions.NewAdmin(db.Ent, cfg.EncryptionKey, cfg.AdminOrigin, cfg.Environment == "production")
		if err != nil {
			return err
		}
		service := administration.New(db.Ent, box, users, notifications.New(db.Ent, box, nil), web)
		ticket, err := service.EmergencyReset(ctx, *target, *revision, *reason)
		if err != nil {
			return err
		}
		if err := json.NewEncoder(file).Encode(ticket); err != nil {
			return errors.New("recovery state changed but writing the private file failed; reissue with the new revision")
		}
		if err := file.Sync(); err != nil {
			return errors.New("recovery state changed but syncing the private file failed; verify storage")
		}
		slog.Info("administrator recovery prepared; use the private output file within fifteen minutes, then enroll MFA; no credential values were displayed")
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
