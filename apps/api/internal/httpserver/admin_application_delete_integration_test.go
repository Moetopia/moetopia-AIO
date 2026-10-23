//go:build integration

package httpserver

import (
	"context"
	"testing"

	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
)

func TestAdminApplicationDeletionRequiresMFAAndRevokesAccess(t *testing.T) {
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
	app := created.Application
	path := "/api/v1/admin/applications/" + app.ID
	body := map[string]any{"revision": app.Revision, "reason": "Remove test application"}
	if f.request("DELETE", path, cookie, csrf, adminOrigin, body).Code != 403 {
		t.Fatal("deletion bypassed MFA")
	}
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	access := f.applicationAccess(t, app.ClientID, "", applicationConfig().RedirectURIs[0])
	if f.request("DELETE", path, cookie, csrf, portalOrigin, body).Code != 403 {
		t.Fatal("cross-origin deletion accepted")
	}
	if f.request("DELETE", path, cookie, "", adminOrigin, body).Code != 403 {
		t.Fatal("deletion bypassed CSRF")
	}
	if f.request("DELETE", path, cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision, "reason": "  "}).Code != 422 {
		t.Fatal("blank deletion reason accepted")
	}
	if f.request("DELETE", path, cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision + 1, "reason": "Stale decision"}).Code != 409 {
		t.Fatal("stale deletion revision accepted")
	}
	f.db.SQL.SetMaxOpenConns(1)
	if f.request("DELETE", path, cookie, csrf, adminOrigin, body).Code != 204 {
		t.Fatal("administrator could not delete application")
	}
	if f.checkApplicationToken(access) != 401 {
		t.Fatal("deleted application token remained active")
	}
	if f.request("GET", path, cookie, "", "", nil).Code != 404 {
		t.Fatal("deleted application remained available")
	}
	list := decodeAccount[developer.Applications](t, f.request("GET", "/api/v1/admin/applications?status=all", cookie, "", "", nil), 200)
	for _, item := range list.Items {
		if item.ID == app.ID {
			t.Fatal("deleted application remained in list")
		}
	}
	row, err := f.db.Ent.DeveloperApplication.Get(t.Context(), app.ID)
	if err != nil || !row.Deleted || row.Enabled || row.Revision != app.Revision+1 {
		t.Fatal("deletion did not preserve its tombstone")
	}
	client, err := f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil || client.Enabled {
		t.Fatal("deleted client remains enabled")
	}
	event, err := f.db.Ent.SecurityEvent.Query().Where(securityevent.Action("application.admin.deleted"), securityevent.TargetID(app.ID), securityevent.UserID(f.userID)).Only(t.Context())
	if err != nil || event.Reason != "Remove test application" || event.RequestID == "" {
		t.Fatal("administrator deletion lacks audit evidence")
	}
	if _, err := f.applications.SetEnabled(t.Context(), owner, app.ID, row.Revision, true); err != developer.ErrMissing {
		t.Fatal("owner restored deleted application")
	}
}

func TestAdminApplicationDeletionRollsBackWhenAuditFails(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	owner, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), owner, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	app := created.Application
	access := f.applicationAccess(t, app.ClientID, "", applicationConfig().RedirectURIs[0])
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE audit.security_events ADD CONSTRAINT test_admin_delete_audit CHECK (action <> 'application.admin.deleted') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE audit.security_events DROP CONSTRAINT test_admin_delete_audit`); err != nil {
			t.Error("could not clear isolated audit failure fixture")
		}
	})
	if f.request("DELETE", "/api/v1/admin/applications/"+app.ID, cookie, csrf, adminOrigin, map[string]any{"revision": app.Revision, "reason": "Audit must succeed"}).Code != 503 {
		t.Fatal("deletion ignored failed audit")
	}
	row, err := f.db.Ent.DeveloperApplication.Get(t.Context(), app.ID)
	if err != nil || row.Deleted || !row.Enabled || row.Revision != app.Revision {
		t.Fatal("failed deletion partially changed application")
	}
	if f.checkApplicationToken(access) != 200 {
		t.Fatal("failed deletion committed token revocation")
	}
}
