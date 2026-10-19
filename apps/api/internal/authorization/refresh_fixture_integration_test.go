//go:build integration

package authorization

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
)

func refreshFixture(t *testing.T, db *persistence.Database, box *cryptobox.Box, key *Key, encoded string) *codeFixture {
	t.Helper()
	f := newCodeFixture(t, db, box, key, encoded)
	_, err := db.Ent.OAuthClient.UpdateOneID(f.clientID).SetGrantTypes([]string{"authorization_code", "refresh_token"}).SetScopes([]string{"openid", "profile", "email", "demo.read", "offline_access"}).Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *codeFixture) offline(t *testing.T) codeTokens {
	t.Helper()
	code := f.approve(t, "openid profile email offline_access")
	tokens := decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
	if tokens.Refresh == "" {
		t.Fatal("offline grant did not return a refresh token")
	}
	return tokens
}
func (f *codeFixture) form(path string, form url.Values) *httptest.ResponseRecorder {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := httptest.NewRequest("POST", path, strings.NewReader(form.Encode())).WithContext(ctx)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	if path == "/oauth/token" {
		f.provider.Token(w, r)
	} else {
		f.provider.Revoke(w, r)
	}
	return w
}
func (f *codeFixture) refresh(token, client string) *httptest.ResponseRecorder {
	return f.form("/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client}, "refresh_token": {token}})
}
func (f *codeFixture) revoke(token, client string) *httptest.ResponseRecorder {
	return f.form("/oauth/revoke", url.Values{"token": {token}, "client_id": {client}})
}
func tamperOpaque(raw string) string {
	parts := strings.Split(raw, ".")
	index := len(parts[0]) - 2
	last := parts[0][index : index+1]
	if last == "A" {
		last = "B"
	} else {
		last = "A"
	}
	parts[0] = parts[0][:index] + last + parts[0][index+1:]
	return strings.Join(parts, ".")
}
