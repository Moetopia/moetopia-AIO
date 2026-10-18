package authorization

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/identity"
)

func (p *Provider) Metadata() any {
	return map[string]any{
		"issuer": p.issuer, "authorization_endpoint": p.issuer + "/oauth/authorize",
		"token_endpoint": p.issuer + "/oauth/token", "userinfo_endpoint": p.issuer + "/oauth/userinfo", "jwks_uri": p.issuer + "/oauth/jwks",
		"response_types_supported": []string{"code"}, "response_modes_supported": []string{"query"},
		"grant_types_supported":   []string{"authorization_code", "client_credentials"},
		"subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"code_challenge_methods_supported":      []string{"S256"}, "scopes_supported": []string{"openid", "profile", "email", "demo.read"},
		"claims_supported":            []string{"sub", "iss", "aud", "exp", "iat", "nonce", "auth_time", "at_hash", "name", "email", "email_verified"},
		"request_parameter_supported": false, "request_uri_parameter_supported": false,
	}
}

func identityClaims(user identity.User, scopes fosite.Arguments) map[string]interface{} {
	claims := map[string]interface{}{"sub": user.ID}
	if scopes.Has("profile") {
		claims["name"] = user.DisplayName
	}
	if scopes.Has("email") {
		claims["email"], claims["email_verified"] = user.Email, true
	}
	return claims
}

func (p *Provider) UserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	fail := func() {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || p.users == nil {
		fail()
		return
	}
	_, request, err := p.oauth.IntrospectToken(r.Context(), parts[1], fosite.AccessToken, newSession(p.key.ID), "openid")
	if err != nil || !request.GetGrantedScopes().Has("openid") {
		fail()
		return
	}
	user, err := p.users.User(r.Context(), request.GetSession().GetSubject())
	if err != nil {
		fail()
		return
	}
	_ = json.NewEncoder(w).Encode(identityClaims(user, request.GetGrantedScopes()))
}
