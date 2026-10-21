//go:build integration

package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
)

func responseCookie(t *testing.T, w *httptest.ResponseRecorder, name string) string {
	t.Helper()
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == name {
			return cookie.Name + "=" + cookie.Value
		}
	}
	t.Fatal("expected browser proof cookie missing")
	return ""
}

func beginCentralLogout(t *testing.T, f *portalFixture, cookie string) (string, string) {
	t.Helper()
	u := url.Values{"client_id": {relyingparty.ClientID}, "post_logout_redirect_uri": {portalOrigin + relyingparty.LogoutCallbackPath}, "state": {"opaque-rp-state"}}
	response := f.request("GET", "/oauth/logout?"+u.Encode(), cookie, "", "", nil)
	if response.Code != 303 {
		t.Fatalf("logout start HTTP %d", response.Code)
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return location.Query().Get("logout_request"), cookie + "; " + responseCookie(t, response, "__Host-moetopia_authorization")
}

func TestCentralLogoutHTTPAndRPReturn(t *testing.T) {
	f, portalCookie, csrf := managementFixture(t)
	if f.request("POST", "/api/v1/portal/central-logout", portalCookie, csrf, "https://attacker.test", nil).Code != 403 {
		t.Fatal("RP logout bypassed origin")
	}
	startedResponse := f.request("POST", "/api/v1/portal/central-logout", portalCookie, csrf, portalOrigin, nil)
	started := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, startedResponse, 200)
	if len(startedResponse.Result().Cookies()) != 2 {
		t.Fatal("RP logout did not clear local cookie and bind browser")
	}
	proof := responseCookie(t, startedResponse, "__Host-moetopia_rp_browser")
	if _, _, err := f.portal.User(t.Context(), f.portal.CookieToken(portalCookie)); !errors.Is(err, sessions.ErrMissing) {
		t.Fatal("RP logout retained current local session")
	}
	other, err := f.sessions.Create(t.Context(), f.userID, "")
	if err != nil {
		t.Fatal(err)
	}
	central := f.request("GET", started.RedirectTo, f.identityCookie+"; "+proof, "", "", nil)
	if central.Code != 303 {
		t.Fatalf("central request HTTP %d", central.Code)
	}
	location, err := url.Parse(central.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	requestID := location.Query().Get("logout_request")
	cookies := f.identityCookie + "; " + proof + "; " + responseCookie(t, central, "__Host-moetopia_authorization")
	path := "/api/v1/account/logout-requests/" + requestID
	details := decodeAccount[struct {
		ClientID string `json:"client_id"`
		User     struct {
			ID string `json:"id"`
		} `json:"user"`
	}](t, f.request("GET", path, cookies, "", "", nil), 200)
	if details.ClientID != relyingparty.ClientID || details.User.ID != f.userID {
		t.Fatal("logout confirmation lost caller or selected user")
	}
	if f.request("GET", path, f.identityCookie, "", "", nil).Code != 404 {
		t.Fatal("logout proof crossed browsers")
	}
	body := map[string]string{"decision": "approve", "subject_id": "switched-account"}
	if f.request("POST", path, cookies, f.sessions.CSRF(f.identityToken), portalOrigin, body).Code != 409 {
		t.Fatal("logout ignored account switch")
	}
	body["subject_id"] = f.userID
	result := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, f.request("POST", path, cookies, f.sessions.CSRF(f.identityToken), portalOrigin, body), 200)
	if _, err := f.sessions.Find(t.Context(), f.identityToken); !errors.Is(err, sessions.ErrMissing) {
		t.Fatal("central logout retained selected session")
	}
	if _, err := f.sessions.Find(t.Context(), other); err != nil {
		t.Fatal("central logout removed another device")
	}
	if f.request("POST", path, cookies, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"decision": "approve", "subject_id": ""}).Code != 404 {
		t.Fatal("logout confirmation replayed")
	}
	newLocal, err := f.portal.Sessions.Create(t.Context(), f.userID, "")
	if err != nil {
		t.Fatal(err)
	}
	returned := f.request("GET", result.RedirectTo, proof+"; "+f.portal.Sessions.CookieName()+"="+newLocal, "", "", nil)
	if returned.Code != 303 || returned.Header().Get("Set-Cookie") != "" {
		t.Fatal("RP return failed or cleared a newer session")
	}
	if _, err := f.portal.Sessions.Find(t.Context(), newLocal); err != nil {
		t.Fatal("RP return revoked a newer sign-in")
	}
	if f.request("GET", result.RedirectTo, proof, "", "", nil).Code != 400 {
		t.Fatal("RP logout state replayed")
	}
}

