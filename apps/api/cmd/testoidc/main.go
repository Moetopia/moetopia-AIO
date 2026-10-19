package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/password"
	"moetopia.local/aio/api/internal/persistence"
)

func validate(environment, dsn, name, callback string) error {
	u, err := url.Parse(dsn)
	redirect, redirectErr := url.Parse(callback)
	if environment != "test" || err != nil || !regexp.MustCompile(`^moetopia_test_[a-f0-9]{16}$`).MatchString(name) || u.Path != "/"+name || u.Port() != "55432" || redirectErr != nil || callback != "http://localhost:54310/callback" || redirect.User != nil {
		return errors.New("OIDC fixture requires the explicitly provisioned test database and loopback callback")
	}
	return nil
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	callback := os.Getenv("MOETOPIA_TEST_OIDC_CALLBACK")
	if err := validate(cfg.Environment, cfg.DatabaseURL, os.Getenv("MOETOPIA_TEST_DATABASE_NAME"), callback); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	service, err := identity.New(identity.NewStore(db.Ent, nil), cfg.EncryptionKey)
	if err != nil {
		return err
	}
	invitation, err := service.IssueInvitation(ctx, time.Now().Add(time.Hour))
	if err != nil {
		return err
	}
	id := uuid.NewString()
	_, err = db.Ent.OAuthClient.Create().SetID(id).SetName("Independent OIDC test RP").SetEnabled(true).SetPublic(true).SetRedirectUris([]string{callback}).SetGrantTypes([]string{"authorization_code", "refresh_token"}).SetResponseTypes([]string{"code"}).SetScopes([]string{"openid", "profile", "email", "demo.read", "offline_access"}).SetAudiences([]string{cfg.Issuer + "/api"}).Save(ctx)
	if err != nil {
		return err
	}
	if os.Getenv("MOETOPIA_TEST_OIDC_SUITE") == "admin" {
		email := os.Getenv("MOETOPIA_TEST_OIDC_EMAIL")
		value := os.Getenv("MOETOPIA_TEST_OIDC_PASSWORD")
		if email == "" || !password.Valid(value) {
			return errors.New("isolated administrator fixture credentials are required")
		}
		hash, err := password.Hash(ctx, value)
		if err != nil {
			return err
		}
		if err := db.Ent.AccountUser.Create().SetID(uuid.NewString()).SetEmail(email).SetDisplayName("OIDC administrator fixture").SetPasswordHash(hash).SetEmailVerifiedAt(time.Now()).Exec(ctx); err != nil {
			return err
		}
		if err := administration.Bootstrap(ctx, db.Ent, service, email); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"client_id": id, "invitation": invitation})
}
func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString("OIDC test fixture failed; no configuration or credential values were displayed\n")
		os.Exit(1)
	}
}
