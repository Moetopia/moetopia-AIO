//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"moetopia.local/aio/api/internal/persistence/ent/emaildelivery"
)

func TestAdminStatisticsRegistrationCommitAndMissingHistory(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	if f.request("GET", "/api/v1/admin/statistics", cookie, "", "", nil).Code != 403 {
		t.Fatal("statistics bypassed MFA")
	}
	cookie, _, _, _ = f.enroll(t, cookie, csrf)
	before := decodeAccount[AdminStatisticsOutputBody](t, f.request("GET", "/api/v1/admin/statistics", cookie, "", "", nil), 200)
	if len(before.Identity.Registrations) != 30 || before.Identity.Registrations[0].Registrations != nil || before.Identity.Registrations[29].Registrations == nil || before.Identity.RecordingSince.IsZero() || before.Timezone != "UTC" {
		t.Fatal("admin statistics fabricated collection history")
	}
	email := uuid.NewString() + "@registration-stats.test"
	id, code := f.verification(t, email)
	invite, err := f.identity.IssueInvitation(t.Context(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE identity.registration_events ADD CONSTRAINT test_registration_statistics_failure CHECK (false) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE identity.registration_events DROP CONSTRAINT IF EXISTS test_registration_statistics_failure`); err != nil {
			t.Error("could not remove registration statistics fixture")
		}
	}
	defer drop()
	if err := f.identity.Register(t.Context(), id, code, invite, "Statistics reader", initialPassword); err == nil {
		t.Fatal("registration ignored failed transactional observation")
	}
	if exists, err := f.db.Ent.AccountUser.Query().Where(accountuser.Email(email)).Exist(t.Context()); err != nil || exists {
		t.Fatal("failed statistics committed account")
	}
	drop()
	if err := f.identity.Register(t.Context(), id, code, invite, "Statistics reader", initialPassword); err != nil {
		t.Fatal("failed transaction consumed registration proof", err)
	}
	response := f.request("GET", "/api/v1/admin/statistics", cookie, "", "", nil)
	after := decodeAccount[AdminStatisticsOutputBody](t, response, 200)
	if after.Identity.TotalUsers != before.Identity.TotalUsers+1 || *after.Identity.Registrations[29].Registrations != *before.Identity.Registrations[29].Registrations+1 || bytes.Contains(response.Body.Bytes(), []byte(email)) {
		t.Fatal("admin totals/trend incorrect or exposed registration identity")
	}
	owner, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), owner, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.Submit(t.Context(), owner, created.Application.ID, created.Application.Revision); err != nil {
		t.Fatal(err)
	}
	appStats, err := f.applications.Overview(t.Context())
	if err != nil || appStats.Total != after.Applications.Total+1 || appStats.Pending != after.Applications.Pending+1 {
		t.Fatal("application totals used list length or stale state")
	}
}

type AdminStatisticsOutputBody struct {
	Timezone string `json:"timezone"`
	Identity struct {
		TotalUsers     int       `json:"total_users"`
		RecordingSince time.Time `json:"recording_since"`
		Registrations  []struct {
			Registrations *int `json:"registrations"`
		} `json:"registrations"`
	} `json:"identity"`
	Applications developer.Overview `json:"applications"`
}

func TestAdminNotificationRetryGuardsAndAuditRollback(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	target, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.outbox.Enqueue(t.Context(), f.db.Ent, notifications.Message{UserID: target.ID, To: target.Email, Subject: "Notification retry fixture", Text: "Private notification fixture"}, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	job, err := f.db.Ent.EmailDelivery.Query().Where(emaildelivery.UserID(target.ID)).Order(ent.Desc(emaildelivery.FieldAvailableAt)).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.EmailDelivery.UpdateOneID(job.ID).SetAttempts(5).SetLeaseUntil(time.Unix(0, 0)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	response := f.request("GET", "/api/v1/admin/notifications/failed", cookie, "", "", nil)
	failed := decodeAccount[notifications.FailedDeliveries](t, response, 200)
	found := false
	for _, item := range failed.Items {
		if item.ID == job.ID {
			found = true
		}
	}
	if !found || bytes.Contains(response.Body.Bytes(), []byte(target.Email)) || bytes.Contains(response.Body.Bytes(), []byte("Private notification fixture")) {
		t.Fatal("failed delivery listing missing or leaked payload")
	}
	path := "/api/v1/admin/notifications/" + job.ID + "/retry"
	body := map[string]any{"attempts": 5, "reason": "SMTP configuration repaired"}
	if f.request("POST", path, cookie, "", adminOrigin, body).Code != 403 {
		t.Fatal("notification retry bypassed CSRF")
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE audit.security_events ADD CONSTRAINT test_retry_audit_failure CHECK (action <> 'notification.requeued') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE audit.security_events DROP CONSTRAINT IF EXISTS test_retry_audit_failure`); err != nil {
			t.Error("could not remove retry audit fixture")
		}
	}
	defer drop()
	if f.request("POST", path, cookie, csrf, adminOrigin, body).Code != 503 {
		t.Fatal("retry ignored failed audit")
	}
	row, err := f.db.Ent.EmailDelivery.Get(t.Context(), job.ID)
	if err != nil || row.Attempts != 5 {
		t.Fatal("failed retry changed job state")
	}
	drop()
	if f.request("POST", path, cookie, csrf, adminOrigin, body).Code != 204 || f.request("POST", path, cookie, csrf, adminOrigin, body).Code != 409 {
		t.Fatal("notification retry state/duplicate guards incorrect")
	}
	if err := f.db.Ent.EmailDelivery.UpdateOneID(job.ID).SetAttempts(5).SetExpiresAt(time.Now().Add(-time.Second)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.request("POST", path, cookie, csrf, adminOrigin, body).Code != 409 {
		t.Fatal("retry resurrected expired notification")
	}
}
