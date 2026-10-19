package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/persistence"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	callback := flag.String("redirect-uri", "", "Exact loopback callback URI for the development client")
	offline := flag.Bool("offline-access", false, "Allow explicitly consented offline access with refresh token rotation")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	dsn, err := url.Parse(cfg.DatabaseURL)
	if err != nil || cfg.Environment != "development" || dsn.Path != "/moetopia_dev" || dsn.Port() != "55432" {
		return errors.New("development client creation requires moetopia_dev on port 55432")
	}
	u, err := url.Parse(*callback)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Port() == "" || !(u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return errors.New("redirect-uri must be an explicit HTTP loopback callback with a port, without credentials, query or fragment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ready(ctx); err != nil {
		return err
	}
	id := "dev-" + uuid.NewString()
	grants := []string{"authorization_code"}
	scopes := []string{"openid", "profile", "email", "demo.read"}
	if *offline {
		grants = append(grants, "refresh_token")
		scopes = append(scopes, "offline_access")
	}
	_, err = db.Ent.OAuthClient.Create().SetID(id).SetName("Local OIDC integration").SetPublic(true).SetEnabled(true).SetRedirectUris([]string{*callback}).SetGrantTypes(grants).SetResponseTypes([]string{"code"}).SetScopes(scopes).SetAudiences([]string{cfg.Issuer + "/api"}).Save(ctx)
	if err != nil {
		return errors.New("development client could not be created")
	}
	fmt.Println("client_id:", id)
	fmt.Println("redirect_uri:", *callback)
	fmt.Println("issuer:", cfg.Issuer)
	fmt.Println("audience:", cfg.Issuer+"/api")
	fmt.Println("Public development client created without a client secret; use authorization_code with PKCE S256.")
	return nil
}
