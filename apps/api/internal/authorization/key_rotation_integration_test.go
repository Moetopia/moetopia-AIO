//go:build integration

package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
)

func testSigningKeyRotation(t *testing.T, db *persistence.Database, box *cryptobox.Box, initial *Key, encoded string) {
	f := refreshFixture(t, db, box, initial, encoded)
	provider, err := NewPersistentProvider(NewStore(db.Ent, box), f.provider.issuer, encoded, f.provider.users)
	if err != nil {
		t.Fatal(err)
	}
	f.provider = provider
	first := f.offline(t)
	pendingCode := f.approve(t, "openid profile")
	if err := RotateKey(t.Context(), db, box); err != nil {
		t.Fatal(err)
	}
	current, err := LoadActiveKey(t.Context(), db, box)
	if err != nil || current.ID == initial.ID {
		t.Fatal("rotation did not activate a new key")
	}
	old, err := db.Ent.SigningKey.Get(t.Context(), initial.ID)
	if err != nil || old.Active || old.RetireAfter == nil || old.RetireAfter.Before(time.Now().Add(14*time.Minute)) || len(old.PrivateKey) == 0 {
		t.Fatal("old key lost overlap")
	}
	keys, err := provider.JWKS(t.Context())
	if err != nil || len(keys.(map[string]any)["keys"].([]map[string]string)) != 2 {
		t.Fatal("JWKS does not contain both public keys")
	}
	if f.userInfo(first.Access).Code != 200 {
		t.Fatal("rotation invalidated unexpired old access token")
	}
	second := decodeTokens(t, f.refresh(first.Refresh, f.clientID), 200)
	for _, raw := range []string{second.Access, second.Identity, decodeTokens(t, f.exchange(pendingCode, f.verifier, "https://client.moetopia.test/callback", f.clientID), 200).Identity} {
		token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
			if token.Header["kid"] != current.ID {
				return nil, errors.New("stale signing key header")
			}
			return &current.Private.PublicKey, nil
		}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(provider.issuer), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			t.Fatal("new token was not consistently signed by active key")
		}
	}
	if f.userInfo(second.Access).Code != 200 {
		t.Fatal("new access token rejected by original running provider")
	}
	secret := confidentialFixture(t, f)
	introspectionResult(t, f.basicForm("/oauth/introspect", f.clientID, secret, url.Values{"token": {second.Access}}), true)
	restarted, err := NewPersistentProvider(NewStore(db.Ent, box), provider.issuer, encoded, provider.users)
	if err != nil {
		t.Fatal(err)
	}
	f.provider = restarted
	if f.userInfo(second.Access).Code != 200 {
		t.Fatal("rotated key did not survive restart")
	}
	failAudit := true
	db.Ent.SecurityEvent.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			if failAudit {
				return nil, errors.New("injected key audit failure")
			}
			return next.Mutate(ctx, mutation)
		})
	})
	if err := RotateKey(t.Context(), db, box); err == nil {
		t.Fatal("rotation committed without audit")
	}
	failAudit = false
	unchanged, err := LoadActiveKey(t.Context(), db, box)
	if err != nil || unchanged.ID != current.ID {
		t.Fatal("failed rotation changed active key")
	}
	if count, err := db.Ent.SigningKey.Query().Count(t.Context()); err != nil || count != 2 {
		t.Fatal("failed rotation left key material")
	}
	if exists, err := db.Ent.SecurityEvent.Query().Where(securityevent.Action("signing_key.rotated"), securityevent.TargetID(current.ID)).Exist(t.Context()); err != nil || !exists {
		t.Fatal("rotation audit missing")
	}
	if err := db.Ent.SigningKey.UpdateOneID(initial.ID).SetRetireAfter(time.Now().Add(-time.Minute)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := provider.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	old, err = db.Ent.SigningKey.Get(t.Context(), initial.ID)
	if err != nil || len(old.PrivateKey) != 0 {
		t.Fatal("retired private material retained")
	}
	keys, err = provider.JWKS(t.Context())
	if err != nil || len(keys.(map[string]any)["keys"].([]map[string]string)) != 1 {
		t.Fatal("expired key still published")
	}
	if _, err := provider.keys.Public(t.Context(), initial.ID); err == nil {
		t.Fatal("expired key still accepted")
	}
	db.SQL.SetMaxOpenConns(1)
	t.Cleanup(func() { db.SQL.SetMaxOpenConns(8) })
	if f.userInfo(second.Access).Code != 200 {
		t.Fatal("single-connection verification failed")
	}
	r := httptest.NewRecorder()
	provider.WriteJWKS(r, httptest.NewRequest("GET", "/oauth/jwks", nil))
	var public map[string]any
	if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &public) != nil || r.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("public key endpoint failed")
	}
	if count, err := db.Ent.SigningKey.Query().Where(signingkey.Active(true)).Count(t.Context()); err != nil || count != 1 {
		t.Fatal("active key uniqueness lost")
	}
	activeRow, err := db.Ent.SigningKey.Get(t.Context(), current.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ent.SigningKey.UpdateOneID(current.ID).SetPrivateKey([]byte("invalid-encrypted-key")).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	decodeTokens(t, f.basicForm("/oauth/token", f.clientID, secret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {second.Refresh}}), 503)
	if err := db.Ent.SigningKey.UpdateOneID(current.ID).SetPrivateKey(activeRow.PrivateKey).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	decodeTokens(t, f.basicForm("/oauth/token", f.clientID, secret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {second.Refresh}}), 200)
}
