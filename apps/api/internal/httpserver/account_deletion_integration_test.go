//go:build integration

package httpserver

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/lifecycle"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence/ent/accountchallenge"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
	"moetopia.local/aio/api/internal/persistence/ent/emaildelivery"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
	"moetopia.local/aio/api/internal/persistence/ent/websession"
)

func deletionFixture(t *testing.T) (*portalFixture, *lifecycle.Service, string, string) {
	t.Helper()
	f, cookie, csrf := managementFixture(t)
	service := lifecycle.New(f.db.Ent, f.identity, f.applications, f.protocol)
	f.handler = New(Dependencies{Tokens: f.protocol, Valkey: f.cache, Portal: PortalEndpoints{Service: f.portal}, Developers: f.applications, Lifecycle: service, Accounts: AccountEndpoints{Identity: f.identity, Sessions: f.sessions, Cache: f.cache, Authorization: f.protocol}})
	return f, service, cookie, csrf
}

func TestAccountDeletionRequestAndCancellation(t *testing.T) {
	f, _, cookie, csrf := deletionFixture(t)
	email := currentEmail(t, f)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	app, err := f.applications.Create(t.Context(), user, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	access := f.applicationAccess(t, app.Application.ClientID, "", applicationConfig().RedirectURIs[0])
	body := map[string]any{"current_password": initialPassword, "revision": 0}
	if f.request("POST", "/api/v1/portal/deletion", cookie, csrf, "https://attacker.test", body).Code != 403 {
		t.Fatal("deletion bypassed Origin")
	}
	if f.request("POST", "/api/v1/portal/deletion", f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, body).Code != 403 {
		t.Fatal("identity cookie bypassed RP protection")
	}
	result := decodeAccount[identity.Deletion](t, f.request("POST", "/api/v1/portal/deletion", cookie, csrf, portalOrigin, body), 200)
	if result.DueAt.Sub(result.RequestedAt) != 7*24*time.Hour {
		t.Fatal("incorrect deletion cooldown")
	}
	if _, err := f.identity.User(t.Context(), user.ID); !errors.Is(err, identity.ErrMissing) {
		t.Fatal("pending account remains usable")
	}
	if _, err := f.identity.Login(t.Context(), email, initialPassword); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("pending account can sign in")
	}
	if f.checkApplicationToken(access) != 401 {
		t.Fatal("deletion left OAuth access active")
	}
	if exists, err := f.db.Ent.WebSession.Query().Where(websession.UserID(user.ID)).Exist(t.Context()); err != nil || exists {
		t.Fatal("deletion left browser sessions")
	}
	if _, err := f.identity.UpdateProfile(t.Context(), user.ID, "Bypass", "", 1); err == nil {
		t.Fatal("pending account profile is editable")
	}
	if _, err := f.portal.Sessions.CreateVerified(t.Context(), user.ID, "", 1, f.identity); err == nil {
		t.Fatal("pending account can recreate sessions")
	}
	anonymousResponse := f.request("GET", "/api/v1/account/context", "", "", "", nil)
	anonymous := decodeAccount[AccountContextBody](t, anonymousResponse, 200)
	anonymousCookie := cookieValue(t, anonymousResponse)
	challenge := decodeAccount[struct {
		ChallengeID string `json:"challenge_id"`
	}](t, f.request("POST", "/api/v1/account/deletion-cancellations", anonymousCookie, anonymous.CSRFToken, portalOrigin, map[string]string{"email": email, "locale": "en"}), 200)
	id := challenge.ChallengeID
	code := resetCode(t, f, email)
	if err := f.identity.CancelDeletion(t.Context(), id, code, "wrong"); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("cancellation skipped password")
	}
	if err := f.identity.CancelDeletion(t.Context(), id, "bad-code", initialPassword); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("cancellation skipped email proof")
	}
	decodeAccount[any](t, f.request("POST", "/api/v1/account/deletion-cancellations/confirm", anonymousCookie, anonymous.CSRFToken, portalOrigin, map[string]string{"challenge_id": id, "code": code, "current_password": initialPassword}), 204)
	if err := f.identity.CancelDeletion(t.Context(), id, code, initialPassword); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("cancellation challenge replay accepted")
	}
	if _, err := f.identity.Login(t.Context(), email, initialPassword); err != nil {
		t.Fatal("cancelled account cannot sign in")
	}
	state := decodeAccount[PortalContextBody](t, f.request("GET", "/api/v1/portal/context", cookie, "", "", nil), 200)
	if state.Authenticated || f.checkApplicationToken(access) != 401 {
		t.Fatal("cancellation resurrected old credentials")
	}
	row, err := f.db.Ent.DeveloperApplication.Get(t.Context(), app.Application.ID)
	if err != nil || row.Enabled {
		t.Fatal("cancellation automatically enabled application")
	}
}

