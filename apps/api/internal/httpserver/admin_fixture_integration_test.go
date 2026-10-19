//go:build integration

package httpserver

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/pquerna/otp/totp"
	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
	"time"
)

const adminOrigin = "https://admin.moetopia.test"

type adminFixture struct {
	*portalFixture
	admin *administration.Service
	rp    *relyingparty.Service
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	f, _, _ := managementFixture(t)
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	box, err := cryptobox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := relyingparty.InitializeAdminClient(t.Context(), f.db.Ent, portalOrigin, adminOrigin, key); err != nil {
		t.Fatal(err)
	}
	web, err := sessions.NewAdmin(f.db.Ent, key, adminOrigin, true)
	if err != nil {
		t.Fatal(err)
	}
	service := administration.New(f.db.Ent, box, f.identity, f.outbox, web)
	rp, err := relyingparty.NewAdmin(f.db.Ent, box, service, portalOrigin, adminOrigin, key, true, f.backchannel)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.AdminPrincipal.Create().SetID(f.userID).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.handler = New(Dependencies{Tokens: f.protocol, Valkey: f.cache, Portal: PortalEndpoints{Service: f.portal}, Developers: f.applications, Admin: AdminEndpoints{RP: rp, Service: service}, Accounts: AccountEndpoints{Identity: f.identity, Sessions: f.sessions, Cache: f.cache, Authorization: f.protocol}})
	return &adminFixture{f, service, rp}
}
func (f *adminFixture) adminCallback(t *testing.T) (string, string) {
	t.Helper()
	start := f.request("GET", "/api/v1/admin/login?return_to=/applications", "", "", "", nil)
	if start.Code != 303 {
		t.Fatalf("admin login HTTP %d", start.Code)
	}
	browser := cookieValue(t, start)
	authorize, _ := url.Parse(start.Header().Get("Location"))
	if authorize.Host != "account.moetopia.test" || authorize.Query().Get("client_id") != relyingparty.AdminClientID || authorize.Query().Get("prompt") != "login" || authorize.Query().Get("max_age") != "0" || authorize.Query().Get("redirect_uri") != adminOrigin+relyingparty.AdminCallbackPath {
		t.Fatal("administrator login must request its own fresh OIDC transaction")
	}
	response := f.request("GET", authorize.RequestURI(), browser+"; "+f.identityCookie, "", "", nil)
	if response.Code != 303 {
		t.Fatalf("admin authorize HTTP %d", response.Code)
	}
	page, _ := url.Parse(response.Header().Get("Location"))
	challenge := page.Query().Get("authorization_request")
	cookies := browser + "; " + cookieValue(t, response) + "; " + f.identityCookie
	details := decodeAccount[authorization.AuthorizationDetails](t, f.request("GET", "/api/v1/account/authorizations/"+challenge, cookies, "", "", nil), 200)
	if !details.LoginRequired {
		t.Fatal("administrator OIDC request did not require fresh identity authentication")
	}
	login := f.request("POST", "/api/v1/account/login", f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"email": currentEmail(t, f.portalFixture), "password": initialPassword})
	decodeAccount[AccountContextBody](t, login, 200)
	f.identityCookie = cookieValue(t, login)
	f.identityToken = strings.SplitN(f.identityCookie, "=", 2)[1]
	cookies = browser + "; " + cookieValue(t, response) + "; " + f.identityCookie
	decision := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, f.request("POST", "/api/v1/account/authorizations/"+challenge, cookies, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"decision": "approve", "subject_id": f.userID}), 200)
	callback, _ := url.Parse(decision.RedirectTo)
	if callback.Scheme+"://"+callback.Host != adminOrigin || callback.Path != relyingparty.AdminCallbackPath {
		t.Fatal("administrator callback must target the independent admin origin")
	}
	return callback.RequestURI(), cookies
}
func (f *adminFixture) login(t *testing.T) (string, string) {
	t.Helper()
	path, cookies := f.adminCallback(t)
	response := f.request("GET", path, cookies, "", "", nil)
	if response.Code != 303 || response.Header().Get("Location") != adminOrigin+"/applications" {
		t.Fatalf("admin callback HTTP %d", response.Code)
	}
	cookie := cookieValue(t, response)
	state := decodeAccount[AdminContextBody](t, f.request("GET", "/api/v1/admin/context", cookie, "", "", nil), 200)
	if !state.Authenticated || state.MFA == nil || state.MFA.Verified {
		t.Fatal("administrator first factor must not grant elevated access")
	}
	return cookie, state.CSRFToken
}
func (f *adminFixture) enroll(t *testing.T, cookie, csrf string) (string, string, string, []string) {
	t.Helper()
	setup := decodeAccount[administration.Enrollment](t, f.request("POST", "/api/v1/admin/mfa/enrollment", cookie, csrf, adminOrigin, map[string]string{"current_password": initialPassword}), 200)
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	response := f.request("POST", "/api/v1/admin/mfa/enrollment/confirm", cookie, csrf, adminOrigin, map[string]string{"code": code})
	state := decodeAccount[AdminContextBody](t, response, 200)
	if !state.Authenticated || state.MFA == nil || !state.MFA.Enrolled || !state.MFA.Verified || len(state.RecoveryCodes) != 10 {
		t.Fatal("MFA enrollment did not complete with one-time recovery codes")
	}
	newCookie := cookieValue(t, response)
	if newCookie == cookie || state.CSRFToken == csrf {
		t.Fatal("privilege elevation did not rotate cookie and CSRF")
	}
	return newCookie, state.CSRFToken, setup.Secret, state.RecoveryCodes
}
