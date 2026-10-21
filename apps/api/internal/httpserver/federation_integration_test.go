//go:build integration

package httpserver

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/federation"
)

type externalTransport func(*http.Request) (*http.Response, error)

func (f externalTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFederationHTTPBrowserFlow(t *testing.T) {
	f := newAccountFixture(t)
	original := http.DefaultTransport
	http.DefaultTransport = externalTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.String() {
		case "https://github.com/login/oauth/access_token":
			body = `{"access_token":"upstream-test-token","token_type":"bearer"}`
		case "https://api.github.com/user":
			body = fmt.Sprintf(`{"id":%d,"login":"test-subject","name":"Provider name"}`, time.Now().UnixNano())
		case "https://api.github.com/user/emails":
			body = `[{"email":"hint@example.test","primary":true,"verified":true}]`
		default:
			t.Fatal("unexpected external endpoint")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	provider, err := federation.New("github", portalOrigin+"/api/v1/account/federation/github/callback", federation.Credentials{ClientID: "test-client", ClientSecret: "test-secret"})
	http.DefaultTransport = original
	if err != nil {
		t.Fatal(err)
	}
	box, err := cryptobox.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	service := federation.NewService(f.db.Ent, box, f.identity, map[string]*federation.Provider{"github": provider}, nil)
	f.handler = New(Dependencies{Valkey: f.cache, Accounts: AccountEndpoints{Identity: f.identity, Sessions: f.sessions, Cache: f.cache}, Federation: FederationEndpoints{Service: service, Origin: portalOrigin}})
	providers := decodeAccount[struct {
		Items []string `json:"items"`
	}](t, f.request("GET", "/api/v1/account/federation/providers", "", "", "", nil), 200)
	if len(providers.Items) != 1 || providers.Items[0] != "github" {
		t.Fatal("unconfigured providers advertised")
	}
	contextResponse := f.request("GET", "/api/v1/account/context", "", "", "", nil)
	anonymous := decodeAccount[AccountContextBody](t, contextResponse, 200)
	cookie := cookieValue(t, contextResponse)
	if f.request("POST", "/api/v1/account/federation/github", cookie, anonymous.CSRFToken, "https://attacker.test", map[string]string{}).Code != 403 {
		t.Fatal("federation start bypassed CSRF origin")
	}
	startedResponse := f.request("POST", "/api/v1/account/federation/github", cookie, anonymous.CSRFToken, portalOrigin, map[string]string{})
	started := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, startedResponse, 200)
	upstream, err := url.Parse(started.RedirectTo)
	if err != nil || upstream.Host != "github.com" {
		t.Fatal("invalid upstream redirect")
	}
	federationCookie := cookieValue(t, startedResponse)
	allCookies := cookie + "; " + federationCookie
	callback := "/api/v1/account/federation/github/callback?state=" + upstream.Query().Get("state") + "&code=upstream-code"
	if f.request("GET", callback, cookie, "", "", nil).Code != 400 {
		t.Fatal("callback did not require browser proof")
	}
	if f.request("GET", callback+"&state=duplicate", allCookies, "", "", nil).Code != 400 {
		t.Fatal("duplicate callback parameters accepted")
	}
	callbackResponse := f.request("GET", callback, allCookies, "", "", nil)
	if callbackResponse.Code != 303 || callbackResponse.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("external callback failed: %d", callbackResponse.Code)
	}
	completionURL, err := url.Parse(callbackResponse.Header().Get("Location"))
	if err != nil || completionURL.Path != "/auth/federation" || strings.Contains(completionURL.String(), "upstream") || strings.Contains(completionURL.String(), "hint@") {
		t.Fatal("registration redirect exposed credentials or profile")
	}
	completion := completionURL.Query().Get("completion")
	decodeAccount[federation.RegistrationDetails](t, f.request("GET", "/api/v1/account/federation/registration?completion="+completion, allCookies, "", "", nil), 200)
	if f.request("GET", callback, allCookies, "", "", nil).Code != 400 {
		t.Fatal("HTTP callback replayed")
	}
	if f.request("POST", "/api/v1/account/federation/google", cookie, anonymous.CSRFToken, portalOrigin, map[string]string{}).Code != 503 {
		t.Fatal("unconfigured provider accepted login")
	}
}
