package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"moetopia.local/aio/api/internal/authorization"
)

type exportedAPI struct {
	Paths map[string]*huma.PathItem `json:"paths"`
}

func exportedDocument(t *testing.T) *exportedAPI {
	t.Helper()
	response := httptest.NewRecorder()
	New(Dependencies{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if response.Code != 200 {
		t.Fatalf("document status: %d", response.Code)
	}
	var doc exportedAPI
	if err := json.Unmarshal(response.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	return &doc
}

func TestOpenAPIBusinessTitlesAndMFAInstructions(t *testing.T) {
	doc := exportedDocument(t)
	ids := map[string]bool{}
	for path, item := range doc.Paths {
		for method, op := range map[string]*huma.Operation{"GET": item.Get, "POST": item.Post, "PUT": item.Put, "PATCH": item.Patch, "DELETE": item.Delete} {
			if op == nil {
				continue
			}
			if op.Summary == "" || len(op.Tags) != 1 || ids[op.OperationID] || op.OperationID == "" {
				t.Fatalf("missing title/tag or conflicting operationId: %s %s", method, path)
			}
			ids[op.OperationID] = true
			if strings.HasPrefix(path, "/api/v1/") && (businessSummaries[method+" "+path] == "" || op.Summary != businessSummaries[method+" "+path]) {
				t.Fatalf("business operation needs an explicit title: %s %s", method, path)
			}
		}
	}
	op := doc.Paths["/api/v1/admin/mfa/enrollment"].Post
	if !strings.Contains(op.Description, "当前 Moetopia 账号的本地登录密码") || !strings.Contains(op.Description, "enrollment/confirm") {
		t.Fatal("enrollment must explain the password and confirmation step")
	}
	if op.OperationID != "post-api-v1-admin-mfa-enrollment" {
		t.Fatal("business operationId changed")
	}
}

func TestOpenAPIProtocolContracts(t *testing.T) {
	doc := exportedDocument(t)
	for path, methods := range map[string][]string{
		"/.well-known/openid-configuration": {"GET"}, "/oauth/jwks": {"GET"}, "/oauth/authorize": {"GET"},
		"/oauth/userinfo": {"GET"}, "/oauth/token": {"POST"}, "/oauth/revoke": {"POST"},
		"/oauth/introspect": {"POST"}, "/oauth/logout": {"GET", "POST"},
	} {
		item := doc.Paths[path]
		if item == nil {
			t.Fatalf("missing protocol path: %s", path)
		}
		for _, method := range methods {
			op := item.Get
			if method == "POST" {
				op = item.Post
				if op == nil || op.RequestBody == nil || op.RequestBody.Content["application/x-www-form-urlencoded"] == nil || op.RequestBody.Content["application/json"] != nil {
					t.Fatalf("protocol POST must describe form inputs: %s", path)
				}
			}
			if op == nil || len(op.Responses) == 0 {
				t.Fatalf("missing protocol method: %s %s", method, path)
			}
		}
	}
	if doc.Paths["/oauth/revoke"].Post.Responses["200"].Content != nil {
		t.Fatal("revocation success must have an empty response body")
	}
	introspect := doc.Paths["/oauth/introspect"].Post
	if len(introspect.Security) != 1 || introspect.Security[0]["OAuthClientBasic"] == nil {
		t.Fatal("introspection requires confidential HTTP Basic authentication")
	}
	if doc.Paths["/oauth/userinfo"].Get.Security[0]["OAuthAccessToken"] == nil {
		t.Fatal("userinfo requires an access token")
	}
	if doc.Paths["/oauth/logout"].Get.Responses["303"].Headers["Location"] == nil {
		t.Fatal("logout must describe browser redirects")
	}
	first := httptest.NewRecorder()
	second := httptest.NewRecorder()
	app := New(Dependencies{})
	app.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	New(Dependencies{}).ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if first.Body.String() != second.Body.String() {
		t.Fatal("OpenAPI output must be deterministic")
	}
}

func TestOpenAPICoversRegisteredProtocolRoutes(t *testing.T) {
	doc := exportedDocument(t)
	router := New(Dependencies{Tokens: &authorization.Provider{}}).(chi.Routes)
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/oauth/") && route != "/.well-known/openid-configuration" {
			return nil
		}
		item := doc.Paths[route]
		if item == nil || method == http.MethodGet && item.Get == nil || method == http.MethodPost && item.Post == nil {
			t.Errorf("registered protocol route is undocumented: %s %s", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
