//go:build integration

package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
)

func (f *codeFixture) basicForm(path, client, secret string, form url.Values) *httptest.ResponseRecorder {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := httptest.NewRequest("POST", path, strings.NewReader(form.Encode())).WithContext(ctx)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(url.QueryEscape(client), url.QueryEscape(secret))
	w := httptest.NewRecorder()
	if path == "/oauth/introspect" {
		f.provider.Introspect(w, r)
	} else {
		f.provider.Token(w, r)
	}
	return w
}

func confidentialFixture(t *testing.T, f *codeFixture) string {
	t.Helper()
	secret, hash, err := NewClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.OAuthClient.UpdateOneID(f.clientID).SetPublic(false).SetSecretHash(hash).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	return secret
}

func introspectionResult(t *testing.T, w *httptest.ResponseRecorder, active bool) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || result["active"] != active || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected introspection status %d, expected active=%t", w.Code, active)
	}
	if !active && len(result) != 1 {
		t.Fatal("inactive introspection disclosed token metadata")
	}
	for _, name := range []string{"email", "name", "security_version", "id_token_claims", "access_token", "refresh_token"} {
		if _, exists := result[name]; exists {
			t.Fatal("introspection disclosed private state")
		}
	}
	return result
}

type unavailableSubject struct{}

func (unavailableSubject) User(context.Context, string) (identity.User, error) {
	return identity.User{}, errors.New("synthetic identity outage")
}

