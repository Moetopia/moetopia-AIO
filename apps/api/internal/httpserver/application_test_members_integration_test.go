//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/clienttestsubject"
	"moetopia.local/aio/api/internal/persistence/ent/emaildelivery"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
)

func memberFixture(t *testing.T, f *portalFixture) (*portalFixture, identity.User) {
	t.Helper()
	id := uuid.NewString()
	if err := f.db.Ent.AccountUser.Create().SetID(id).SetEmail(id + "@member.test").SetDisplayName("Test member").SetPasswordHash("unusable-test-hash").SetEmailVerifiedAt(time.Now()).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	user, err := f.identity.User(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	token, err := f.sessions.CreateVerified(t.Context(), id, "", user.SecurityVersion, f.identity)
	if err != nil {
		t.Fatal(err)
	}
	member := *f
	member.userID, member.identityToken = id, token
	member.identityCookie = f.sessions.CookieName() + "=" + token
	return &member, user
}

func (f *portalFixture) testMemberCode(t *testing.T, client, redirect string) (string, string) {
	t.Helper()
	verifier := strings.Repeat("m", 43)
	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{"client_id": {client}, "redirect_uri": {redirect}, "response_type": {"code"}, "scope": {"openid email offline_access"}, "state": {"member-state"}, "nonce": {"member-nonce"}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}}
	start := f.request("GET", "/oauth/authorize?"+query.Encode(), f.identityCookie, "", "", nil)
	if start.Code != 303 {
		t.Fatalf("member authorization HTTP %d", start.Code)
	}
	page, _ := url.Parse(start.Header().Get("Location"))
	decision := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, f.request("POST", "/api/v1/account/authorizations/"+page.Query().Get("authorization_request"), cookieValue(t, start)+"; "+f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"decision": "approve", "subject_id": f.userID}), 200)
	callback, _ := url.Parse(decision.RedirectTo)
	return callback.Query().Get("code"), verifier
}

func (f *portalFixture) memberProtocol(path, client, secret string, form url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	r.RemoteAddr = f.ip
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(client, secret)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}