func TestLogoutRedirectsDenialAndAtomicity(t *testing.T) {
	f, _, _ := managementFixture(t)
	for _, query := range []string{
		"client_id=moetopia-user-site&post_logout_redirect_uri=https://attacker.test/",
		"client_id=moetopia-user-site&client_id=duplicate",
		"client_id=unknown",
		"client_id=moetopia-user-site&id_token_hint=not-a-token",
	} {
		response := f.request("GET", "/oauth/logout?"+query, f.identityCookie, "", "", nil)
		if response.Code != 400 || response.Header().Get("Location") != "" {
			t.Fatal("invalid logout redirected or was accepted")
		}
	}
	id, cookies := beginCentralLogout(t, f, f.identityCookie)
	path := "/api/v1/account/logout-requests/" + id
	decodeAccount[any](t, f.request("POST", path, cookies, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"decision": "deny", "subject_id": f.userID}), 200)
	if _, err := f.sessions.Find(t.Context(), f.identityToken); err != nil {
		t.Fatal("denial ended central session")
	}
	id, cookies = beginCentralLogout(t, f, f.identityCookie)
	path = "/api/v1/account/logout-requests/" + id
	fail := true
	f.db.Ent.SecurityEvent.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			if fail {
				return nil, errors.New("injected logout audit failure")
			}
			return next.Mutate(ctx, mutation)
		})
	})
	if f.request("POST", path, cookies, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"decision": "approve", "subject_id": f.userID}).Code != 503 {
		t.Fatal("logout committed without audit")
	}
	fail = false
	if _, err := f.sessions.Find(t.Context(), f.identityToken); err != nil {
		t.Fatal("failed logout deleted session")
	}
	if _, err := f.protocol.LogoutDetails(t.Context(), id, f.protocol.BrowserToken(cookies), f.userID); err != nil {
		t.Fatal("failed logout consumed proof")
	}
	if err := f.db.Ent.OAuthClient.UpdateOneID(relyingparty.ClientID).SetPostLogoutRedirectUris([]string{}).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.db.Ent.OAuthClient.UpdateOneID(relyingparty.ClientID).SetPostLogoutRedirectUris([]string{portalOrigin + relyingparty.LogoutCallbackPath}).Exec(context.Background())
	})
	if _, err := f.protocol.LogoutDetails(t.Context(), id, f.protocol.BrowserToken(cookies), f.userID); !errors.Is(err, authorization.ErrLogout) {
		t.Fatal("removed redirect remained usable")
	}
	if _, err := f.sessions.Find(t.Context(), f.identityToken); err != nil {
		t.Fatal("invalidated request ended session")
	}
	if strings.Contains(f.request("GET", "/oauth/logout?id_token_hint=invalid", cookies, "", "", nil).Body.String(), "id_token_hint") {
		t.Fatal("logout error echoed credential parameter")
	}
}

