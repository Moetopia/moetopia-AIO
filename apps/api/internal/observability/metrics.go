package observability

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

var buckets = []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5}
var routes = []string{"token", "authorize", "userinfo", "introspect", "revoke", "logout", "login", "registration", "admin", "account", "portal", "developer", "metadata", "health", "other"}
var jobs = []string{"delivery", "administration", "lifecycle", "identity", "sessions", "authorization", "relying_party", "other"}

type sample struct {
	count, failures int64
	seconds         float64
	buckets         [8]int64
}
type Registry struct {
	mu       sync.Mutex
	requests map[string]*sample
	tasks    map[string]*sample
}

func New() *Registry {
	r := &Registry{requests: map[string]*sample{}, tasks: map[string]*sample{}}
	for _, route := range routes {
		r.requests[route] = &sample{}
	}
	for _, job := range jobs {
		r.tasks[job] = &sample{}
	}
	return r
}
func routeGroup(path string) string {
	switch path {
	case "/oauth/token":
		return "token"
	case "/oauth/authorize":
		return "authorize"
	case "/oauth/userinfo":
		return "userinfo"
	case "/oauth/introspect":
		return "introspect"
	case "/oauth/revoke":
		return "revoke"
	case "/oauth/logout":
		return "logout"
	case "/api/v1/account/login":
		return "login"
	case "/api/v1/account/register":
		return "registration"
	case "/.well-known/openid-configuration", "/oauth/jwks":
		return "metadata"
	case "/healthz", "/readyz":
		return "health"
	}
	for _, group := range []string{"admin", "account", "portal", "developer"} {
		if strings.HasPrefix(path, "/api/v1/"+group+"/") {
			return group
		}
	}
	return "other"
}

func (r *Registry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		started := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, request.ProtoMajor)
		defer func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			s := r.requests[routeGroup(request.URL.Path)]
			seconds := time.Since(started).Seconds()
			s.count++
			s.seconds += seconds
			if wrapped.Status() >= 400 {
				s.failures++
			}
			for i, limit := range buckets {
				if seconds <= limit {
					s.buckets[i]++
				}
			}
		}()
		next.ServeHTTP(wrapped, request)
	})
}

func (r *Registry) ObserveJob(job string, elapsed time.Duration, failed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.tasks[job]
	if !ok {
		s = r.tasks["other"]
	}
	s.count++
	s.seconds += elapsed.Seconds()
	if failed {
		s.failures++
	}
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	fmt.Fprintf(w, "# TYPE moetopia_go_heap_bytes gauge\nmoetopia_go_heap_bytes %d\n# TYPE moetopia_go_goroutines gauge\nmoetopia_go_goroutines %d\n", memory.HeapAlloc, runtime.NumGoroutine())
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Fprint(w, "# TYPE moetopia_http_requests_total counter\n# TYPE moetopia_http_failures_total counter\n# TYPE moetopia_http_request_duration_seconds histogram\n")
	for _, route := range routes {
		s := r.requests[route]
		fmt.Fprintf(w, "moetopia_http_requests_total{route=%q} %d\nmoetopia_http_failures_total{route=%q} %d\n", route, s.count, route, s.failures)
		for i, limit := range buckets {
			fmt.Fprintf(w, "moetopia_http_request_duration_seconds_bucket{route=%q,le=%q} %d\n", route, fmt.Sprint(limit), s.buckets[i])
		}
		fmt.Fprintf(w, "moetopia_http_request_duration_seconds_bucket{route=%q,le=\"+Inf\"} %d\nmoetopia_http_request_duration_seconds_count{route=%q} %d\nmoetopia_http_request_duration_seconds_sum{route=%q} %g\n", route, s.count, route, s.count, route, s.seconds)
	}
	fmt.Fprint(w, "# TYPE moetopia_job_runs_total counter\n# TYPE moetopia_job_failures_total counter\n# TYPE moetopia_job_duration_seconds_total counter\n")
	for _, job := range jobs {
		s := r.tasks[job]
		fmt.Fprintf(w, "moetopia_job_runs_total{job=%q} %d\nmoetopia_job_failures_total{job=%q} %d\nmoetopia_job_duration_seconds_total{job=%q} %g\n", job, s.count, job, s.failures, job, s.seconds)
	}
}
