//go:build integration

package httpserver

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"moetopia.local/aio/api/internal/developer"
)

func TestIntrospectionHTTPContractAndClientSecretRotation(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	config := applicationConfig()
	created := decodeAccount[developer.Created](t, f.request("POST", "/api/v1/developer/applications", cookie, csrf, portalOrigin, map[string]any{"public": false, "config": config}), 200)
	app := created.Application
	token := f.applicationAccess(t, app.ClientID, created.ClientSecret, config.RedirectURIs[0])
	introspect := func(secret string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(url.Values{"token": {token}}.Encode()))
		r.RemoteAddr = f.ip
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.SetBasicAuth(app.ClientID, secret)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	result := decodeAccount[struct {
		Active   bool   `json:"active"`
		ClientID string `json:"client_id"`
	}](t, introspect(created.ClientSecret), 200)
	if !result.Active || result.ClientID != app.ClientID {
		t.Fatal("HTTP introspection did not authenticate the managed application")
	}
	rotated := decodeAccount[struct {
		Secret string `json:"client_secret"`
	}](t, f.request("POST", "/api/v1/developer/applications/"+app.ID+"/secret", cookie, csrf, portalOrigin, map[string]any{"revision": app.Revision, "current_password": initialPassword}), 200)
	if introspect(created.ClientSecret).Code != 401 {
		t.Fatal("old client secret still authenticates introspection")
	}
	decodeAccount[map[string]any](t, introspect(rotated.Secret), 200)
	metadata := decodeAccount[map[string]json.RawMessage](t, f.request("GET", "/.well-known/openid-configuration", "", "", "", nil), 200)
	var endpoint string
	var methods []string
	if err := json.Unmarshal(metadata["introspection_endpoint"], &endpoint); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(metadata["introspection_endpoint_auth_methods_supported"], &methods); err != nil {
		t.Fatal(err)
	}
	if endpoint != portalOrigin+"/oauth/introspect" || len(methods) != 1 || methods[0] != "client_secret_basic" {
		t.Fatal("discovery introspection profile incorrect")
	}
	f.handler = New(Dependencies{Tokens: f.protocol})
	if introspect(rotated.Secret).Code != 503 {
		t.Fatal("introspection bypassed unavailable rate limiting")
	}
}
