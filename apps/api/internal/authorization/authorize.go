package authorization

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/sessions"
)

var ErrInteraction = errors.New("authorization interaction unavailable")
var ErrLoginRequired = errors.New("fresh authentication required")

func (p *Provider) authorizeRequest(ctx context.Context, query url.Values) (fosite.AuthorizeRequester, error) {
	client, err := p.store.GetClient(ctx, query.Get("client_id"))
	if err != nil {
		return nil, fosite.ErrInvalidClient
	}
	redirect := query.Get("redirect_uri")
	u, err := url.Parse(redirect)
	if err != nil || !slices.Contains(client.GetRedirectURIs(), redirect) || u.User != nil || u.Fragment != "" || u.Host == "" || !(u.Scheme == "https" || u.Scheme == "http" && fosite.IsLocalhost(u) && strings.HasPrefix(p.issuer, "http://")) {
		return nil, fosite.ErrInvalidRequest.WithHint("An exact registered HTTPS redirect URI is required")
	}
	r := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/oauth/authorize", RawQuery: query.Encode()}, Header: http.Header{}}
	ar, err := p.oauth.NewAuthorizeRequest(ctx, r)
	if err != nil {
		return ar, err
	}
	invalid := func() (fosite.AuthorizeRequester, error) { return ar, fosite.ErrInvalidRequest }
	for name, values := range query {
		if len(values) != 1 || !slices.Contains([]string{"client_id", "redirect_uri", "response_type", "response_mode", "scope", "audience", "state", "nonce", "code_challenge", "code_challenge_method", "prompt", "max_age"}, name) {
			return invalid()
		}
	}
	if !ar.GetResponseTypes().ExactOne("code") || !client.GetGrantTypes().Has("authorization_code") || query.Get("response_mode") != "" && query.Get("response_mode") != "query" {
		return ar, fosite.ErrUnsupportedResponseType
	}
	challenge, err := base64.RawURLEncoding.DecodeString(query.Get("code_challenge"))
	if err != nil || len(challenge) != 32 || query.Get("code_challenge_method") != "S256" {
		return invalid()
	}
	if ar.GetRequestedScopes().Has("offline_access") {
		return ar, fosite.ErrInvalidScope
	}
	for _, scope := range ar.GetRequestedScopes() {
		if !client.GetScopes().Has(scope) || !slices.Contains([]string{"openid", "profile", "email", "demo.read"}, scope) {
			return ar, fosite.ErrInvalidScope
		}
	}
	if ar.GetRequestedScopes().Has("openid") && len(query.Get("nonce")) < 8 {
		return invalid()
	}
	if !slices.Contains([]string{"", "none", "login", "consent"}, query.Get("prompt")) {
		return invalid()
	}
	if value := query.Get("max_age"); value != "" {
		if n, err := strconv.ParseUint(value, 10, 31); err != nil || n > 2147483647 {
			return invalid()
		}
	}
	audiences := ar.GetRequestedAudience()
	if len(audiences) == 0 && len(client.GetAudience()) == 1 {
		audiences = client.GetAudience()
		ar.SetRequestedAudience(audiences)
	}
	if len(audiences) != 1 {
		return invalid()
	}
	if err := fosite.DefaultAudienceMatchingStrategy(client.GetAudience(), audiences); err != nil {
		return ar, err
	}
	return ar, nil
}

func (p *Provider) browserCookie() string {
	if strings.HasPrefix(p.issuer, "https://") {
		return "__Host-moetopia_authorization"
	}
	return "moetopia_authorization"
}
func (p *Provider) BrowserToken(cookies string) string {
	r := &http.Request{Header: http.Header{"Cookie": []string{cookies}}}
	cookie, err := r.Cookie(p.browserCookie())
	if err != nil || !sessions.ValidToken(cookie.Value) {
		return ""
	}
	return cookie.Value
}

func (p *Provider) Authorize(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if len(r.URL.RawQuery) > 8192 {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	ar, err := p.authorizeRequest(r.Context(), query)
	if err != nil {
		if ar == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
			return
		}
		p.oauth.WriteAuthorizeError(r.Context(), w, ar, err)
		return
	}
	if query.Get("prompt") == "none" {
		p.oauth.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrInteractionRequired)
		return
	}
	token := p.BrowserToken(r.Header.Get("Cookie"))
	if token == "" {
		token, err = sessions.RandomToken()
		if err != nil {
			http.Error(w, "temporarily_unavailable", 503)
			return
		}
	}
	id, err := p.beginInteraction(r.Context(), token, query)
	if err != nil {
		http.Error(w, "temporarily_unavailable", 503)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: p.browserCookie(), Value: token, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(p.issuer, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 3600})
	http.Redirect(w, r, p.issuer+"/?"+url.Values{"authorization_request": {id}}.Encode(), http.StatusSeeOther)
}

func loginRequired(query url.Values, created, authenticated time.Time) bool {
	if authenticated.IsZero() {
		return true
	}
	if query.Get("prompt") == "login" && authenticated.Before(created) {
		return true
	}
	if value := query.Get("max_age"); value != "" {
		seconds, err := strconv.ParseUint(value, 10, 31)
		if err == nil && seconds == 0 {
			return authenticated.Before(created)
		}
		return err != nil || time.Since(authenticated) > time.Duration(seconds)*time.Second
	}
	return false
}
