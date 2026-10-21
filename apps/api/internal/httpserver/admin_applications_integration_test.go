//go:build integration

package httpserver

import (
	"context"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/persistence/ent/adminelevation"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
)

func TestAdminReviewSuspensionAndAudit(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	owner, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), owner, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	app, err := f.applications.Submit(t.Context(), owner, created.Application.ID, created.Application.Revision)
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/admin/applications/" + app.ID
	if got := f.request("POST", path+"/review", cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision, "approve": true, "reason": ""}).Code; got != 403 {
		t.Fatal("review bypassed MFA")
	}
	if f.request("GET", path+"/versions", cookie, "", "", nil).Code != 403 {
		t.Fatal("application history bypassed MFA")
	}
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	list := decodeAccount[developer.Applications](t, f.request("GET", "/api/v1/admin/applications?status=pending", cookie, "", "", nil), 200)
	found := false
	for _, item := range list.Items {
		if item.ID == app.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("pending application absent from review queue")
	}
	app = decodeAccount[developer.Application](t, f.request("POST", path+"/review", cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision, "approve": true, "reason": "Reviewed"}), 200)
	if app.ActiveVersion != 1 || app.Latest.Status != "approved" {
		t.Fatal("authenticated review did not publish application")
	}
	history := decodeAccount[developer.VersionHistory](t, f.request("GET", path+"/versions", cookie, "", "", nil), 200)
	if len(history.Items) != 1 || history.Items[0].Status != "approved" || history.ActiveVersion != 1 {
		t.Fatal("administrator history omitted approved configuration")
	}
	access := f.applicationAccess(t, app.ClientID, "", applicationConfig().RedirectURIs[0])
	app = decodeAccount[developer.Application](t, f.request("POST", path+"/suspension", cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision, "suspended": true, "reason": "Investigating application behavior"}), 200)
	if !app.Suspended || f.checkApplicationToken(access) != 401 {
		t.Fatal("platform suspension did not stop credential use")
	}
	app, err = f.applications.SetEnabled(t.Context(), owner, app.ID, app.Revision, true)
	if err != nil {
		t.Fatal(err)
	}
	client, err := f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil || client.Enabled || !app.Suspended {
		t.Fatal("owner bypassed platform suspension by enabling application")
	}
	app = decodeAccount[developer.Application](t, f.request("POST", path+"/suspension", cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision, "suspended": false, "reason": ""}), 200)
	if app.Suspended || f.checkApplicationToken(access) != 401 {
		t.Fatal("resuming restored old credentials")
	}
	audits, err := f.db.Ent.SecurityEvent.Query().Where(securityevent.UserID(f.userID), securityevent.Action("application.review.approved"), securityevent.TargetID(app.ID), securityevent.Outcome("success")).All(t.Context())
	if err != nil || len(audits) != 1 || audits[0].RequestID == "" {
		t.Fatal("review audit lacks actor, target, outcome or request ID")
	}
	if _, err := f.db.Ent.AdminElevation.Update().Where(adminelevation.UserID(f.userID)).SetExpiresAt(time.Now().Add(-time.Second)).Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := f.request("GET", path, cookie, "", "", nil).Code; got != 403 {
		t.Fatal("expired MFA still permits review data")
	}
	if f.request("GET", path+"/versions", cookie, "", "", nil).Code != 403 {
		t.Fatal("expired MFA still permits application history")
	}
}
func TestAdminReviewAndGateShareOneTransaction(t *testing.T) {
	f := newAdminFixture(t)
	f.db.SQL.SetMaxOpenConns(1)
	cookie, csrf := f.login(t)
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	owner, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	config := applicationConfig()
	config.Name = "Blocked-admin-audit-test"
	created, err := f.applications.Create(t.Context(), owner, true, config)
	if err != nil {
		t.Fatal(err)
	}
	app, err := f.applications.Submit(t.Context(), owner, created.Application.ID, created.Application.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE audit.security_events ADD CONSTRAINT test_admin_audit_failure CHECK (action <> 'application.review.approved') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE audit.security_events DROP CONSTRAINT test_admin_audit_failure`); err != nil {
			t.Error("could not clear dedicated audit failure fixture")
		}
	})
	if got := f.request("POST", "/api/v1/admin/applications/"+app.ID+"/review", cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision, "approve": true, "reason": ""}).Code; got != 503 {
		t.Fatalf("review audit failure HTTP %d", got)
	}
	current, err := f.applications.Detail(t.Context(), owner.ID, app.ID)
	if err != nil || current.ActiveVersion != 0 || current.Latest.Status != "pending" || current.Revision != app.Revision {
		t.Fatal("failed admin audit partially published configuration")
	}
	client, err := f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil || !client.TestOnly {
		t.Fatal("failed admin command committed protocol client update")
	}
}

func TestAdminCanSuspendApplicationWithDisabledOwner(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	other, _, _ := managementFixture(t)
	owner, err := other.identity.User(t.Context(), other.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), owner, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(owner.ID).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	app := decodeAccount[developer.Application](t, f.request("POST", "/api/v1/admin/applications/"+created.Application.ID+"/suspension", cookie, csrf, adminOrigin, map[string]any{"revision": created.Application.Revision, "suspended": true, "reason": "Reviewing owner security"}), 200)
	if !app.Suspended {
		t.Fatal("disabled owner prevented platform suspension")
	}
	if email, err := f.identity.VerifiedEmail(t.Context(), owner.ID); err != nil || email != owner.Email {
		t.Fatal("disabled owner cannot receive a security notice")
	}
}
