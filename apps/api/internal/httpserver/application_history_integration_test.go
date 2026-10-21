//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/persistence/ent/applicationversion"
	"moetopia.local/aio/api/internal/persistence/ent/emaildelivery"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
)

func TestApplicationVersionHistoryPagination(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	created := decodeAccount[developer.Created](t, f.request("POST", "/api/v1/developer/applications", cookie, csrf, portalOrigin, map[string]any{"public": true, "config": applicationConfig()}), 200)
	app := created.Application
	first, err := f.db.Ent.ApplicationVersion.Query().Where(applicationversion.ApplicationID(app.ID)).Only(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.ApplicationVersion.UpdateOneID(first.ID).SetStatus(applicationversion.StatusRejected).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	for version := 2; version <= 43; version++ {
		status := applicationversion.StatusRejected
		if version == 43 {
			status = applicationversion.StatusDraft
		} else if version == 17 {
			status = applicationversion.StatusApproved
		}
		if err := f.db.Ent.ApplicationVersion.Create().SetID(uuid.NewString()).SetApplicationID(app.ID).SetVersion(version).SetConfig(first.Config).SetStatus(status).SetReviewReason("History fixture").Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.db.Ent.DeveloperApplication.UpdateOneID(app.ID).SetLatestVersion(43).SetActiveVersion(17).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/developer/applications/" + app.ID + "/versions"
	page := decodeAccount[developer.VersionHistory](t, f.request("GET", path, cookie, "", "", nil), 200)
	if len(page.Items) != 20 || page.Items[0].Version.Version != 43 || page.Items[19].Version.Version != 24 || page.NextBefore != 24 || page.ActiveVersion != 17 || page.LatestVersion != 43 || page.Items[0].CreatedAt.IsZero() {
		t.Fatal("first version history page incorrect")
	}
	page = decodeAccount[developer.VersionHistory](t, f.request("GET", path+"?before=24", cookie, "", "", nil), 200)
	if len(page.Items) != 20 || page.Items[0].Version.Version != 23 || page.Items[19].Version.Version != 4 || page.NextBefore != 4 {
		t.Fatal("version pagination duplicated or omitted versions")
	}
	page = decodeAccount[developer.VersionHistory](t, f.request("GET", path+"?before=4", cookie, "", "", nil), 200)
	if len(page.Items) != 3 || page.Items[2].Version.Version != 1 || page.NextBefore != 0 {
		t.Fatal("last version page incorrect")
	}
	page = decodeAccount[developer.VersionHistory](t, f.request("GET", path+"?before=1", cookie, "", "", nil), 200)
	if len(page.Items) != 0 || page.Items == nil || page.NextBefore != 0 {
		t.Fatal("empty history page did not retain array shape")
	}
	_, other, _ := managementFixture(t)
	if f.request("GET", path, other, "", "", nil).Code != 404 || f.request("GET", path, "", "", "", nil).Code != 401 || f.request("GET", path+"?before=-1", cookie, "", "", nil).Code != 422 {
		t.Fatal("history authentication, ownership or cursor validation failed")
	}
	if err := f.db.Ent.DeveloperApplication.UpdateOneID(app.ID).SetDeleted(true).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.request("GET", path, cookie, "", "", nil).Code != 404 {
		t.Fatal("deleted application history remained accessible")
	}
}

func TestApplicationNoticeAtomicityAndSecretRedaction(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), user, false, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	app, err := f.applications.Submit(t.Context(), user, created.Application.ID, created.Application.Revision)
	if err != nil {
		t.Fatal(err)
	}
	count, err := f.db.Ent.EmailDelivery.Query().Count(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE notifications.email_deliveries ADD CONSTRAINT test_application_notice_failure CHECK (false) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE notifications.email_deliveries DROP CONSTRAINT IF EXISTS test_application_notice_failure`); err != nil {
			t.Error("could not remove notification failure fixture")
		}
	}
	defer drop()
	if _, err := f.applications.Review(t.Context(), "dedicated-reviewer", app.ID, app.Revision, true, ""); err == nil {
		t.Fatal("notification storage failure did not abort review")
	}
	current, err := f.applications.Detail(t.Context(), user.ID, app.ID)
	if err != nil || current.ActiveVersion != 0 || current.Revision != app.Revision || current.Latest.Status != "pending" {
		t.Fatal("notification failure partially published application")
	}
	client, err := f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil || !client.TestOnly {
		t.Fatal("notification failure committed protocol configuration")
	}
	if exists, err := f.db.Ent.SecurityEvent.Query().Where(securityevent.TargetID(app.ID), securityevent.Action("application.review.approved")).Exist(t.Context()); err != nil || exists {
		t.Fatal("notification failure committed success audit")
	}
	if after, err := f.db.Ent.EmailDelivery.Query().Count(t.Context()); err != nil || after != count {
		t.Fatal("failed review queued a message")
	}
	drop()
	app, err = f.applications.Review(t.Context(), "dedicated-reviewer", app.ID, app.Revision, true, "Reviewed")
	if err != nil {
		t.Fatal(err)
	}
	access := f.applicationAccess(t, app.ClientID, created.ClientSecret, applicationConfig().RedirectURIs[0])
	client, err = f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE notifications.email_deliveries ADD CONSTRAINT test_application_notice_failure CHECK (false) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.RotateSecret(t.Context(), user, app.ID, app.Revision); err == nil {
		t.Fatal("notification failure did not roll back secret rotation")
	}
	after, err := f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil || !bytes.Equal(client.SecretHash, after.SecretHash) {
		t.Fatal("failed secret notice committed new credentials")
	}
	if _, err := f.applications.Suspend(t.Context(), "dedicated-reviewer", app.ID, app.Revision, true, "Investigating"); err == nil {
		t.Fatal("notification failure did not roll back suspension")
	}
	current, err = f.applications.Detail(t.Context(), user.ID, app.ID)
	if err != nil || current.Suspended || current.Revision != app.Revision || f.checkApplicationToken(access) != 200 {
		t.Fatal("failed suspension committed state or revoked credentials")
	}
	drop()
	rotated := decodeAccount[struct {
		Secret string `json:"client_secret"`
	}](t, f.request("POST", "/api/v1/developer/applications/"+app.ID+"/secret", cookie, csrf, portalOrigin, map[string]any{"revision": app.Revision, "current_password": initialPassword}), 200)
	for range 30 {
		if err := f.outbox.Deliver(t.Context()); err != nil {
			t.Fatal(err)
		}
		left, err := f.db.Ent.EmailDelivery.Query().Where(emaildelivery.AttemptsLT(5)).Count(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if left == 0 {
			break
		}
	}
	notices := 0
	for _, message := range f.sender.values {
		if !strings.Contains(message.Text, app.ID) {
			continue
		}
		notices++
		if message.To != user.Email || strings.Contains(message.Text, rotated.Secret) || strings.Contains(message.Text, created.ClientSecret) || bytes.Contains([]byte(message.Text), []byte(initialPassword)) {
			t.Fatal("application notice recipient or secret redaction incorrect")
		}
	}
	if notices != 2 {
		t.Fatal("successful review and secret rotation did not each queue one notice")
	}
}
