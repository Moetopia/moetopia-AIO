package httpserver

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestProxyTrustIsRestrictedToImmediatePeer(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("172.31.240.0/28")}
	for _, test := range []struct {
		peer, header, want string
		valid              bool
	}{
		{"198.51.100.1:60000", "203.0.113.1", "198.51.100.1", true},
		{"172.31.240.2:60000", "203.0.113.1", "203.0.113.1", true},
		{"172.31.240.2:60000", "", "172.31.240.2", true},
		{"172.31.240.2:60000", "203.0.113.1, 198.51.100.1", "", false},
		{"172.31.240.2:60000", "https://attacker.test", "", false},
		{"172.31.240.2:60000", "2001:db8::1", "2001:db8::1", true},
		{"bad", "203.0.113.1", "", false},
	} {
		r := httptest.NewRequest("GET", "https://issuer.example.test/", nil)
		r.RemoteAddr = test.peer
		if test.header != "" {
			r.Header.Set("X-Moetopia-Client-IP", test.header)
		}
		r.Header.Set("X-Forwarded-Host", "attacker.test")
		r.Header.Set("X-Forwarded-For", "192.0.2.99")
		got, ok := requestClientAddress(r, trusted)
		if got != test.want || ok != test.valid {
			t.Fatalf("proxy address mismatch for %s", test.peer)
		}
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.31.240.2:12345"
	r.Header.Add("X-Moetopia-Client-IP", "203.0.113.1")
	r.Header.Add("X-Moetopia-Client-IP", "203.0.113.2")
	if _, ok := requestClientAddress(r, trusted); ok {
		t.Fatal("duplicate forwarded identity accepted")
	}
}
