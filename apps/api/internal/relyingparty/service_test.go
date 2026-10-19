package relyingparty

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReturnPathAndQueryRejectNavigationInjection(t *testing.T) {
	for _, value := range []string{"https://evil.test", "//evil.test", "/\\evil.test", "/profile?next=evil", "/profile#fragment", "/auth/login", "/api/v1/portal/login", "/developer/../auth/login", "/%2f%2fevil.test"} {
		if _, err := ReturnPath(value); err == nil {
			t.Fatalf("unapproved return path accepted: %q", value)
		}
	}
	for _, value := range []string{"", "/profile", "/sessions", "/security", "/developer", "/developer/applications", "/developer/docs"} {
		if _, err := ReturnPath(value); err != nil {
			t.Fatalf("approved path rejected: %q", value)
		}
	}
	for _, raw := range []string{"state=one&state=two", "state=%zz", "return_to=/profile;state=one"} {
		if _, err := parseQuery(httptest.NewRequest("GET", "/?"+raw, nil)); err == nil {
			t.Fatal("ambiguous query accepted")
		}
	}
}

func TestBackchannelCannotReachArbitraryDestinations(t *testing.T) {
	called := false
	transport := protocolTransport{"https://account.moetopia.test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) })}
	for _, target := range []string{"https://evil.test/oauth/jwks", "https://user:password@account.moetopia.test/oauth/jwks", "https://account.moetopia.test/api/v1/account/context", "https://account.moetopia.test/oauth/jwks?target=evil", "http://account.moetopia.test/oauth/jwks"} {
		request := httptest.NewRequest("GET", target, nil)
		if _, err := transport.RoundTrip(request); err == nil {
			t.Fatal("untrusted backchannel accepted")
		}
	}
	if called {
		t.Fatal("untrusted requests reached protocol handler")
	}
	if _, err := transport.RoundTrip(httptest.NewRequest("GET", "https://account.moetopia.test/oauth/jwks", nil)); err != nil || !called {
		t.Fatal("trusted JWKS request failed")
	}
}