func TestApplicationTestMemberAccessRevocationAndPublication(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	member, user := memberFixture(t, f)
	config := applicationConfig()
	config.Scopes = append(config.Scopes, "offline_access")
	created := decodeAccount[developer.Created](t, f.request("POST", "/api/v1/developer/applications", cookie, csrf, portalOrigin, map[string]any{"public": false, "config": config}), 200)
	app, secret := created.Application, created.ClientSecret
	path := "/api/v1/developer/applications/" + app.ID + "/test-members"
	empty := decodeAccount[developer.TestMembers](t, f.request("GET", path, cookie, "", "", nil), 200)
	if empty.Items == nil || len(empty.Items) != 0 || empty.Revision != app.Revision || empty.Limit != authorization.MaxTestSubjects {
		t.Fatal("initial roster incorrect")
	}
	body := map[string]any{"revision": app.Revision, "user_id": user.ID}
	missingCSRF := f.request("POST", path, cookie, "", portalOrigin, body).Code
	foreignOrigin := f.request("POST", path, cookie, csrf, "https://foreign.test", body).Code
	identityOnly := f.request("POST", path, f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, body).Code
	if missingCSRF != 403 || foreignOrigin != 403 || identityOnly != 403 {
		t.Fatalf("member mutation boundary: CSRF=%d origin=%d identity=%d", missingCSRF, foreignOrigin, identityOnly)
	}
	addedResponse := f.request("POST", path, cookie, csrf, portalOrigin, body)
	added := decodeAccount[developer.TestMembers](t, addedResponse, 200)
	if added.Revision != app.Revision+1 || len(added.Items) != 1 || added.Items[0].UserID != user.ID || added.Items[0].CreatedAt.IsZero() || bytes.Contains(addedResponse.Body.Bytes(), []byte(user.Email)) {
		t.Fatal("member response invalid or exposed email")
	}
	app.Revision = added.Revision
	memberPortalToken, err := f.portal.Sessions.CreateVerified(t.Context(), user.ID, "", user.SecurityVersion, f.identity)
	if err != nil {
		t.Fatal(err)
	}
	memberCookie := f.portal.Sessions.CookieName() + "=" + memberPortalToken
	if f.request("GET", path, memberCookie, "", "", nil).Code != 404 || f.request("GET", "/api/v1/developer/applications/"+app.ID, memberCookie, "", "", nil).Code != 404 {
		t.Fatal("test access granted management access")
	}
	if f.request("POST", path, cookie, csrf, portalOrigin, body).Code != 409 {
		t.Fatal("stale member revision accepted")
	}
	code, verifier := member.testMemberCode(t, app.ClientID, config.RedirectURIs[0])
	tokens := decodeAccount[struct {
		Access  string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}](t, f.memberProtocol("/oauth/token", app.ClientID, secret, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {config.RedirectURIs[0]}}), 200)
	if tokens.Access == "" || tokens.Refresh == "" || f.checkApplicationToken(tokens.Access) != 200 {
		t.Fatal("test member could not use approved tokens")
	}
	membership, err := f.db.Ent.ClientTestSubject.Query().Where(clienttestsubject.ClientID(app.ClientID), clienttestsubject.UserID(user.ID)).Only(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.ClientTestSubject.DeleteOneID(membership.ID).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.checkApplicationToken(tokens.Access) != 401 {
		t.Fatal("online access trusted a grant without live test membership")
	}
	if err := f.db.Ent.ClientTestSubject.Create().SetID(membership.ID).SetClientID(app.ClientID).SetUserID(user.ID).SetCreatedAt(membership.CreatedAt).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.checkApplicationToken(tokens.Access) != 200 {
		t.Fatal("live membership denial corrupted otherwise-valid grant")
	}
	unusedCode, unusedVerifier := member.testMemberCode(t, app.ClientID, config.RedirectURIs[0])
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE notifications.email_deliveries ADD CONSTRAINT test_member_remove_failure CHECK (user_id <> '`+user.ID+`') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	clearFailure := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE notifications.email_deliveries DROP CONSTRAINT IF EXISTS test_member_remove_failure`); err != nil {
			t.Error("could not clear removal failure fixture")
		}
	}
	defer clearFailure()
	if f.request("DELETE", path+"/"+user.ID, cookie, csrf, portalOrigin, map[string]int{"revision": app.Revision}).Code != 503 || f.checkApplicationToken(tokens.Access) != 200 {
		t.Fatal("failed removal did not roll back token revocation")
	}
	clearFailure()
	removed := decodeAccount[developer.TestMembers](t, f.request("DELETE", path+"/"+user.ID, cookie, csrf, portalOrigin, map[string]int{"revision": app.Revision}), 200)
	app.Revision = removed.Revision
	if len(removed.Items) != 0 || removed.Items == nil || f.checkApplicationToken(tokens.Access) != 401 {
		t.Fatal("removed member retained online access")
	}
	active := decodeAccount[struct {
		Active bool `json:"active"`
	}](t, f.memberProtocol("/oauth/introspect", app.ClientID, secret, url.Values{"token": {tokens.Access}}), 200)
	if active.Active {
		t.Fatal("removed member introspection remained active")
	}
	if f.memberProtocol("/oauth/token", app.ClientID, secret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.Refresh}}).Code != 400 || f.memberProtocol("/oauth/token", app.ClientID, secret, url.Values{"grant_type": {"authorization_code"}, "code": {unusedCode}, "code_verifier": {unusedVerifier}, "redirect_uri": {config.RedirectURIs[0]}}).Code != 400 {
		t.Fatal("removed member exchanged old credentials")
	}
	owner, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	roster, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, user.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	app.Revision = roster.Revision
	if f.checkApplicationToken(tokens.Access) != 401 {
		t.Fatal("re-add revived revoked grants")
	}
	fresh := member.applicationAccess(t, app.ClientID, secret, config.RedirectURIs[0])
	if f.checkApplicationToken(fresh) != 200 {
		t.Fatal("re-added member cannot obtain fresh consent")
	}
	app, err = f.applications.Submit(t.Context(), owner, app.ID, app.Revision)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Review(t.Context(), "test-reviewer", app.ID, app.Revision, true, "")
	if err != nil {
		t.Fatal(err)
	}
	publicToken := member.applicationAccess(t, app.ClientID, secret, config.RedirectURIs[0])
	_, err = f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, user.ID, false)
	if err != nil || f.checkApplicationToken(publicToken) != 200 {
		t.Fatal("published roster removal revoked ordinary public access")
	}
	if err := f.protocol.AllowSubject(t.Context(), app.ClientID, uuid.NewString()); err != nil {
		t.Fatal("published app remained test-only")
	}
}

func TestApplicationTestMemberValidationRollbackQuotaAndCleanup(t *testing.T) {
	f, _, _ := managementFixture(t)
	_, target := memberFixture(t, f)
	owner, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), owner, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	app := created.Application
	for _, id := range []string{"bad", uuid.Nil.String(), owner.ID, uuid.NewString()} {
		if _, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, id, true); !errors.Is(err, developer.ErrInvalid) {
			t.Fatal("invalid subject accepted")
		}
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(target.ID).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, target.ID, true); !errors.Is(err, developer.ErrInvalid) {
		t.Fatal("disabled member accepted")
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(target.ID).SetEnabled(true).SetDeletionRequestedAt(time.Now()).SetDeletionDueAt(time.Now().Add(time.Hour)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, target.ID, true); !errors.Is(err, developer.ErrInvalid) {
		t.Fatal("deleting member accepted")
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(target.ID).ClearDeletionRequestedAt().ClearDeletionDueAt().Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE notifications.email_deliveries ADD CONSTRAINT test_member_notice_failure CHECK (user_id <> '`+target.ID+`') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE notifications.email_deliveries DROP CONSTRAINT IF EXISTS test_member_notice_failure`); err != nil {
			t.Error("failed to remove test constraint")
		}
	}
	defer drop()
	if _, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, target.ID, true); err == nil {
		t.Fatal("notification failure did not abort membership")
	}
	row, err := f.db.Ent.DeveloperApplication.Get(t.Context(), app.ID)
	if err != nil || row.Revision != app.Revision {
		t.Fatal("failed mutation changed revision")
	}
	if n, err := f.db.Ent.ClientTestSubject.Query().Where(clienttestsubject.ClientID(app.ClientID)).Count(t.Context()); err != nil || n != 0 {
		t.Fatal("failed mutation retained membership")
	}
	if n, err := f.db.Ent.SecurityEvent.Query().Where(securityevent.UserID(owner.ID), securityevent.Action("application.test_member.added"), securityevent.TargetID(app.ID+":"+target.ID)).Count(t.Context()); err != nil || n != 0 {
		t.Fatal("failed mutation retained audit")
	}
	drop()
	f.db.SQL.SetMaxOpenConns(1)
	roster, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, target.ID, true)
	f.db.SQL.SetMaxOpenConns(8)
	if err != nil {
		t.Fatal(err)
	}
	app.Revision = roster.Revision
	if _, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, target.ID, true); !errors.Is(err, developer.ErrConflict) {
		t.Fatal("duplicate member accepted")
	}
	for range authorization.MaxTestSubjects - 1 {
		if err := f.db.Ent.ClientTestSubject.Create().SetID(uuid.NewString()).SetClientID(app.ClientID).SetUserID(uuid.NewString()).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	_, another := memberFixture(t, f)
	if _, err := f.applications.ChangeTestMember(t.Context(), owner, app.ID, app.Revision, another.ID, true); !errors.Is(err, developer.ErrQuota) {
		t.Fatal("member quota bypassed")
	}
	if exists, err := f.db.Ent.EmailDelivery.Query().Where(emaildelivery.UserID(target.ID)).Exist(t.Context()); err != nil || !exists {
		t.Fatal("member notification missing")
	}
	unit, err := persistence.Begin(t.Context(), f.db.Ent)
	if err != nil {
		t.Fatal(err)
	}
	ctx := persistence.WithClient(t.Context(), unit.Client())
	if err := f.protocol.PurgeAccount(ctx, target.ID); err != nil {
		_ = unit.Rollback()
		t.Fatal(err)
	}
	if err := unit.Commit(); err != nil {
		t.Fatal(err)
	}
	if exists, err := f.db.Ent.ClientTestSubject.Query().Where(clienttestsubject.UserID(target.ID)).Exist(t.Context()); err != nil || exists {
		t.Fatal("account purge retained foreign memberships")
	}
	unit, err = persistence.Begin(t.Context(), f.db.Ent)
	if err != nil {
		t.Fatal(err)
	}
	ctx = persistence.WithClient(t.Context(), unit.Client())
	if err := f.protocol.PurgeAccount(ctx, owner.ID); err != nil {
		_ = unit.Rollback()
		t.Fatal(err)
	}
	if err := unit.Commit(); err != nil {
		t.Fatal(err)
	}
	if exists, err := f.db.Ent.ClientTestSubject.Query().Where(clienttestsubject.ClientID(app.ClientID)).Exist(t.Context()); err != nil || exists {
		t.Fatal("owner purge retained owned client memberships")
	}
}

func TestApplicationTestMembersCrossOwnerLockOrder(t *testing.T) {
	f, _, _ := managementFixture(t)
	_, other := memberFixture(t, f)
	owner, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	a, err := f.applications.Create(t.Context(), owner, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.applications.Create(t.Context(), other, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var group sync.WaitGroup
	for _, item := range []struct {
		user   identity.User
		app    developer.Application
		target string
	}{{owner, a.Application, other.ID}, {other, b.Application, owner.ID}} {
		group.Go(func() {
			if _, err := f.applications.ChangeTestMember(ctx, item.user, item.app.ID, item.app.Revision, item.target, true); err != nil {
				t.Error("cross-owner membership failed", err)
			}
		})
	}
	group.Wait()
}
