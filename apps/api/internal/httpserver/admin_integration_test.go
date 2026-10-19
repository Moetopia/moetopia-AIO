//go:build integration

package httpserver

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"moetopia.local/aio/api/internal/persistence/ent/adminelevation"
)

func TestAdminLoginMFAIsolationAndReplay(t *testing.T) {
	f := newAdminFixture(t)
	if state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", f.identityCookie, "", "", nil), 200); state.Authenticated {
		t.Fatal("identity cookie authenticated administrator")
	}
	cookie, csrf := f.login(t)
	if got := f.request("GET", "/api/v1/admin/applications", cookie, "", "", nil).Code; got != 403 {
		t.Fatal("first factor bypassed MFA")
	}
	if got := f.request("POST", "/api/v1/admin/mfa/enrollment", cookie, csrf, portalOrigin, map[string]string{"current_password": initialPassword}).Code; got != 403 {
		t.Fatal("user site origin can enroll administrator factor")
	}
	beforeCookie := cookie
	cookie, csrf, secret, codes := f.enroll(t, cookie, csrf)
	if state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", beforeCookie, "", "", nil), 200); state.Authenticated {
		t.Fatal("old first-factor session survived privilege elevation")
	}
	if got := f.request("GET", "/api/v1/portal/sessions", cookie, "", "", nil).Code; got != 401 {
		t.Fatal("admin cookie authenticated user-site session")
	}
	stored, err := f.db.Ent.AdminPrincipal.Get(t.Context(), f.userID)
	if err != nil || bytes.Contains(stored.Secret, []byte(secret)) || strings.Join(stored.RecoveryHashes, ",") == strings.Join(codes, ",") {
		t.Fatal("MFA credentials were stored in plaintext")
	}
	current, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := f.request("POST", "/api/v1/admin/mfa/verify", cookie, csrf, adminOrigin, map[string]string{"code": current}).Code; got != 422 {
		t.Fatal("consumed TOTP was accepted twice")
	}
	context := f.request("GET", "/api/v1/admin/context", cookie, "", "", nil)
	if bytes.Contains(context.Body.Bytes(), []byte(secret)) || bytes.Contains(context.Body.Bytes(), []byte(codes[0])) {
		t.Fatal("context leaked MFA material")
	}
	if _, err := f.db.Ent.AdminElevation.Update().Where(adminelevation.UserID(f.userID)).SetExpiresAt(time.Now().Add(-time.Second)).Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := f.request("GET", "/api/v1/admin/applications", cookie, "", "", nil).Code; got != 403 {
		t.Fatal("expired elevation grants admin access")
	}
	decodeAccount[AdminContextBody](t, f.request("POST", "/api/v1/admin/logout", cookie, csrf, adminOrigin, nil), 200)
}
func TestAdminRecoveryConsumesCodesAndRequiresReenrollment(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, csrf, oldSecret, codes := f.enroll(t, cookie, csrf)
	var group sync.WaitGroup
	type result struct {
		status       int
		cookie, csrf string
	}
	results := make(chan result, 2)
	for range 2 {
		group.Go(func() {
			response := f.request("POST", "/api/v1/admin/mfa/recover", cookie, csrf, adminOrigin, map[string]string{"current_password": initialPassword, "recovery_code": codes[0]})
			value := result{status: response.Code}
			if response.Code == 200 {
				var body AdminContextBody
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					value.status = 500
				} else {
					value.csrf = body.CSRFToken
					for _, c := range response.Result().Cookies() {
						value.cookie = c.Name + "=" + c.Value
					}
				}
			}
			results <- value
		})
	}
	group.Wait()
	close(results)
	successes := 0
	for result := range results {
		if result.status == 200 {
			successes++
			cookie, csrf = result.cookie, result.csrf
		} else if result.status != 401 && result.status != 422 {
			t.Fatalf("concurrent recovery HTTP %d", result.status)
		}
	}
	if successes != 1 {
		t.Fatal("recovery code was not consumed atomically")
	}
	state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", cookie, "", "", nil), 200)
	if !state.Authenticated || state.MFA == nil || state.MFA.Enrolled || state.MFA.Verified || state.MFA.RecoveryCodesRemaining != 0 {
		t.Fatal("recovery bypassed required reenrollment")
	}
	if got := f.request("GET", "/api/v1/admin/applications", cookie, "", "", nil).Code; got != 403 {
		t.Fatal("recovery directly granted privileges")
	}
	if got := f.request("POST", "/api/v1/admin/mfa/recover", cookie, csrf, adminOrigin, map[string]string{"current_password": initialPassword, "recovery_code": codes[1]}).Code; got != 422 {
		t.Fatal("old recovery collection survived recovery")
	}
	cookie, csrf, newSecret, newCodes := f.enroll(t, cookie, csrf)
	if newSecret == oldSecret || newCodes[0] == codes[0] {
		t.Fatal("recovery reused factor material")
	}
}
func TestAdminRoleRemovalAndMFAFailureLimit(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	for range 8 {
		if got := f.request("POST", "/api/v1/admin/mfa/verify", cookie, csrf, adminOrigin, map[string]string{"code": "abcdef"}).Code; got != 422 {
			t.Fatalf("wrong MFA HTTP %d", got)
		}
	}
	if got := f.request("POST", "/api/v1/admin/mfa/verify", cookie, csrf, adminOrigin, map[string]string{"code": "abcdef"}).Code; got != 429 {
		t.Fatal("MFA verification attempts were not limited")
	}
	if err := f.db.Ent.AdminPrincipal.UpdateOneID(f.userID).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", cookie, "", "", nil), 200)
	if state.Authenticated {
		t.Fatal("disabled role retained admin identity")
	}
}

func TestAdminRejectsNonAdministratorAndOtherRPState(t *testing.T) {
	f := newAdminFixture(t)
	userStart := f.request("GET", "/api/v1/portal/login", "", "", "", nil)
	if userStart.Code != 303 {
		t.Fatal("user RP login unavailable")
	}
	userURL, _ := url.Parse(userStart.Header().Get("Location"))
	browser := cookieValue(t, userStart)
	value := strings.SplitN(browser, "=", 2)[1]
	foreign := f.request("GET", "/api/v1/admin/callback?state="+url.QueryEscape(userURL.Query().Get("state"))+"&code=foreign-code", "__Host-moetopia_admin_browser="+value, "", "", nil)
	if foreign.Code != 400 || !bytes.Contains(foreign.Body.Bytes(), []byte("rp_login_invalid")) {
		t.Fatal("administrator callback accepted another RP transaction")
	}
	path, cookies := f.adminCallback(t)
	if err := f.db.Ent.AdminPrincipal.UpdateOneID(f.userID).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := f.request("GET", path, cookies, "", "", nil).Code; got != 401 {
		t.Fatal("OIDC callback created an administrator session without an active role")
	}
}
