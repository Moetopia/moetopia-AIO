package httpserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/valkey-io/valkey-go"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/lifecycle"
)

type Dependencies struct {
	DatabaseReady func(context.Context) error
	CacheReady    func(context.Context) error
	Tokens        *authorization.Provider
	Valkey        valkey.Client
	Accounts      AccountEndpoints
	Portal        PortalEndpoints
	RateNamespace string
	Developers    *developer.Service
	Admin         AdminEndpoints
	Lifecycle     *lifecycle.Service
	Federation    FederationEndpoints
}

type Status struct {
	Body struct {
		Service string `json:"service"`
		Phase   string `json:"phase"`
	}
}

func New(deps Dependencies) http.Handler {
	deps.Accounts.RateNamespace = deps.RateNamespace
	deps.Admin.Accounts, deps.Admin.Developers = deps.Accounts, deps.Developers
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, accountClientIP)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-store")
			limit := int64(64 << 10)
			if req.Method == http.MethodPut && req.URL.Path == "/api/v1/portal/avatar" {
				limit = 768 << 10
			}
			req.Body = http.MaxBytesReader(w, req.Body, limit)
			req = req.WithContext(audit.WithRequest(req.Context(), middleware.GetReqID(req.Context())))
			next.ServeHTTP(w, req)
		})
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		for _, check := range []func(context.Context) error{deps.DatabaseReady, deps.CacheReady} {
			if check == nil || check(ctx) != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	if deps.Tokens != nil {
		r.Get("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, deps.Tokens.Metadata()) })
		r.Get("/oauth/authorize", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Tokens.Authorize), deps.RateNamespace))
		r.Get("/oauth/userinfo", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Tokens.UserInfo), deps.RateNamespace))
		r.Get("/oauth/jwks", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, deps.Tokens.JWKS()) })
		r.Post("/oauth/token", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Tokens.Token), deps.RateNamespace))
		r.Post("/oauth/revoke", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Tokens.Revoke), deps.RateNamespace))
		r.Post("/oauth/introspect", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Tokens.Introspect), deps.RateNamespace))
	}
	api := humachi.New(r, huma.DefaultConfig("Moetopia API", "0.1.0"))
	deps.Federation.Accounts, deps.Federation.Portal = deps.Accounts, deps.Portal
	if deps.Federation.Service != nil {
		r.Get("/api/v1/account/federation/{provider}/callback", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Federation.callback), deps.RateNamespace))
	}
	deps.Federation.Register(api)
	if deps.Portal.Service != nil {
		r.Get("/api/v1/portal/login", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Portal.Service.Login), deps.RateNamespace))
		r.Get("/api/v1/portal/callback", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Portal.Service.Callback), deps.RateNamespace))
	}
	deps.Portal.Register(api)
	if deps.Admin.RP != nil {
		r.Get("/api/v1/admin/login", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Admin.RP.Login), deps.RateNamespace))
		r.Get("/api/v1/admin/callback", limitCredentials(deps.Valkey, http.HandlerFunc(deps.Admin.RP.Callback), deps.RateNamespace))
	}
	deps.Admin.Register(api)
	deps.Portal.registerManagement(api, deps.Accounts)
	deps.Accounts.registerRecovery(api)
	registerDeletion(api, deps.Portal, deps.Accounts, deps.Lifecycle)
	DeveloperEndpoints{Portal: deps.Portal, Service: deps.Developers, Accounts: deps.Accounts}.Register(api)
	r.Get("/api/v1/avatars/{id}", deps.Accounts.avatar)
	deps.Accounts.Register(api)
	deps.Accounts.registerAuthorization(api)
	huma.Get(api, "/api/v1/status", func(context.Context, *struct{}) (*Status, error) {
		result := &Status{}
		result.Body.Service, result.Body.Phase = "moetopia", "foundation"
		return result, nil
	})
	return r
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func limitCredentials(client valkey.Client, next http.Handler, namespace string) http.HandlerFunc {
	script := valkey.NewLuaScript("local n = redis.call('INCR', KEYS[1]); if n == 1 then redis.call('EXPIRE', KEYS[1], 60) end; return n")
	return func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			writeJSON(w, 503, map[string]string{"error": "temporarily_unavailable"})
			return
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid_request"})
			return
		}
		count, err := script.Exec(r.Context(), client, []string{namespace + "limit:token:" + host}, nil).AsInt64()
		if err != nil {
			writeJSON(w, 503, map[string]string{"error": "temporarily_unavailable"})
			return
		}
		if count > 60 {
			w.Header().Set("Retry-After", "60")
			writeJSON(w, 429, map[string]string{"error": "temporarily_unavailable"})
			return
		}
		next.ServeHTTP(w, r)
	}
}
