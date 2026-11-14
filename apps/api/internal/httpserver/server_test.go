package httpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"moetopia.local/aio/api/internal/authorization"
)

func TestLivenessDoesNotPretendDependenciesAreReady(t *testing.T) {
	app := New(Dependencies{DatabaseReady: func(context.Context) error { return errors.New("private-dsn must never be exposed") }, CacheReady: func(context.Context) error { return nil }})
	for _, test := range []struct {
		path   string
		status int
	}{{"/healthz", 200}, {"/readyz", 503}, {"/api/v1/status", 200}, {"/.well-known/openid-configuration", 404}, {"/.well-known/oauth-authorization-server", 404}} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.status {
			t.Fatalf("%s: got %d", test.path, response.Code)
		}
		if strings.Contains(response.Body.String(), "private-dsn") {
			t.Fatal("dependency error leaked private data")
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("response can be cached")
		}
	}
}

func TestDiscoveryPathsPublishSameConfiguredIssuer(t *testing.T) {
	const issuer = "https://account.example.test"
	provider, err := authorization.NewProvider(authorization.NewStore(nil, nil), nil, issuer, base64.StdEncoding.EncodeToString(make([]byte, 32)), nil)
	if err != nil {
		t.Fatal(err)
	}
	app := New(Dependencies{Tokens: provider})
	oidc := httptest.NewRecorder()
	app.ServeHTTP(oidc, httptest.NewRequest(http.MethodGet, "https://other.example.test/.well-known/openid-configuration", nil))
	oauth := httptest.NewRecorder()
	app.ServeHTTP(oauth, httptest.NewRequest(http.MethodGet, "https://other.example.test/.well-known/oauth-authorization-server", nil))
	if oidc.Code != http.StatusOK || oauth.Code != http.StatusOK || oidc.Body.String() != oauth.Body.String() {
		t.Fatal("discovery routes must return the same metadata")
	}
	var metadata map[string]any
	if err := json.Unmarshal(oauth.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["issuer"] != issuer || metadata["token_endpoint"] != issuer+"/oauth/token" {
		t.Fatal("discovery did not use the configured issuer")
	}
}

func TestCredentialIssuanceFailsClosedWhenLimiterIsUnavailable(t *testing.T) {
	called := false
	handler := limitCredentials(nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }), "test:")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/oauth/token", nil))
	if response.Code != 503 || called {
		t.Fatal("unavailable rate limiter allowed credential issuance")
	}
}
