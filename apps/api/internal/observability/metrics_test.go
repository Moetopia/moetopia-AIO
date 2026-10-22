package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMetricsBoundedLabelsAndConcurrentRecording(t *testing.T) {
	r := New()
	handler := r.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(401) }))
	var group sync.WaitGroup
	for range 100 {
		group.Go(func() {
			request := httptest.NewRequest("GET", "http://example.test/api/v1/account/private-user?email=private@example.test", nil)
			request.Header.Set("Authorization", "Bearer private-token")
			handler.ServeHTTP(httptest.NewRecorder(), request)
		})
	}
	group.Wait()
	r.ObserveJob("delivery", time.Second, true)
	r.ObserveJob("private-user-label", time.Second, false)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest("GET", "/internal/metrics", nil))
	text := response.Body.String()
	if !strings.Contains(text, `moetopia_http_requests_total{route="account"} 100`) || !strings.Contains(text, `moetopia_http_failures_total{route="account"} 100`) || !strings.Contains(text, `moetopia_job_failures_total{job="delivery"} 1`) {
		t.Fatal("metrics counts incorrect")
	}
	for _, secret := range []string{"private-user", "private@example.test", "private-token"} {
		if strings.Contains(text, secret) {
			t.Fatal("metrics exposed unbounded or private labels")
		}
	}
	if len(r.requests) != len(routes) || len(r.tasks) != len(jobs) {
		t.Fatal("unbounded metric cardinality")
	}
}