func TestAccountDeletionGuardsAndRollback(t *testing.T) {
	f, service, _, _ := deletionFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), user, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	app, err := f.applications.Submit(t.Context(), user, created.Application.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Review(t.Context(), "dedicated-reviewer", app.ID, app.Revision, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(t.Context(), user, initialPassword, 0); !errors.Is(err, developer.ErrPublished) {
		t.Fatal("enabled published application did not block deletion")
	}
	if _, err := f.applications.SetEnabled(t.Context(), user, app.ID, app.Revision, false); err != nil {
		t.Fatal(err)
	}
	draft, err := f.applications.Create(t.Context(), user, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	access := f.applicationAccess(t, draft.Application.ClientID, "", applicationConfig().RedirectURIs[0])
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE notifications.email_deliveries ADD CONSTRAINT test_deletion_mail_failure CHECK (false) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE notifications.email_deliveries DROP CONSTRAINT IF EXISTS test_deletion_mail_failure`); err != nil {
			t.Error("could not clear deletion notice failure fixture")
		}
	}
	defer drop()
	if _, err := service.Request(t.Context(), user, initialPassword, 0); err == nil {
		t.Fatal("deletion notice failure did not abort request")
	}
	row, err := f.db.Ent.AccountUser.Get(t.Context(), user.ID)
	if err != nil || row.DeletionDueAt != nil || row.SecurityVersion != 0 {
		t.Fatal("failed deletion committed account state")
	}
	if f.checkApplicationToken(access) != 200 {
		t.Fatal("failed deletion revoked credentials")
	}
	active, err := f.db.Ent.DeveloperApplication.Get(t.Context(), draft.Application.ID)
	if err != nil || !active.Enabled {
		t.Fatal("failed deletion disabled applications")
	}
	drop()
	if err := f.db.Ent.AdminPrincipal.Create().SetID(user.ID).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(t.Context(), user, initialPassword, 0); !errors.Is(err, administration.ErrProtected) {
		t.Fatal("administrator can use ordinary deletion")
	}
}

func TestAccountDeletionPurgeAndFailureRetry(t *testing.T) {
	f, service, _, _ := deletionFixture(t)
	other, _, _ := managementFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), user, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	f.applicationAccess(t, created.Application.ClientID, "", applicationConfig().RedirectURIs[0])
	f.callback(t, "approve")
	grants, err := f.db.Ent.AuthorizationGrant.Query().Where(authorizationgrant.UserID(user.ID)).IDs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.AuthorizationGrant.Update().Where(authorizationgrant.UserID(user.ID)).SetUserID("").Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(t.Context(), user, initialPassword, 0); err != nil {
		t.Fatal(err)
	}
	if err := service.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	row, err := f.db.Ent.AccountUser.Get(t.Context(), user.ID)
	if err != nil || row.DeletionDueAt == nil {
		t.Fatal("cleanup purged before cooldown")
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(user.ID).SetDeletionRequestedAt(time.Now().Add(-8 * 24 * time.Hour)).SetDeletionDueAt(time.Now().Add(-time.Second)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := f.outbox.Enqueue(t.Context(), f.db.Ent, notifications.Message{UserID: user.ID, To: user.Email, Subject: "Pending test", Text: "purge fixture"}, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE audit.security_events ADD CONSTRAINT test_deletion_purge_failure CHECK (action <> 'account.deletion.completed') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE audit.security_events DROP CONSTRAINT IF EXISTS test_deletion_purge_failure`); err != nil {
			t.Error("could not clear deletion purge failure fixture")
		}
	}
	defer drop()
	if err := service.Cleanup(t.Context()); err == nil {
		t.Fatal("purge audit failure did not abort cleanup")
	}
	if _, err := f.db.Ent.DeveloperApplication.Get(t.Context(), created.Application.ID); err != nil {
		t.Fatal("failed purge deleted application")
	}
	if _, err := f.db.Ent.OAuthClient.Get(t.Context(), created.Application.ClientID); err != nil {
		t.Fatal("failed purge deleted protocol client")
	}
	drop()
	f.db.SQL.SetMaxOpenConns(1)
	if err := service.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.db.SQL.SetMaxOpenConns(8)
	if exists, err := f.db.Ent.AccountUser.Query().Where(accountuser.Email(user.Email)).Exist(t.Context()); err != nil || exists {
		t.Fatal("purge retained email")
	}
	for _, check := range []func(context.Context) (bool, error){
		f.db.Ent.DeveloperApplication.Query().Where(developerapplication.OwnerID(user.ID)).Exist,
		f.db.Ent.AuthorizationGrant.Query().Where(authorizationgrant.UserID(user.ID)).Exist,
		f.db.Ent.AuthorizationGrant.Query().Where(authorizationgrant.IDIn(grants...)).Exist,
		f.db.Ent.ProtocolSession.Query().Where(protocolsession.RequestIDIn(grants...)).Exist,
		f.db.Ent.AccountChallenge.Query().Where(accountchallenge.UserID(user.ID)).Exist,
		f.db.Ent.WebSession.Query().Where(websession.UserID(user.ID)).Exist,
		f.db.Ent.EmailDelivery.Query().Where(emaildelivery.UserID(user.ID)).Exist,
	} {
		if exists, err := check(t.Context()); err != nil || exists {
			t.Fatal("purge retained owned resources")
		}
	}
	if _, err := other.identity.User(t.Context(), other.userID); err != nil {
		t.Fatal("purge affected an unrelated account")
	}
	if exists, err := f.db.Ent.SecurityEvent.Query().Where(securityevent.UserID(user.ID), securityevent.Action("account.deletion.completed")).Exist(t.Context()); err != nil || !exists {
		t.Fatal("purge removed pseudonymous audit event")
	}
	var group sync.WaitGroup
	for range 2 {
		group.Go(func() {
			if err := service.Cleanup(t.Context()); err != nil {
				t.Error("idempotent cleanup failed")
			}
		})
	}
	group.Wait()
}

