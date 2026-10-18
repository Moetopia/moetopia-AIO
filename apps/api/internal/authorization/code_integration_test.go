//go:build integration

package authorization

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationinteraction"
)

func testCodeFlow(t *testing.T, db *persistence.Database, box *cryptobox.Box, key *Key, encoded string) {
	f := newCodeFixture(t, db, box, key, encoded)
	t.Run("signed_tokens_scope_and_replay", func(t *testing.T) {
		code := f.approve(t, "openid profile email demo.read")
		tokens := decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
		if tokens.Refresh != "" || tokens.Identity == "" || tokens.Access == "" {
			t.Fatal("incorrect token set")
		}
		for _, item := range []struct{ raw, audience, typ string }{{tokens.Access, "https://api.moetopia.test", "at+jwt"}, {tokens.Identity, f.clientID, "JWT"}} {
			parsed, err := jwt.Parse(item.raw, func(*jwt.Token) (any, error) { return &key.Private.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer("https://account.moetopia.test"), jwt.WithAudience(item.audience), jwt.WithSubject(f.user.ID), jwt.WithExpirationRequired())
			if err != nil || !parsed.Valid || parsed.Header["typ"] != item.typ || parsed.Header["kid"] != key.ID {
				t.Fatal("invalid independent JWT verification")
			}
			claims := parsed.Claims.(jwt.MapClaims)
			if item.typ == "JWT" && (claims["nonce"] != "test-nonce-value" || claims["email"] != f.user.Email || claims["name"] != f.user.DisplayName || claims["at_hash"] == nil || claims["auth_time"] == nil) {
				t.Fatal("missing identity claims")
			}
			if item.typ == "JWT" {
				sum := sha256.Sum256([]byte(tokens.Access))
				if claims["at_hash"] != base64.RawURLEncoding.EncodeToString(sum[:16]) {
					t.Fatal("incorrect OIDC access token hash")
				}
			}
			if item.typ == "at+jwt" && (claims["email"] != nil || claims["name"] != nil) {
				t.Fatal("access token disclosed identity claims")
			}
		}
		if f.userInfo(tokens.Access).Code != 200 || f.userInfo(tokens.Identity).Code != 401 {
			t.Fatal("UserInfo accepted wrong token profile")
		}
		decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 400)
		if f.userInfo(tokens.Access).Code != 401 {
			t.Fatal("replayed code did not revoke existing access token")
		}
	})
	t.Run("failed_validation_preserves_code", func(t *testing.T) {
		code := f.approve(t, "openid")
		for _, bad := range []struct{ verifier, redirect, client string }{{strings.Repeat("x", 43), "https://client.moetopia.test/callback", f.clientID}, {f.verifier, "https://client.moetopia.test/wrong", f.clientID}, {f.verifier, "https://client.moetopia.test/callback", "unknown-client"}} {
			if f.exchange(code, bad.verifier, bad.redirect, bad.client).Code == 200 {
				t.Fatal("invalid exchange succeeded")
			}
		}
		tokens := decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
		parsed, err := jwt.Parse(tokens.Identity, func(*jwt.Token) (any, error) { return &key.Private.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}))
		if err != nil {
			t.Fatal("identity signature failed")
		}
		claims := parsed.Claims.(jwt.MapClaims)
		if claims["email"] != nil || claims["name"] != nil {
			t.Fatal("unrequested identity claims disclosed")
		}
	})
	t.Run("issuance_failure_rolls_back_consumption", func(t *testing.T) {
		code := f.approve(t, "openid")
		if _, err := db.SQL.ExecContext(t.Context(), `ALTER TABLE "authorization".protocol_sessions ADD CONSTRAINT test_block_access CHECK (kind <> 'access') NOT VALID`); err != nil {
			t.Fatal(err)
		}
		drop := func() {
			if _, err := db.SQL.ExecContext(t.Context(), `ALTER TABLE "authorization".protocol_sessions DROP CONSTRAINT IF EXISTS test_block_access`); err != nil {
				t.Error("could not remove dedicated test constraint")
			}
		}
		defer drop()
		if response := f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID); response.Code < 500 {
			t.Fatalf("simulated issuance failure HTTP %d", response.Code)
		}
		drop()
		decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
	})
	t.Run("one_database_connection", func(t *testing.T) {
		db.SQL.SetMaxOpenConns(1)
		defer db.SQL.SetMaxOpenConns(8)
		code := f.approve(t, "openid")
		decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
	})
	t.Run("oauth_only_and_disabled_client", func(t *testing.T) {
		code := f.approve(t, "demo.read")
		tokens := decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
		if tokens.Identity != "" || f.userInfo(tokens.Access).Code != 401 {
			t.Fatal("OAuth-only grant obtained identity privileges")
		}
		code = f.approve(t, "openid")
		if _, err := db.Ent.OAuthClient.UpdateOneID(f.clientID).SetEnabled(false).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID).Code == 200 {
			t.Fatal("disabled client obtained tokens")
		}
		if _, err := db.Ent.OAuthClient.UpdateOneID(f.clientID).SetEnabled(true).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("concurrent_redemption", func(t *testing.T) {
		code := f.approve(t, "openid")
		var wg sync.WaitGroup
		results := make(chan *httptest.ResponseRecorder, 2)
		start := make(chan struct{})
		for range 2 {
			wg.Go(func() {
				<-start
				results <- f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID)
			})
		}
		close(start)
		wg.Wait()
		close(results)
		successes := 0
		for result := range results {
			if result.Code == 200 {
				successes++
			} else if result.Code != 400 {
				t.Fatalf("unexpected concurrency HTTP %d", result.Code)
			}
		}
		if successes != 1 {
			t.Fatalf("concurrent redemption successes: %d", successes)
		}
	})
	t.Run("interaction_binding_deny_expiry", func(t *testing.T) {
		id, cookie := f.begin(t, f.query("openid"))
		if _, err := f.provider.Details(t.Context(), id, strings.Repeat("a", 43), f.authenticated); !errors.Is(err, ErrInteraction) {
			t.Fatal("different browser accepted")
		}
		details, err := f.provider.Details(t.Context(), id, cookie, f.authenticated)
		if err != nil || details.ClientName != "Consent test app" || details.LoginRequired {
			t.Fatal("incorrect authorization preview")
		}
		location, err := f.provider.Decide(t.Context(), id, cookie, f.user, f.authenticated, false)
		if err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(location)
		if u.Query().Get("error") != "access_denied" || u.Query().Get("state") != "test-state-value" || u.Query().Get("code") != "" {
			t.Fatal("incorrect denial callback")
		}
		if _, err := f.provider.Decide(t.Context(), id, cookie, f.user, f.authenticated, true); !errors.Is(err, ErrInteraction) {
			t.Fatal("consumed interaction reused")
		}
		id, cookie = f.begin(t, f.query("openid"))
		_, err = db.Ent.AuthorizationInteraction.Update().Where(authorizationinteraction.ID(id)).SetExpiresAt(time.Now().Add(-time.Second)).Save(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.provider.Details(t.Context(), id, cookie, f.authenticated); !errors.Is(err, ErrInteraction) {
			t.Fatal("expired interaction accepted")
		}
	})
	t.Run("unsafe_requests", func(t *testing.T) {
		for _, change := range []func(url.Values){func(q url.Values) { q.Set("code_challenge_method", "plain") }, func(q url.Values) { q.Del("nonce") }, func(q url.Values) { q.Set("scope", "openid offline_access") }, func(q url.Values) { q.Set("request_uri", "https://attacker.test/request") }} {
			q := f.query("openid")
			change(q)
			if _, err := f.provider.authorizeRequest(t.Context(), q); err == nil {
				t.Fatal("unsafe request accepted")
			}
		}
		q := f.query("openid")
		q.Set("redirect_uri", "https://attacker.test/callback")
		w := httptest.NewRecorder()
		f.provider.Authorize(w, httptest.NewRequest("GET", "/oauth/authorize?"+q.Encode(), nil))
		if w.Header().Get("Location") != "" || w.Code == 200 {
			t.Fatal("unsafe callback redirected")
		}
	})
	t.Run("force_login_and_disabled_identity", func(t *testing.T) {
		q := f.query("openid")
		q.Set("prompt", "login")
		id, cookie := f.begin(t, q)
		if _, err := f.provider.Decide(t.Context(), id, cookie, f.user, f.authenticated, true); !errors.Is(err, ErrLoginRequired) {
			t.Fatal("prompt=login accepted stale session")
		}
		if _, err := f.provider.Decide(t.Context(), id, cookie, f.user, time.Now(), true); err != nil {
			t.Fatal(err)
		}
		q = f.query("openid")
		q.Set("max_age", "0")
		id, cookie = f.begin(t, q)
		if _, err := f.provider.Decide(t.Context(), id, cookie, f.user, f.authenticated, true); !errors.Is(err, ErrLoginRequired) {
			t.Fatal("max_age=0 accepted stale session")
		}
		if _, err := f.provider.Decide(t.Context(), id, cookie, f.user, time.Now(), true); err != nil {
			t.Fatal(err)
		}
		code := f.approve(t, "openid")
		if _, err := db.Ent.AccountUser.UpdateOneID(f.user.ID).SetEnabled(false).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID).Code == 200 {
			t.Fatal("disabled identity obtained tokens")
		}
	})
}
