//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
)

func applicationConfig() developer.Config {
	return developer.Config{Name: "Test OAuth application", Description: "Integration fixture", HomepageURL: "https://client.moetopia.test", RedirectURIs: []string{"https://client.moetopia.test/callback"}, Scopes: []string{"openid", "email"}}
}
func (f *portalFixture) applicationAccess(t *testing.T, client, secret, redirect string) string {
	t.Helper()
	verifier := strings.Repeat("v", 43)
	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{"client_id": {client}, "redirect_uri": {redirect}, "response_type": {"code"}, "scope": {"openid email"}, "state": {"application-test-state"}, "nonce": {"application-test-nonce"}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}}
	start := f.request("GET", "/oauth/authorize?"+query.Encode(), f.identityCookie, "", "", nil)
	if start.Code != 303 {
		t.Fatalf("application authorize HTTP %d", start.Code)
	}
	page, _ := url.Parse(start.Header().Get("Location"))
	cookies := cookieValue(t, start) + "; " + f.identityCookie
	decision := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, f.request("POST", "/api/v1/account/authorizations/"+page.Query().Get("authorization_request"), cookies, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"decision": "approve", "subject_id": f.userID}), 200)
	callback, _ := url.Parse(decision.RedirectTo)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {callback.Query().Get("code")}, "redirect_uri": {redirect}, "code_verifier": {verifier}, "client_id": {client}}
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	r.RemoteAddr = f.ip
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if secret != "" {
		r.SetBasicAuth(client, secret)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return decodeAccount[struct {
		Access string `json:"access_token"`
	}](t, w, 200).Access
}
func (f *portalFixture) checkApplicationToken(token string) int {
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.RemoteAddr = f.ip
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w.Code
}

