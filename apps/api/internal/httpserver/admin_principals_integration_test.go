//go:build integration

package httpserver

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence/ent/adminprincipal"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
)

func isolateAdministratorSet(t *testing.T, f *adminFixture) {
	t.Helper()
	// Fixtures share one disposable database and keep rows between sequential tests.
	previous, err := f.db.Ent.AdminPrincipal.Query().Where(adminprincipal.IDNEQ(f.userID)).IDs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	active, err := f.db.Ent.AdminPrincipal.Query().Where(adminprincipal.IDIn(previous...), adminprincipal.Enabled(true)).IDs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.AdminPrincipal.Update().Where(adminprincipal.IDIn(active...)).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		if err := f.db.Ent.AdminPrincipal.Update().Where(adminprincipal.IDNotIn(previous...)).SetEnabled(false).Exec(ctx); err != nil {
			t.Error("could not disable current test administrators")
		}
		if err := f.db.Ent.AdminPrincipal.Update().Where(adminprincipal.IDIn(active...)).SetEnabled(true).Exec(ctx); err != nil {
			t.Error("could not restore previous test administrators")
		}
	})
}

func adminActor(t *testing.T, f *adminFixture, cookie string) (identity.User, string) {
	t.Helper()
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	return user, f.rp.CookieToken(cookie)
}

