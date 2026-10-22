//go:build integration

package authorization

import (
	"context"
	"errors"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
	"testing"
)

func testEmergencySigningKeyRotation(t *testing.T, db *persistence.Database, box *cryptobox.Box, initial *Key, encoded string) {
	f := refreshFixture(t, db, box, initial, encoded)
	provider, err := NewPersistentProvider(NewStore(db.Ent, box), f.provider.issuer, encoded, f.provider.users)
	if err != nil {
		t.Fatal(err)
	}
	f.provider = provider
	before, err := LoadActiveKey(t.Context(), db, box)
	if err != nil {
		t.Fatal(err)
	}
	first := f.offline(t)
	pending := f.approve(t, "openid profile")
	failAudit := true
	db.Ent.SecurityEvent.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if failAudit {
				return nil, errors.New("injected emergency audit failure")
			}
			return next.Mutate(ctx, m)
		})
	})
	reset := func(ctx context.Context) error { return DiscardRestoredCredentials(ctx, db.Ent) }
	if err := EmergencyRotateKey(t.Context(), db, box, "Private signing key incident", reset); err == nil {
		t.Fatal("emergency rotation ignored failed audit")
	}
	failAudit = false
	unchanged, err := LoadActiveKey(t.Context(), db, box)
	if err != nil || unchanged.ID != before.ID || f.userInfo(first.Access).Code != 200 {
		t.Fatal("failed emergency rotation changed key or protocol state")
	}
	if err := EmergencyRotateKey(t.Context(), db, box, "Private signing key incident", reset); err != nil {
		t.Fatal(err)
	}
	after, err := LoadActiveKey(t.Context(), db, box)
	if err != nil || after.ID == before.ID {
		t.Fatal("emergency rotation did not create a new active key")
	}
	keys, err := provider.JWKS(t.Context())
	if err != nil || len(keys.(map[string]any)["keys"].([]map[string]string)) != 1 {
		t.Fatal("emergency rotation retained old public overlap")
	}
	if f.userInfo(first.Access).Code != 401 || f.refresh(first.Refresh, f.clientID).Code != 400 || f.exchange(pending, f.verifier, "https://client.moetopia.test/callback", f.clientID).Code != 400 {
		t.Fatal("emergency rotation retained old protocol credentials")
	}
	rows, err := db.Ent.SigningKey.Query().Where(signingkey.IDNEQ(after.ID)).All(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Active || len(row.PrivateKey) != 0 || row.RetireAfter == nil {
			t.Fatal("old private material survived emergency rotation")
		}
	}
	fresh := f.offline(t)
	if f.userInfo(fresh.Access).Code != 200 {
		t.Fatal("fresh consent failed after emergency rotation")
	}
}
