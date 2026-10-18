package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLivenessDoesNotPretendDependenciesAreReady(t *testing.T) {
	app := New(Dependencies{DatabaseReady: func(context.Context) error { return errors.New("private-dsn must never be exposed") }, CacheReady: func(context.Context) error { return nil }})
	for _, test := range []struct {
		path   string
		status int
	}{{"/healthz", 200}, {"/readyz", 503}, {"/api/v1/status", 200}, {"/.well-known/openid-configuration", 404}} {
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

func TestCredentialIssuanceFailsClosedWhenLimiterIsUnavailable(t *testing.T) {
	called := false
	handler := limitCredentials(nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/oauth/token", nil))
	if response.Code != 503 || called {
		t.Fatal("unavailable rate limiter allowed credential issuance")
	}
}