func secondAdministrator(t *testing.T, f *adminFixture, ownerCookie string) (*adminFixture, string, string) {
	t.Helper()
	member, target := memberFixture(t, f.portalFixture)
	row, err := f.db.Ent.AccountUser.Get(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(target.ID).SetPasswordHash(row.PasswordHash).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	owner, token := adminActor(t, f, ownerCookie)
	web, err := f.admin.Sessions.Find(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.ChangePrincipal(t.Context(), owner, web, target.ID, 0, true, initialPassword, "Add another administrator"); err != nil {
		t.Fatal(err)
	}
	other := &adminFixture{member, f.admin, f.rp}
	cookie, csrf := other.login(t)
	cookie, csrf, _, _ = other.enroll(t, cookie, csrf)
	return other, cookie, csrf
}

func TestAdministratorPrincipalLifecycleAndLastAdministrator(t *testing.T) {
	f := newAdminFixture(t)
	isolateAdministratorSet(t, f)
	cookie, csrf := f.login(t)
	if f.request("GET", "/api/v1/admin/principals", cookie, "", "", nil).Code != 403 {
		t.Fatal("principal listing bypassed MFA")
	}
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	list := decodeAccount[administration.Principals](t, f.request("GET", "/api/v1/admin/principals", cookie, "", "", nil), 200)
	var current administration.Principal
	for _, item := range list.Items {
		if item.ID == f.userID {
			current = item
		}
	}
	if current.ID == "" || !current.Enrolled {
		t.Fatal("principal listing incorrect")
	}
	body := map[string]any{"revision": current.Revision, "enabled": false, "current_password": initialPassword, "reason": "Retire role"}
	path := "/api/v1/admin/principals/" + f.userID + "/availability"
	if f.request("POST", path, cookie, "", adminOrigin, body).Code != 403 || f.request("POST", path, cookie, csrf, "https://foreign.test", body).Code != 403 {
		t.Fatal("role mutation bypassed request protection")
	}
	if f.request("POST", path, cookie, csrf, adminOrigin, body).Code != 409 {
		t.Fatal("last administrator could revoke own role")
	}
	other, otherCookie, _ := secondAdministrator(t, f, cookie)
	otherRow, err := f.db.Ent.AdminPrincipal.Get(t.Context(), other.userID)
	if err != nil {
		t.Fatal(err)
	}
	owner, token := adminActor(t, f, cookie)
	web, err := f.admin.Sessions.Find(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.ChangePrincipal(t.Context(), owner, web, other.userID, otherRow.FactorVersion-1, false, initialPassword, "Retire role"); !errors.Is(err, administration.ErrState) {
		t.Fatal("stale principal revision accepted")
	}
	if _, err := f.admin.ChangePrincipal(t.Context(), owner, web, other.userID, otherRow.FactorVersion, false, "wrong", "Retire role"); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("principal change skipped password proof")
	}
	revoked, err := f.admin.ChangePrincipal(t.Context(), owner, web, other.userID, otherRow.FactorVersion, false, initialPassword, "Retire role")
	if err != nil || revoked.Enabled || revoked.Enrolled || revoked.Revision != otherRow.FactorVersion+1 {
		t.Fatal("principal revocation incomplete", err)
	}
	state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", otherCookie, "", "", nil), 200)
	if state.Authenticated {
		t.Fatal("revocation retained administrator session")
	}
	restored, err := f.admin.ChangePrincipal(t.Context(), owner, web, other.userID, revoked.Revision, true, initialPassword, "Restore role with fresh enrollment")
	if err != nil || !restored.Enabled || restored.Enrolled {
		t.Fatal("role restoration revived MFA", err)
	}
	if _, err := f.admin.ChangePrincipal(t.Context(), owner, web, owner.ID, current.Revision, false, initialPassword, "Try retiring before replacement enrollment"); !errors.Is(err, administration.ErrLastAdministrator) {
		t.Fatal("unenrolled replacement bypassed last administrator protection")
	}
	if _, err := f.db.Ent.SecurityEvent.Query().Where(securityevent.Action("admin.role.revoked"), securityevent.TargetID(other.userID), securityevent.Reason("Retire role")).Only(t.Context()); err != nil {
		t.Fatal("role reason audit missing")
	}
}

func TestAdministratorRoleRollbackAndConcurrentRevocation(t *testing.T) {
	f := newAdminFixture(t)
	isolateAdministratorSet(t, f)
	cookie, csrf := f.login(t)
	cookie, _, _, _ = f.enroll(t, cookie, csrf)
	other, otherCookie, _ := secondAdministrator(t, f, cookie)
	owner, token := adminActor(t, f, cookie)
	web, err := f.admin.Sessions.Find(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	row, err := f.db.Ent.AdminPrincipal.Get(t.Context(), other.userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE notifications.email_deliveries ADD CONSTRAINT test_role_notice_failure CHECK (user_id <> '`+other.userID+`') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE notifications.email_deliveries DROP CONSTRAINT IF EXISTS test_role_notice_failure`); err != nil {
			t.Error("could not remove role failure fixture")
		}
	}
	defer drop()
	if _, err := f.admin.ChangePrincipal(t.Context(), owner, web, other.userID, row.FactorVersion, false, initialPassword, "Rollback fixture"); err == nil {
		t.Fatal("notification failure committed revocation")
	}
	if state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", otherCookie, "", "", nil), 200); !state.Authenticated || !state.MFA.Verified {
		t.Fatal("failed revocation cleared MFA session")
	}
	drop()
	otherUser, otherToken := adminActor(t, other, otherCookie)
	otherWeb, err := f.admin.Sessions.Find(t.Context(), otherToken)
	if err != nil {
		t.Fatal(err)
	}
	ownerRow, err := f.db.Ent.AdminPrincipal.Get(t.Context(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	results := make(chan error, 2)
	group.Go(func() {
		_, err := f.admin.ChangePrincipal(t.Context(), owner, web, otherUser.ID, row.FactorVersion, false, initialPassword, "Concurrent retirement")
		results <- err
	})
	group.Go(func() {
		_, err := f.admin.ChangePrincipal(t.Context(), otherUser, otherWeb, owner.ID, ownerRow.FactorVersion, false, initialPassword, "Concurrent retirement")
		results <- err
	})
	group.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("concurrent revocation did not leave one active administrator")
	}
	if count, err := f.db.Ent.AdminPrincipal.Query().Where(adminprincipal.Enabled(true)).Count(t.Context()); err != nil || count != 1 {
		t.Fatal("last administrator invariant violated")
	}
}

func TestAdministratorEmergencyRecoveryAndBrowserBinding(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, _, _, _ = f.enroll(t, cookie, csrf)
	row, err := f.db.Ent.AdminPrincipal.Get(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	f.db.SQL.SetMaxOpenConns(1)
	ticket, err := f.admin.EmergencyReset(t.Context(), f.userID, row.FactorVersion, "Operator verified account ownership after complete factor loss")
	f.db.SQL.SetMaxOpenConns(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticket.Token) != 64 || ticket.Revision != row.FactorVersion+1 {
		t.Fatal("emergency ticket invalid")
	}
	if state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", cookie, "", "", nil), 200); state.Authenticated {
		t.Fatal("emergency reset retained administrator sessions")
	}
	cookie, csrf = f.login(t)
	user, token := adminActor(t, f, cookie)
	web, err := f.admin.Sessions.Find(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.BeginEnrollment(t.Context(), user, web, initialPassword); !errors.Is(err, administration.ErrEmergency) {
		t.Fatal("password alone bypassed operator recovery")
	}
	if _, err := f.admin.AcceptEmergency(t.Context(), user, web, initialPassword, strings.Repeat("0", 64)); !errors.Is(err, administration.ErrProof) {
		t.Fatal("invalid operator credential accepted")
	}
	if err := f.db.Ent.AdminPrincipal.UpdateOneID(user.ID).SetEmergencyExpiresAt(time.Now().Add(-time.Second)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.AcceptEmergency(t.Context(), user, web, initialPassword, ticket.Token); !errors.Is(err, administration.ErrProof) {
		t.Fatal("expired operator credential accepted")
	}
	if err := f.db.Ent.AdminPrincipal.UpdateOneID(user.ID).SetEmergencyExpiresAt(ticket.ExpiresAt).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	response := f.request("POST", "/api/v1/admin/mfa/emergency", cookie, csrf, adminOrigin, map[string]string{"current_password": initialPassword, "token": ticket.Token})
	state := decodeAccount[AdminContextBody](t, response, 200)
	if !state.Authenticated || state.MFA.Enrolled || state.MFA.Verified || state.MFA.EmergencyRequired {
		t.Fatal("operator proof granted full MFA permissions")
	}
	recoveredCookie := cookieValue(t, response)
	recoveredCSRF := state.CSRFToken
	freshCookie, _ := f.login(t)
	freshUser, freshToken := adminActor(t, f, freshCookie)
	freshWeb, err := f.admin.Sessions.Find(t.Context(), freshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.BeginEnrollment(t.Context(), freshUser, freshWeb, initialPassword); !errors.Is(err, administration.ErrState) {
		t.Fatal("different first-factor session stole recovery enrollment")
	}
	_, _, _, _ = f.enroll(t, recoveredCookie, recoveredCSRF)
	if _, err := f.admin.AcceptEmergency(t.Context(), freshUser, freshWeb, initialPassword, ticket.Token); err == nil {
		t.Fatal("operator credential replay accepted")
	}
}
