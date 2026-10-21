//go:build integration

package authorization

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/pressly/goose/v3"
	"github.com/valkey-io/valkey-go"
	"golang.org/x/crypto/bcrypt"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/migrations"
)

func TestPersistentClientCredentialsAndRestart(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasPrefix(u.Path, "/moetopia_test_") {
		t.Fatal("TEST_DATABASE_DSN must explicitly select a dedicated moetopia_test_ database")
	}
	address := os.Getenv("TEST_VALKEY_ADDRESS")
	if address == "" {
		t.Fatal("TEST_VALKEY_ADDRESS is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := persistence.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	provider, err := goose.NewProvider(goose.DialectPostgres, db.SQL, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 3); err != nil {
		t.Fatal(err)
	}
	legacyExpiry := time.Now().Add(3 * time.Minute).UTC().Truncate(time.Microsecond)
	_, err = db.Ent.ProtocolSession.Create().SetID("migration-legacy-code").SetKind("code").SetRequestID("migration-legacy-grant").SetClientID("migration-legacy-client").SetPayload([]byte("synthetic-migration-fixture")).SetExpiresAt(legacyExpiry).SetRevoked(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	legacyGrant, err := db.Ent.AuthorizationGrant.Get(ctx, "migration-legacy-grant")
	if err != nil || !legacyGrant.ExpiresAt.Equal(legacyExpiry.Add(10*time.Minute)) {
		t.Fatal("migration did not preserve the legacy grant expiry")
	}
	legacyCode, err := db.Ent.ProtocolSession.Get(ctx, "migration-legacy-code")
	if err != nil || !legacyCode.Revoked {
		t.Fatal("migration reactivated a consumed authorization code")
	}
	if err := db.Ent.ProtocolSession.DeleteOneID("migration-legacy-code").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Ent.AuthorizationGrant.DeleteOneID("migration-legacy-grant").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	cache, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{address}, Password: os.Getenv("TEST_VALKEY_PASSWORD"), DisableCache: true})
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	if err := cache.Do(ctx, cache.B().Ping().Build()).Error(); err != nil {
		t.Fatal(err)
	}
	root := make([]byte, 32)
	if _, err := rand.Read(root); err != nil {
		t.Fatal(err)
	}
	encoded := base64.StdEncoding.EncodeToString(root)
	t.Run("administrator bootstrap", func(t *testing.T) { testAdministratorBootstrap(t, db, encoded) })
	box, err := cryptobox.New(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := InitializeKey(ctx, db, box); err != nil {
		t.Fatal(err)
	}
	key, err := LoadActiveKey(ctx, db, box)
	if err != nil {
		t.Fatal(err)
	}
	if err := InitializeKey(ctx, db, box); err == nil {
		t.Fatal("initialization silently rotated an active key")
	}
	secret := base64.RawURLEncoding.EncodeToString(root)
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Ent.OAuthClient.Create().SetID("test-client").SetName("Test client").SetEnabled(true).SetSecretHash(hash).SetGrantTypes([]string{"client_credentials"}).SetScopes([]string{"demo.read"}).SetAudiences([]string{"https://api.moetopia.test"}).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db.Ent, box)
	op, err := NewProvider(store, key, "https://account.moetopia.test", encoded, nil)
	if err != nil {
		t.Fatal(err)
	}
	exchange := func(scope, audience, password string) *httptest.ResponseRecorder {
		form := url.Values{"grant_type": {"client_credentials"}, "scope": {scope}, "audience": {audience}}
		r := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.SetBasicAuth("test-client", password)
		w := httptest.NewRecorder()
		op.Token(w, r)
		return w
	}
	w := exchange("demo.read", "https://api.moetopia.test", secret)
	if w.Code != 200 {
		t.Fatalf("token exchange failed: HTTP %d", w.Code)
	}
	var response struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.RefreshToken != "" || response.ExpiresIn < 599 || response.ExpiresIn > 600 {
		t.Fatalf("client_credentials returned an unexpected credential or lifespan (%d seconds)", response.ExpiresIn)
	}
	parsed, err := jwt.Parse(response.AccessToken, func(*jwt.Token) (any, error) { return &key.Private.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer("https://account.moetopia.test"), jwt.WithAudience("https://api.moetopia.test"), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid || parsed.Header["typ"] != "at+jwt" || parsed.Header["kid"] != key.ID {
		t.Fatal("invalid JWT profile")
	}
	expires, expiryErr := parsed.Claims.GetExpirationTime()
	issued, issuedErr := parsed.Claims.GetIssuedAt()
	if expiryErr != nil || issuedErr != nil || expires == nil || issued == nil || expires.Sub(issued.Time) != 10*time.Minute {
		t.Fatal("JWT lifetime must be ten minutes")
	}
	signature := strings.Split(response.AccessToken, ".")[2]
	if _, err := store.GetAccessTokenSession(ctx, signature, &oauth2.JWTSession{}); err != nil {
		t.Fatal(err)
	}
	reopened, err := persistence.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reloaded, err := LoadActiveKey(ctx, reopened, box)
	if err != nil || reloaded.ID != key.ID {
		t.Fatal("signing key did not survive reopening")
	}
	reloadedStore := NewStore(reopened.Ent, box)
	if _, err := reloadedStore.GetAccessTokenSession(ctx, signature, &oauth2.JWTSession{}); err != nil {
		t.Fatal("protocol state did not survive reopening")
	}
	if err := reloadedStore.DeleteAccessTokenSession(ctx, signature); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAccessTokenSession(ctx, signature, &oauth2.JWTSession{}); err == nil {
		t.Fatal("revoked protocol state remained active")
	}
	for _, invalid := range []struct{ scope, audience, password string }{{"admin.write", "https://api.moetopia.test", secret}, {"demo.read", "https://other.moetopia.test", secret}, {"demo.read", "https://api.moetopia.test", "incorrect"}} {
		if exchange(invalid.scope, invalid.audience, invalid.password).Code == 200 {
			t.Fatal("invalid authorization request succeeded")
		}
	}
	t.Run("authorization_code_oidc", func(t *testing.T) { testCodeFlow(t, db, box, key, encoded) })
	t.Run("token_introspection", func(t *testing.T) { testIntrospection(t, db, box, key, encoded) })
	t.Run("refresh_rotation_and_revocation", func(t *testing.T) { testRefreshFlow(t, db, box, key, encoded) })
	t.Run("user_grants_and_security_versions", func(t *testing.T) { testUserGrants(t, db, box, key, encoded) })
	t.Run("logout_id_token_hints", func(t *testing.T) { testLogoutHints(t, db, box, key, encoded) })
	t.Run("signing_key_rotation", func(t *testing.T) { testSigningKeyRotation(t, db, box, key, encoded) })
}