func TestDeletionCancellationRecoveryDeadlineAndAttemptLimit(t *testing.T) {
	f, service, _, _ := deletionFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(t.Context(), user, initialPassword, 0); err != nil {
		t.Fatal(err)
	}
	id, err := f.identity.BeginReset(t.Context(), user.Email, "en")
	if err != nil {
		t.Fatal(err)
	}
	code := resetCode(t, f, user.Email)
	if err := f.identity.ResetPassword(t.Context(), id, code, nextPassword); err != nil {
		t.Fatal(err)
	}
	if _, err := f.identity.Login(t.Context(), user.Email, nextPassword); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("password recovery cancelled deletion")
	}
	id, err = f.identity.BeginDeletionCancel(t.Context(), user.Email, "en")
	if err != nil {
		t.Fatal(err)
	}
	code = resetCode(t, f, user.Email)
	for range 5 {
		if err := f.identity.CancelDeletion(t.Context(), id, "bad-code", nextPassword); !errors.Is(err, identity.ErrInvalid) {
			t.Fatal("invalid cancellation proof accepted")
		}
	}
	if err := f.identity.CancelDeletion(t.Context(), id, code, nextPassword); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("OTP attempt limit bypassed")
	}
	id, err = f.identity.BeginDeletionCancel(t.Context(), user.Email, "en")
	if err != nil {
		t.Fatal(err)
	}
	code = resetCode(t, f, user.Email)
	if err := f.db.Ent.AccountUser.UpdateOneID(user.ID).SetDeletionRequestedAt(time.Now().Add(-8 * 24 * time.Hour)).SetDeletionDueAt(time.Now().Add(-time.Second)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := f.identity.CancelDeletion(t.Context(), id, code, nextPassword); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("deletion was cancelled after deadline")
	}
	if err := service.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
}
