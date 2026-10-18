//go:build integration

package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
)

type codeFixture struct {
	provider      *Provider
	db            *persistence.Database
	key           *Key
	user          identity.User
	clientID      string
	verifier      string
	authenticated time.Time
}

func newCodeFixture(t *testing.T, db *persistence.Database, box *cryptobox.Box, key *Key, encoded string) *codeFixture {
	t.Helper()
	id := uuid.NewString()
	row, err := db.Ent.AccountUser.Create().SetID(id).SetEmail(id + "@example.test").SetDisplayName("Code user").SetPasswordHash("unusable-test-credential").SetEmailVerifiedAt(time.Now()).Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	clientID := uuid.NewString()
	_, err = db.Ent.OAuthClient.Create().SetID(clientID).SetName("Consent test app").SetEnabled(true).SetPublic(true).SetRedirectUris([]string{"https://client.moetopia.test/callback"}).SetGrantTypes([]string{"authorization_code"}).SetResponseTypes([]string{"code"}).SetScopes([]string{"openid", "profile", "email", "demo.read"}).SetAudiences([]string{"https://api.moetopia.test"}).Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	users, err := identity.New(identity.NewStore(db.Ent, nil), encoded)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(NewStore(db.Ent, box), key, "https://account.moetopia.test", encoded, users)
	if err != nil {
		t.Fatal(err)
	}
	return &codeFixture{provider, db, key, identity.User{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName}, clientID, strings.Repeat("v", 43), time.Now().Add(-time.Minute)}
}
func (f *codeFixture) query(scopes string) url.Values {
	sum := sha256.Sum256([]byte(f.verifier))
	return url.Values{"client_id": {f.clientID}, "response_type": {"code"}, "redirect_uri": {"https://client.moetopia.test/callback"}, "scope": {scopes}, "state": {"test-state-value"}, "nonce": {"test-nonce-value"}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}}
}
func (f *codeFixture) begin(t *testing.T, query url.Values) (string, string) {
	t.Helper()
	w := httptest.NewRecorder()
	f.provider.Authorize(w, httptest.NewRequest("GET", "/oauth/authorize?"+query.Encode(), nil))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("authorize HTTP %d", w.Code)
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].Domain != "" {
		t.Fatal("invalid transaction cookie")
	}
	return location.Query().Get("authorization_request"), cookies[0].Value
}
func (f *codeFixture) approve(t *testing.T, scopes string) string {
	t.Helper()
	id, cookie := f.begin(t, f.query(scopes))
	location, err := f.provider.Decide(t.Context(), id, cookie, f.user, f.authenticated, true)
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	u, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("state") != "test-state-value" || u.Query().Get("code") == "" {
		t.Fatal("invalid authorization response")
	}
	return u.Query().Get("code")
}
func (f *codeFixture) exchange(code, verifier, redirect, client string) *httptest.ResponseRecorder {
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}}
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	f.provider.Token(w, r)
	return w
}

type codeTokens struct {
	Access   string `json:"access_token"`
	Identity string `json:"id_token"`
	Refresh  string `json:"refresh_token"`
	Error    string `json:"error"`
}

func decodeTokens(t *testing.T, w *httptest.ResponseRecorder, status int) codeTokens {
	t.Helper()
	var result codeTokens
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != status {
		t.Fatalf("token HTTP %d expected %d; protocol error %s", w.Code, status, result.Error)
	}
	return result
}
func (f *codeFixture) userInfo(token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.provider.UserInfo(w, r)
	return w
}