func testIntrospection(t *testing.T, db *persistence.Database, box *cryptobox.Box, key *Key, encoded string) {
	t.Run("metadata_ownership_and_rotation", func(t *testing.T) {
		f := refreshFixture(t, db, box, key, encoded)
		first := f.offline(t)
		second := decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
		secret := confidentialFixture(t, f)
		check := func(token, hint string, active bool) map[string]any {
			return introspectionResult(t, f.basicForm("/oauth/introspect", f.clientID, secret, url.Values{"token": {token}, "token_type_hint": {hint}}), active)
		}
		access := check(second.Access, "refresh_token", true)
		refresh := check(second.Refresh, "access_token", true)
		check(second.Access, "unknown_hint", true)
		if access["client_id"] != f.clientID || access["sub"] != f.user.ID || access["iss"] != f.provider.issuer || access["token_type"] != "bearer" || refresh["exp"].(float64) <= access["exp"].(float64)+3600 {
			t.Fatal("introspection claims or refresh expiration incorrect")
		}
		for _, token := range []string{first.Refresh, "unknown-token", tamperOpaque(second.Refresh), second.Identity} {
			check(token, "", false)
		}
		other := refreshFixture(t, db, box, key, encoded)
		foreign := other.offline(t)
		check(foreign.Access, "", false)
		check(foreign.Refresh, "", false)
		if other.userInfo(foreign.Access).Code != 200 || f.userInfo(second.Access).Code != 200 {
			t.Fatal("read-only introspection revoked a token family")
		}
		signature := f.provider.store.strategy.AccessTokenSignature(t.Context(), second.Access)
		stored, err := f.provider.store.GetAccessTokenSession(t.Context(), signature, newSession(key.ID))
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Ent.AuthorizationGrant.UpdateOneID(stored.GetID()).SetUserID("").Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := db.Ent.AuthorizationGrant.UpdateOneID(stored.GetID()).SetUserID(f.user.ID).Exec(context.Background()); err != nil {
				t.Error("could not restore legacy grant fixture")
			}
		})
		if err := db.Ent.AccountUser.UpdateOneID(f.user.ID).AddSecurityVersion(1).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		check(second.Access, "", false)
		check(second.Refresh, "", false)
	})
	t.Run("authentication_required", func(t *testing.T) {
		f := refreshFixture(t, db, box, key, encoded)
		tokens := f.offline(t)
		form := url.Values{"token": {tokens.Access}}
		if w := f.basicForm("/oauth/introspect", f.clientID, "", form); w.Code != 401 || w.Header().Get("WWW-Authenticate") == "" {
			t.Fatal("public client admitted to introspection")
		}
		secret := confidentialFixture(t, f)
		for _, credentials := range [][2]string{{f.clientID, "wrong"}, {"unknown", secret}} {
			if w := f.basicForm("/oauth/introspect", credentials[0], credentials[1], form); w.Code != 401 || w.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("invalid client authentication accepted")
			}
		}
		for _, header := range []string{"", "Bearer " + tokens.Access} {
			r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Authorization", header)
			w := httptest.NewRecorder()
			f.provider.Introspect(w, r)
			if w.Code != 401 {
				t.Fatal("anonymous or bearer introspection accepted")
			}
		}
		for _, extra := range []string{"access_token", "client_id", "client_secret"} {
			form.Set(extra, tokens.Access)
			if f.basicForm("/oauth/introspect", f.clientID, secret, form).Code != 401 {
				t.Fatal("mixed authentication accepted")
			}
			form.Del(extra)
		}
		for _, invalid := range []url.Values{{}, {"token": {""}}, {"token": {tokens.Access, tokens.Access}}, {"token": {tokens.Access}, "token_type_hint": {"access_token", "refresh_token"}}} {
			if f.basicForm("/oauth/introspect", f.clientID, secret, invalid).Code != 400 {
				t.Fatal("malformed introspection accepted")
			}
		}
	})
	t.Run("revocation_expiry_and_failures", func(t *testing.T) {
		f := refreshFixture(t, db, box, key, encoded)
		first := f.offline(t)
		second := f.offline(t)
		if f.revoke(first.Access, f.clientID).Code != 200 {
			t.Fatal("revocation failed")
		}
		secret := confidentialFixture(t, f)
		check := func(token string) *httptest.ResponseRecorder {
			return f.basicForm("/oauth/introspect", f.clientID, secret, url.Values{"token": {token}})
		}
		introspectionResult(t, check(first.Access), false)
		introspectionResult(t, check(first.Refresh), false)
		sig := f.provider.store.strategy.AccessTokenSignature(t.Context(), second.Access)
		row, err := db.Ent.ProtocolSession.Get(t.Context(), signatureID("access", sig))
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Ent.ProtocolSession.UpdateOneID(row.ID).SetPayload([]byte("corrupted-test-state")).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		if check(second.Access).Code != 503 {
			t.Fatal("protocol storage corruption was hidden as inactivity")
		}
		if err := db.Ent.ProtocolSession.UpdateOneID(row.ID).SetPayload(row.Payload).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		f.provider.users = unavailableSubject{}
		if check(second.Access).Code != 503 {
			t.Fatal("identity outage was hidden as inactivity")
		}
		f.provider.users, err = identity.New(identity.NewStore(db.Ent, nil), encoded)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Ent.ProtocolSession.UpdateOneID(row.ID).SetExpiresAt(time.Now().Add(-time.Second)).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		introspectionResult(t, check(second.Access), false)
		if err := db.Ent.AccountUser.UpdateOneID(f.user.ID).SetEnabled(false).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		introspectionResult(t, check(second.Refresh), false)
		if err := db.Ent.OAuthClient.UpdateOneID(f.clientID).SetEnabled(false).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		if check(second.Refresh).Code != 401 {
			t.Fatal("disabled client can introspect")
		}
	})
	t.Run("machine_credentials_and_one_connection", func(t *testing.T) {
		f := refreshFixture(t, db, box, key, encoded)
		secret := confidentialFixture(t, f)
		if err := db.Ent.OAuthClient.UpdateOneID(f.clientID).SetGrantTypes([]string{"client_credentials"}).SetScopes([]string{"demo.read"}).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		tokens := decodeTokens(t, f.basicForm("/oauth/token", f.clientID, secret, url.Values{"grant_type": {"client_credentials"}, "scope": {"demo.read"}}), 200)
		f.provider.users = unavailableSubject{}
		db.SQL.SetMaxOpenConns(1)
		defer db.SQL.SetMaxOpenConns(10)
		result := introspectionResult(t, f.basicForm("/oauth/introspect", f.clientID, secret, url.Values{"token": {tokens.Access}}), true)
		if result["sub"] != f.clientID || result["scope"] != "demo.read" {
			t.Fatal("machine token metadata incorrect")
		}
	})
}
