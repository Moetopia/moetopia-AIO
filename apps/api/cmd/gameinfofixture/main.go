//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/migrations"
)

func random() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func main() {
	if run() != nil {
		fmt.Fprintln(os.Stderr, "isolated AIO fixture failed; credentials were not displayed")
		os.Exit(1)
	}
}
func run() error {
	dsn, callback, manifest := os.Getenv("MGI_AIO_DATABASE_URL"), os.Getenv("MGI_AIO_CALLBACK"), os.Getenv("MGI_AIO_MANIFEST")
	dbURL, e1 := url.Parse(dsn)
	redirect, e2 := url.Parse(callback)
	if os.Getenv("MGI_AIO_FIXTURE") != "1" || e1 != nil || e2 != nil || !regexp.MustCompile(`^/mgi_test_aio_[a-f0-9]{16}$`).MatchString(dbURL.Path) || (dbURL.Hostname() != "127.0.0.1" && dbURL.Hostname() != "localhost") || redirect.Scheme != "http" || redirect.Hostname() != "127.0.0.1" || redirect.Port() == "" || redirect.Path != "/oauth/callback" || redirect.User != nil || redirect.RawQuery != "" || redirect.Fragment != "" || manifest == "" {
		return errors.New("explicit isolated database and callback required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := persistence.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	var existing int
	if err = db.SQL.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema')").Scan(&existing); err != nil || existing != 0 {
		return errors.New("fixture database must be empty")
	}
	migrator, err := goose.NewProvider(goose.DialectPostgres, db.SQL, migrations.Files)
	if err != nil {
		return err
	}
	if _, err = migrator.Up(ctx); err != nil {
		return err
	}
	root := make([]byte, 32)
	_, _ = rand.Read(root)
	encoded := base64.StdEncoding.EncodeToString(root)
	box, err := cryptobox.New(encoded)
	if err != nil {
		return err
	}
	if err = authorization.InitializeKey(ctx, db, box); err != nil {
		return err
	}
	userID, clientID, secret, control := uuid.NewString(), uuid.NewString(), random(), random()
	if err = db.Ent.AccountUser.Create().SetID(userID).SetEmail("fixture-" + userID + "@example.test").SetDisplayName("Game Info fixture").SetPasswordHash("unusable-fixture-credential").SetEmailVerifiedAt(time.Now()).Exec(ctx); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err = db.Ent.OAuthClient.Create().SetID(clientID).SetName("Isolated Game Info RP").SetEnabled(true).SetPublic(false).SetSecretHash(hash).SetRedirectUris([]string{callback}).SetGrantTypes([]string{"authorization_code"}).SetResponseTypes([]string{"code"}).SetScopes([]string{"openid"}).SetAudiences([]string{"game-info-fixture"}).Exec(ctx); err != nil {
		return err
	}
	users, err := identity.New(identity.NewStore(db.Ent, nil), encoded)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	issuer := "http://" + listener.Addr().String()
	provider, err := authorization.NewPersistentProvider(authorization.NewStore(db.Ent, box), issuer, encoded, users)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(provider.Metadata()) })
	mux.HandleFunc("GET /oauth/jwks", provider.WriteJWKS)
	mux.HandleFunc("POST /oauth/token", provider.Token)
	mux.HandleFunc("GET /oauth/authorize", provider.Authorize)
	page := template.Must(template.New("consent").Parse(`<h1>Isolated AIO consent fixture</h1><form method="post" action="/fixture/consent"><input type="hidden" name="authorization_request" value="{{.}}"><button name="decision" value="approve">Approve</button><button name="decision" value="deny">Deny</button></form>`))
	mux.HandleFunc("GET /auth/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, r.URL.Query().Get("authorization_request"))
	})
	mux.HandleFunc("POST /fixture/consent", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != issuer || r.ParseForm() != nil || (r.Form.Get("decision") != "approve" && r.Form.Get("decision") != "deny") {
			w.WriteHeader(403)
			return
		}
		location, err := provider.Decide(r.Context(), r.Form.Get("authorization_request"), provider.BrowserToken(r.Header.Get("Cookie")), identity.User{ID: userID}, time.Now(), r.Form.Get("decision") == "approve")
		if err != nil {
			w.WriteHeader(400)
			return
		}
		http.Redirect(w, r, location, http.StatusSeeOther)
	})
	stop := make(chan struct{}, 1)
	mux.HandleFunc("POST /fixture/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Fixture-Control") != control {
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(204)
		select {
		case stop <- struct{}{}:
		default:
		}
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	failures := make(chan error, 1)
	go func() { failures <- server.Serve(listener) }()
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	content, err := json.Marshal(map[string]string{"issuer": issuer, "clientId": clientID, "clientSecret": secret, "control": control})
	if err != nil {
		return err
	}
	if err = os.WriteFile(manifest, content, 0600); err != nil {
		return err
	}
	select {
	case <-stop:
		return nil
	case <-time.After(3 * time.Minute):
		return errors.New("fixture timed out")
	case err := <-failures:
		return err
	}
}
