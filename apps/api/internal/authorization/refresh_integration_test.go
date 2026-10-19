//go:build integration

package authorization

import (
	"crypto/rand"
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

func testRefreshFlow(t *testing.T, db *persistence.Database, box *cryptobox.Box, key *Key, encoded string) {
	f := refreshFixture(t, db, box, key, encoded)
	other := refreshFixture(t, db, box, key, encoded)
	t.Run("rotates_and_preserves_identity", func(t *testing.T) {
		first := f.offline(t)
		second := decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
		if second.Refresh == "" || second.Refresh == first.Refresh || second.Access == first.Access {
			t.Fatal("credentials were not rotated")
		}
		parsed, err := jwt.Parse(second.Identity, func(*jwt.Token) (any, error) { return &key.Private.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(f.provider.issuer), jwt.WithAudience(f.clientID), jwt.WithSubject(f.user.ID), jwt.WithExpirationRequired())
		if err != nil || !parsed.Valid {
			t.Fatal("refreshed ID Token validation failed")
		}
		if parsed.Claims.(jwt.MapClaims)["email"] != f.user.Email || f.userInfo(second.Access).Code != 200 {
			t.Fatal("refreshed identity or UserInfo incorrect")
		}
		decodeTokens(t, f.refresh(first.Refresh, f.clientID), 400)
		decodeTokens(t, f.refresh(second.Refresh, f.clientID), 400)
		if f.userInfo(first.Access).Code != 401 || f.userInfo(second.Access).Code != 401 {
			t.Fatal("reuse did not revoke the whole token family")
		}
	})
	t.Run("foreign_and_tampered_credentials_do_not_revoke", func(t *testing.T) {
		first := f.offline(t)
		decodeTokens(t, f.refresh(first.Refresh, other.clientID), 400)
		second := decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
		decodeTokens(t, f.refresh(first.Refresh, other.clientID), 400)
		decodeTokens(t, f.refresh(tamperOpaque(first.Refresh), f.clientID), 400)
		if f.revoke(second.Refresh, other.clientID).Code != 200 {
			t.Fatal("foreign token revocation exposed ownership")
		}
		decodeTokens(t, f.refresh(second.Refresh, f.clientID), 200)
	})
	t.Run("concurrent_refresh_revokes_winning_family", func(t *testing.T) {
		first := f.offline(t)
		var wg sync.WaitGroup
		start := make(chan struct{})
		results := make(chan *httptest.ResponseRecorder, 2)
		for range 2 {
			wg.Go(func() { <-start; results <- f.refresh(first.Refresh, f.clientID) })
		}
		close(start)
		wg.Wait()
		close(results)
		var winner codeTokens
		successes := 0
		for result := range results {
			if result.Code == 200 {
				successes++
				winner = decodeTokens(t, result, 200)
			} else {
				decodeTokens(t, result, 400)
			}
		}
		if successes != 1 {
			t.Fatalf("concurrent refresh successes: %d", successes)
		}
		decodeTokens(t, f.refresh(winner.Refresh, f.clientID), 400)
		if f.userInfo(winner.Access).Code != 401 {
			t.Fatal("concurrent replay left an active access token")
		}
	})
	t.Run("revocation_unknown_and_spent_tokens", func(t *testing.T) {
		for _, kind := range []string{"access", "refresh", "spent"} {
			first := f.offline(t)
			current := first
			token := first.Access
			if kind != "access" {
				token = first.Refresh
			}
			if kind == "spent" {
				current = decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
			}
			for range 2 {
				response := f.revoke(token, f.clientID)
				if response.Code != 200 || response.Body.Len() != 0 {
					t.Fatal("revocation was not an empty idempotent response")
				}
			}
			decodeTokens(t, f.refresh(current.Refresh, f.clientID), 400)
			if f.userInfo(current.Access).Code != 401 {
				t.Fatal("revoked token family remained active")
			}
		}
		if f.revoke("unknown-credential", f.clientID).Code != 200 {
			t.Fatal("unknown token was distinguishable")
		}
		if f.revoke("unknown-credential", "unknown-client").Code != 401 {
			t.Fatal("revocation accepted an unknown client")
		}
	})
	t.Run("refresh_failure_rolls_back", func(t *testing.T) {
		first := f.offline(t)
		if _, err := db.SQL.ExecContext(t.Context(), `ALTER TABLE "authorization".protocol_sessions ADD CONSTRAINT test_block_refresh CHECK (kind <> 'refresh' OR revoked) NOT VALID`); err != nil {
			t.Fatal(err)
		}
		drop := func() {
			if _, err := db.SQL.ExecContext(t.Context(), `ALTER TABLE "authorization".protocol_sessions DROP CONSTRAINT IF EXISTS test_block_refresh`); err != nil {
				t.Error("could not remove dedicated test constraint")
			}
		}
		defer drop()
		if f.refresh(first.Refresh, f.clientID).Code < 500 {
			t.Fatal("storage failure was hidden")
		}
		drop()
		decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
	})
	t.Run("late_code_replay_and_cleanup", func(t *testing.T) {
		code := f.approve(t, "openid offline_access")
		first := decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
		second := decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
		signature := f.provider.store.strategy.AuthorizeCodeSignature(t.Context(), code)
		if _, err := db.Ent.ProtocolSession.UpdateOneID(signatureID("code", signature)).SetExpiresAt(time.Now().Add(-time.Hour)).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := f.provider.Cleanup(t.Context()); err != nil {
			t.Fatal(err)
		}
		decodeTokens(t, f.exchange(tamperOpaque(code), f.verifier, "https://client.moetopia.test/callback", f.clientID), 400)
		if f.userInfo(second.Access).Code != 200 {
			t.Fatal("forged code replay revoked a valid family")
		}
		decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 400)
		decodeTokens(t, f.refresh(second.Refresh, f.clientID), 400)
	})
	t.Run("confidential_refresh_and_basic_revocation", func(t *testing.T) {
		client := refreshFixture(t, db, box, key, encoded)
		first := client.offline(t)
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			t.Fatal(err)
		}
		secret := base64.RawURLEncoding.EncodeToString(secretBytes)
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Ent.OAuthClient.UpdateOneID(client.clientID).SetPublic(false).SetSecretHash(hash).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if client.refresh(first.Refresh, client.clientID).Code != 401 {
			t.Fatal("confidential refresh skipped client authentication")
		}
		form := url.Values{"grant_type": {"refresh_token"}, "client_id": {client.clientID}, "client_secret": {secret}, "refresh_token": {first.Refresh}}
		current := decodeTokens(t, client.form("/oauth/token", form), 200)
		for _, password := range []string{"incorrect", secret} {
			r := httptest.NewRequest("POST", "/oauth/revoke", strings.NewReader(url.Values{"token": {current.Refresh}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.SetBasicAuth(client.clientID, password)
			w := httptest.NewRecorder()
			client.provider.Revoke(w, r)
			if password == secret {
				if w.Code != 200 {
					t.Fatal("authenticated revocation failed")
				}
			} else if w.Code != 401 || w.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("invalid Basic client authentication was not challenged")
			}
		}
		if client.userInfo(current.Access).Code != 401 {
			t.Fatal("confidential revocation left active tokens")
		}
	})
	t.Run("scopes_audience_and_expiry", func(t *testing.T) {
		first := f.offline(t)
		for _, extra := range []url.Values{{"scope": {"openid admin.write"}}, {"scope": {"openid"}}, {"audience": {"https://outside.test"}}} {
			extra.Set("grant_type", "refresh_token")
			extra.Set("client_id", f.clientID)
			extra.Set("refresh_token", first.Refresh)
			if f.form("/oauth/token", extra).Code == 200 {
				t.Fatal("refresh changed approved privileges")
			}
		}
		signature := f.provider.store.strategy.RefreshTokenSignature(t.Context(), first.Refresh)
		request, err := f.provider.store.GetRefreshTokenSession(t.Context(), signature, newSession(key.ID))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Ent.AuthorizationGrant.UpdateOneID(request.GetID()).SetExpiresAt(time.Now().Add(90 * time.Second)).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		second := decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
		access, err := jwt.Parse(second.Access, func(*jwt.Token) (any, error) { return &key.Private.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}))
		if err != nil {
			t.Fatal("bounded access token signature invalid")
		}
		expires, err := access.Claims.GetExpirationTime()
		if err != nil || expires == nil || expires.Time.After(time.Now().Add(91*time.Second)) {
			t.Fatal("access token exceeded the absolute family lifetime")
		}
		request, err = f.provider.store.GetRefreshTokenSession(t.Context(), f.provider.store.strategy.RefreshTokenSignature(t.Context(), second.Refresh), newSession(key.ID))
		if err != nil {
			t.Fatal(err)
		}
		if request.GetSession().GetExpiresAt("refresh_token").After(time.Now().Add(91 * time.Second)) {
			t.Fatal("refresh extended the absolute family lifetime")
		}
		if _, err := db.Ent.ProtocolSession.Update().Where(protocolsession.ID(signatureID("refresh", f.provider.store.strategy.RefreshTokenSignature(t.Context(), second.Refresh)))).SetExpiresAt(time.Now().Add(-time.Second)).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		decodeTokens(t, f.refresh(second.Refresh, f.clientID), 400)
	})
}
