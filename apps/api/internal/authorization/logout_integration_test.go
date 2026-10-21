//go:build integration

package authorization

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
)

func testLogoutHints(t *testing.T, db *persistence.Database, box *cryptobox.Box, key *Key, encoded string) {
	f := newCodeFixture(t, db, box, key, encoded)
	code := f.approve(t, "openid profile")
	tokens := decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200)
	redirect := "https://client.moetopia.test/signed-out"
	if err := db.Ent.OAuthClient.UpdateOneID(f.clientID).SetPostLogoutRedirectUris([]string{redirect}).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	values := url.Values{"id_token_hint": {tokens.Identity}, "post_logout_redirect_uri": {redirect}, "state": {"opaque-state"}}
	id, data, err := f.provider.logoutParameters(t.Context(), values)
	if err != nil || id != f.clientID || data.HintSubject != f.user.ID || data.State != "opaque-state" {
		t.Fatalf("valid ID hint rejected: %v", err)
	}
	values.Set("client_id", "another-client")
	if _, _, err := f.provider.logoutParameters(t.Context(), values); !errors.Is(err, ErrLogout) {
		t.Fatal("ID hint audience did not bind client")
	}
	values.Set("client_id", f.clientID)
	values.Set("id_token_hint", tokens.Access)
	if _, _, err := f.provider.logoutParameters(t.Context(), values); !errors.Is(err, ErrLogout) {
		t.Fatal("access token was accepted as ID hint")
	}
	claims := jwt.RegisteredClaims{Issuer: f.provider.issuer, Subject: f.user.ID, Audience: jwt.ClaimStrings{f.clientID}, IssuedAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute * 30))}
	for _, scenario := range []string{"expired", "wrong-issuer", "wrong-subject", "wrong-key-id"} {
		changed := claims
		if scenario != "expired" {
			changed.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
		}
		if scenario == "wrong-issuer" {
			changed.Issuer = "https://attacker.test"
		}
		if scenario == "wrong-subject" {
			changed.Subject = ""
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, changed)
		token.Header["kid"] = key.ID
		if scenario == "wrong-key-id" {
			token.Header["kid"] = "unregistered-key"
		}
		raw, err := token.SignedString(key.Private)
		if err != nil {
			t.Fatal(err)
		}
		values.Set("id_token_hint", raw)
		if _, _, err := f.provider.logoutParameters(t.Context(), values); !errors.Is(err, ErrLogout) {
			t.Fatalf("invalid hint accepted: %s", scenario)
		}
	}
}