func TestLogoutRedirectChangesRequirePublication(t *testing.T) {
	f, _, _ := managementFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	config := applicationConfig()
	oldRedirect := "https://client.moetopia.test/old-logout"
	newRedirect := "https://client.moetopia.test/new-logout"
	config.PostLogoutRedirectURIs = []string{oldRedirect}
	created, err := f.applications.Create(t.Context(), user, true, config)
	if err != nil {
		t.Fatal(err)
	}
	app, err := f.applications.Submit(t.Context(), user, created.Application.ID, created.Application.Revision)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Review(t.Context(), "synthetic-reviewer", app.ID, app.Revision, true, "")
	if err != nil {
		t.Fatal(err)
	}
	check := func(redirect string, expected int) {
		t.Helper()
		u := url.Values{"client_id": {app.ClientID}, "post_logout_redirect_uri": {redirect}, "ui_locales": {"zz-unknown"}}
		if result := f.request("GET", "/oauth/logout?"+u.Encode(), f.identityCookie, "", "", nil); result.Code != expected {
			t.Fatalf("logout target status %d expected %d", result.Code, expected)
		}
	}
	config.PostLogoutRedirectURIs = []string{newRedirect}
	app, err = f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, config)
	if err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.Submit(t.Context(), user, app.ID, app.Revision)
	if err != nil {
		t.Fatal(err)
	}
	check(oldRedirect, 303)
	check(newRedirect, 400)
	app, err = f.applications.Review(t.Context(), "synthetic-reviewer", app.ID, app.Revision, true, "")
	if err != nil {
		t.Fatal(err)
	}
	check(oldRedirect, 400)
	check(newRedirect, 303)
	metadataOnly := config
	metadataOnly.PostLogoutRedirectURIs = nil
	metadataOnly.Description = "Metadata-only update"
	app, err = f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, metadataOnly)
	if err != nil || len(app.Latest.Config.PostLogoutRedirectURIs) != 1 || app.Latest.Config.PostLogoutRedirectURIs[0] != newRedirect {
		t.Fatal("older client omitted field and erased logout registration")
	}
	cleared := config
	cleared.PostLogoutRedirectURIs = []string{}
	app, err = f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, cleared)
	if err != nil || len(app.Latest.Config.PostLogoutRedirectURIs) != 0 {
		t.Fatal("explicit empty logout list did not clear draft")
	}
	check(newRedirect, 303)
	invalid := config
	invalid.PostLogoutRedirectURIs = []string{"https://client.moetopia.test/ok?redirect=https://attacker.test"}
	if _, err := f.applications.UpdateDraft(t.Context(), user, app.ID, app.Revision, invalid); !errors.Is(err, developer.ErrInvalid) {
		t.Fatal("logout registration accepted query redirect")
	}
}

func TestConcurrentCentralLogoutIsSingleUse(t *testing.T) {
	f, _, _ := managementFixture(t)
	id, cookies := beginCentralLogout(t, f, f.identityCookie)
	web, err := f.sessions.Find(t.Context(), f.identityToken)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	start := make(chan struct{})
	for range 2 {
		wait.Go(func() {
			<-start
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			_, err := f.protocol.ConfirmLogout(ctx, id, f.protocol.BrowserToken(cookies), f.userID, web.ID, true, f.sessions)
			results <- err
		})
	}
	close(start)
	wait.Wait()
	close(results)
	success, stale := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, sessions.ErrMissing) || errors.Is(err, authorization.ErrLogout) {
			stale++
		} else {
			t.Fatalf("concurrent logout failed unexpectedly: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatal("logout proof was not single-use")
	}
}

func TestAnonymousLogoutAndFormInitiation(t *testing.T) {
	f, _, _ := managementFixture(t)
	contextResponse := f.request("GET", "/api/v1/account/context", "", "", "", nil)
	anonymous := decodeAccount[AccountContextBody](t, contextResponse, 200)
	cookie := cookieValue(t, contextResponse)
	form := url.Values{"client_id": {relyingparty.ClientID}, "post_logout_redirect_uri": {portalOrigin + relyingparty.LogoutCallbackPath}, "state": {"opaque-state"}, "ui_locales": {"zz-unknown"}}
	r := httptest.NewRequest(http.MethodPost, "/oauth/logout", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Cookie", cookie)
	r.RemoteAddr = f.ip
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != 303 {
		t.Fatalf("form logout HTTP %d", w.Code)
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	id := location.Query().Get("logout_request")
	cookies := cookie + "; " + responseCookie(t, w, "__Host-moetopia_authorization")
	path := "/api/v1/account/logout-requests/" + id
	details := decodeAccount[LogoutDetailsOutputBody](t, f.request("GET", path, cookies, "", "", nil), 200)
	if details.User != nil {
		t.Fatal("anonymous logout disclosed an unrelated account")
	}
	decodeAccount[any](t, f.request("POST", path, cookies, anonymous.CSRFToken, portalOrigin, map[string]string{"decision": "approve", "subject_id": ""}), 200)
	if _, err := f.sessions.Find(t.Context(), f.identityToken); err != nil {
		t.Fatal("anonymous logout ended another browser session")
	}
}

type LogoutDetailsOutputBody struct {
	User *struct {
		ID string `json:"id"`
	} `json:"user"`
}