func TestDeveloperApplicationLifecycleAndProtocol(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	config := applicationConfig()
	created := decodeAccount[developer.Created](t, f.request("POST", "/api/v1/developer/applications", cookie, csrf, portalOrigin, map[string]any{"public": false, "config": config}), 200)
	app, secret := created.Application, created.ClientSecret
	if secret == "" || app.ActiveVersion != 0 || app.Latest.Status != "draft" {
		t.Fatal("application creation incomplete")
	}
	stored, err := f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil || !stored.TestOnly || stored.OwnerID != f.userID || bcrypt.CompareHashAndPassword(stored.SecretHash, []byte(secret)) != nil {
		t.Fatal("managed client credentials or test ownership incorrect")
	}
	details := f.request("GET", "/api/v1/developer/applications/"+app.ID, cookie, "", "", nil)
	if details.Code != 200 || bytes.Contains(details.Body.Bytes(), []byte(secret)) || bytes.Contains(details.Body.Bytes(), []byte("secret_hash")) {
		t.Fatal("application detail leaked credentials")
	}
	other, otherCookie, _ := managementFixture(t)
	if got := f.request("GET", "/api/v1/developer/applications/"+app.ID, otherCookie, "", "", nil).Code; got != 404 {
		t.Fatal("foreign application accessible")
	}
	if err := f.applications.Delete(t.Context(), identity.User{ID: other.userID}, app.ID, app.Revision); !errors.Is(err, developer.ErrMissing) {
		t.Fatal("foreign application deletion accepted")
	}
	query := url.Values{"client_id": {app.ClientID}, "redirect_uri": {config.RedirectURIs[0]}, "response_type": {"code"}, "scope": {"openid email"}, "state": {"foreign-test-state"}, "nonce": {"foreign-test-nonce"}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(make([]byte, 32))}}
	start := f.request("GET", "/oauth/authorize?"+query.Encode(), other.identityCookie, "", "", nil)
	page, _ := url.Parse(start.Header().Get("Location"))
	if got := f.request("GET", "/api/v1/account/authorizations/"+page.Query().Get("authorization_request"), cookieValue(t, start)+"; "+other.identityCookie, "", "", nil).Code; got != 403 {
		t.Fatal("non-owner can authorize a test application")
	}
	oldToken := f.applicationAccess(t, app.ClientID, secret, config.RedirectURIs[0])
	rotated := decodeAccount[struct {
		Secret string `json:"client_secret"`
	}](t, f.request("POST", "/api/v1/developer/applications/"+app.ID+"/secret", cookie, csrf, portalOrigin, map[string]any{"revision": app.Revision, "current_password": initialPassword}), 200)
	if rotated.Secret == secret || rotated.Secret == "" {
		t.Fatal("secret was not rotated")
	}
	secret = rotated.Secret
	app.Revision++
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Submit(t.Context(), user, app.ID, app.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, config); !errors.Is(err, developer.ErrConflict) {
		t.Fatal("pending review can be edited")
	}
	app, err = f.applications.Review(t.Context(), "dedicated-test-reviewer", app.ID, app.Revision, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if app.ActiveVersion != 1 || f.checkApplicationToken(oldToken) != 401 {
		t.Fatal("approval did not atomically update active state")
	}
	config.RedirectURIs = []string{"https://client.moetopia.test/new-callback"}
	app, err = f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, config)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Submit(t.Context(), user, app.ID, app.Revision)
	if err != nil {
		t.Fatal(err)
	}
	client, err := f.db.Ent.OAuthClient.Get(t.Context(), app.ClientID)
	if err != nil || client.TestOnly || !slices.Equal(client.RedirectUris, []string{"https://client.moetopia.test/callback"}) {
		t.Fatal("pending changes reached live authorization configuration")
	}
	app, err = f.applications.Review(t.Context(), "dedicated-test-reviewer", app.ID, app.Revision, false, "Keep the approved callback")
	if err != nil {
		t.Fatal(err)
	}
	if app.ActiveVersion != 1 || app.Latest.Status != "rejected" {
		t.Fatal("rejection changed approved version")
	}
	app, err = f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, config)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Submit(t.Context(), user, app.ID, app.Revision)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Review(t.Context(), "dedicated-test-reviewer", app.ID, app.Revision, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if app.ActiveVersion != 3 {
		t.Fatal("new reviewed version was not activated")
	}
	access := f.applicationAccess(t, app.ClientID, secret, config.RedirectURIs[0])
	if f.checkApplicationToken(access) != 200 {
		t.Fatal("approved client cannot authorize")
	}
	app, err = f.applications.SetEnabled(t.Context(), user, app.ID, app.Revision, false)
	if err != nil {
		t.Fatal(err)
	}
	if f.checkApplicationToken(access) != 401 {
		t.Fatal("disabled application retains access")
	}
	app, err = f.applications.SetEnabled(t.Context(), user, app.ID, app.Revision, true)
	if err != nil {
		t.Fatal(err)
	}
	if f.checkApplicationToken(access) != 401 {
		t.Fatal("reenabling application reactivated revoked grants")
	}
	app, err = f.applications.SetEnabled(t.Context(), user, app.ID, app.Revision, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.applications.Delete(t.Context(), user, app.ID, app.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.Detail(t.Context(), user.ID, app.ID); !errors.Is(err, developer.ErrMissing) {
		t.Fatal("deleted application remains available")
	}
}

func TestDeveloperQuotaAndConcurrentDraft(t *testing.T) {
	f, _, _ := managementFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	config := applicationConfig()
	created, err := f.applications.Create(t.Context(), user, true, config)
	if err != nil {
		t.Fatal(err)
	}
	statuses := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Go(func() {
			_, err := f.applications.UpdateDraft(t.Context(), user, created.Application.ID, 1, config)
			statuses <- err
		})
	}
	group.Wait()
	close(statuses)
	successes, conflicts := 0, 0
	for err := range statuses {
		if err == nil {
			successes++
		} else if errors.Is(err, developer.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("concurrent draft overwrote a newer revision")
	}
	for range 18 {
		if _, err := f.applications.Create(t.Context(), user, true, config); err != nil {
			t.Fatal(err)
		}
	}
	statuses = make(chan error, 2)
	for range 2 {
		group.Go(func() { _, err := f.applications.Create(t.Context(), user, true, config); statuses <- err })
	}
	group.Wait()
	close(statuses)
	successes = 0
	quotas := 0
	for err := range statuses {
		if err == nil {
			successes++
		} else if errors.Is(err, developer.ErrQuota) {
			quotas++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || quotas != 1 {
		t.Fatal("application quota was not serialized")
	}
}

func TestDeveloperReviewFailureRollsBack(t *testing.T) {
	f, _, _ := managementFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), user, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	app := created.Application
	app, err = f.applications.Submit(t.Context(), user, app.ID, app.Revision)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Review(t.Context(), "dedicated-test-reviewer", app.ID, app.Revision, true, "")
	if err != nil {
		t.Fatal(err)
	}
	access := f.applicationAccess(t, app.ClientID, "", applicationConfig().RedirectURIs[0])
	config := applicationConfig()
	config.Name = "Blocked-by-test"
	app, err = f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, config)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Submit(t.Context(), user, app.ID, app.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE "authorization".clients ADD CONSTRAINT test_review_failure CHECK (name <> 'Blocked-by-test') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE "authorization".clients DROP CONSTRAINT test_review_failure`); err != nil {
			t.Error("could not remove dedicated failure fixture")
		}
	})
	if _, err := f.applications.Review(t.Context(), "dedicated-test-reviewer", app.ID, app.Revision, true, ""); err == nil {
		t.Fatal("review failure fixture did not reject update")
	}
	current, err := f.applications.Detail(t.Context(), user.ID, app.ID)
	if err != nil || current.ActiveVersion != 1 || current.Latest.Status != "pending" || current.Revision != app.Revision {
		t.Fatal("failed review partially changed application state")
	}
	if f.checkApplicationToken(access) != 200 {
		t.Fatal("failed review committed credential revocation")
	}
	var serialized bytes.Buffer
	_ = json.NewEncoder(&serialized).Encode(current)
	if strings.Contains(serialized.String(), "secret_hash") {
		t.Fatal("review detail leaked credentials")
	}
}
