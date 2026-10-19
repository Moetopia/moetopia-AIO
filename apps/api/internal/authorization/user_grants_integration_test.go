//go:build integration

package authorization

import (
	"errors"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
	"testing"
)

func testUserGrants(t *testing.T, db *persistence.Database, box *cryptobox.Box, key *Key, encoded string) {
	f := refreshFixture(t, db, box, key, encoded)
	first := f.offline(t)
	items, err := f.provider.UserGrants(t.Context(), f.user.ID, 0, "")
	if err != nil || len(items.Items) != 1 || items.Items[0].ClientID != f.clientID {
		t.Fatal("user grant ownership missing")
	}
	grantID := items.Items[0].ID
	if err := f.provider.RevokeUserGrant(t.Context(), "other-user", grantID); !errors.Is(err, ErrGrantMissing) {
		t.Fatal("foreign grant accessible")
	}
	if f.userInfo(first.Access).Code != 200 {
		t.Fatal("foreign grant attempt revoked victim")
	}
	if _, err := db.Ent.AuthorizationGrant.Update().Where(authorizationgrant.ID(grantID)).SetUserID("").SetScopes([]string{}).Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	if count, err := BackfillGrantSubjects(t.Context(), NewStore(db.Ent, box)); err != nil || count != 1 {
		t.Fatal("legacy user grant backfill failed")
	}
	if count, err := BackfillGrantSubjects(t.Context(), NewStore(db.Ent, box)); err != nil || count != 0 {
		t.Fatal("backfill was not idempotent")
	}
	if err := f.provider.RevokeUserGrant(t.Context(), f.user.ID, grantID); err != nil {
		t.Fatal(err)
	}
	if f.userInfo(first.Access).Code != 401 {
		t.Fatal("user grant revocation left access active")
	}
	decodeTokens(t, f.refresh(first.Refresh, f.clientID), 400)
	old := f.offline(t)
	code := f.approve(t, "openid")
	if err := db.Ent.AccountUser.UpdateOneID(f.user.ID).AddSecurityVersion(1).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.userInfo(old.Access).Code != 401 {
		t.Fatal("security change left old access active")
	}
	decodeTokens(t, f.refresh(old.Refresh, f.clientID), 400)
	decodeTokens(t, f.exchange(code, f.verifier, "https://client.moetopia.test/callback", f.clientID), 400)
	remaining, err := f.provider.UserGrants(t.Context(), f.user.ID, 1, "")
	if err != nil || len(remaining.Items) != 0 {
		t.Fatal("old security version remained listed as active")
	}
}
