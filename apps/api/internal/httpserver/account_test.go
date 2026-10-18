package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountValidationDoesNotEchoCredentials(t *testing.T) {
	body := `{"challenge_id":"test","code":"00000000","invitation":"private-invitation","display_name":"Reader","password":"secret"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/account/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	New(Dependencies{}).ServeHTTP(w, r)
	if w.Code != 422 {
		t.Fatalf("expected validation failure, got %d", w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte(`"secret"`)) || bytes.Contains(w.Body.Bytes(), []byte("private-invitation")) {
		t.Fatal("validation response echoed credentials")
	}
}
