package authorization

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntrospectionRejectsInvalidEnvelopeBeforeStorage(t *testing.T) {
	p, err := NewProvider(NewStore(nil, nil), &Key{ID: "test-key"}, "https://account.moetopia.test", base64.StdEncoding.EncodeToString(make([]byte, 32)), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, method, contentType, form, authorization string
		status                                         int
	}{
		{"get", "GET", "application/x-www-form-urlencoded", "token=example", "", 400},
		{"json", "POST", "application/json", `{"token":"example"}`, "", 400},
		{"empty", "POST", "application/x-www-form-urlencoded", "token=", "", 400},
		{"duplicate", "POST", "application/x-www-form-urlencoded", "token=one&token=two", "", 400},
		{"anonymous", "POST", "application/x-www-form-urlencoded", "token=example", "", 401},
		{"bearer", "POST", "application/x-www-form-urlencoded", "token=example", "Bearer another-token", 401},
		{"mixed", "POST", "application/x-www-form-urlencoded", "token=example&access_token=other", "Basic " + base64.StdEncoding.EncodeToString([]byte("client:secret")), 401},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(test.method, "/oauth/introspect", strings.NewReader(test.form))
			r.Header.Set("Content-Type", test.contentType)
			r.Header.Set("Authorization", test.authorization)
			w := httptest.NewRecorder()
			p.Introspect(w, r)
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != test.status || body["error"] == nil || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unexpected protocol rejection: HTTP %d", w.Code)
			}
			if test.status == 401 && w.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("client authentication error omitted challenge")
			}
			if test.status == 401 && body["error"] != "invalid_client" {
				t.Fatal("client authentication error did not use the OAuth error code")
			}
		})
	}
}
