package httpserver

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"moetopia.local/aio/api/internal/federation"
	"moetopia.local/aio/api/internal/identity"
)

func (a FederationEndpoints) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(r.URL.RawQuery) > 8192 {
		writeJSON(w, 400, map[string]string{"code": "external_identity_invalid"})
		return
	}
	for name, values := range query {
		if len(values) != 1 {
			writeJSON(w, 400, map[string]string{"code": "external_identity_invalid"})
			return
		}
		switch name {
		case "code", "state", "error", "error_description", "error_uri", "scope", "iss", "authuser", "prompt", "hd":
		default:
			writeJSON(w, 400, map[string]string{"code": "external_identity_invalid"})
			return
		}
	}
	provider := chi.URLParam(r, "provider")
	issuer := map[string]string{"github": "https://github.com", "google": "https://accounts.google.com"}[provider]
	if issuer == "" || query.Get("iss") != "" && query.Get("iss") != issuer || (query.Get("code") == "") == (query.Get("error") == "") {
		writeJSON(w, 400, map[string]string{"code": "external_identity_invalid"})
		return
	}
	var binding *federation.Binding
	if a.Portal.Service != nil {
		user, web, err := a.Portal.Service.User(r.Context(), a.Portal.Service.CookieToken(r.Header.Get("Cookie")))
		if err == nil {
			binding = &federation.Binding{User: user, SessionID: web.ID}
		}
	}
	result, err := a.Service.Callback(r.Context(), provider, query.Get("state"), a.browser(r.Header.Get("Cookie")), query.Get("code"), query.Get("error") != "", binding)
	if err != nil {
		status, code := 503, "external_identity_unavailable"
		if problem, ok := federationError(err).(*AccountProblem); ok {
			status, code = problem.Status, problem.Code
		}
		if status == 401 || status == 422 || errors.Is(err, identity.ErrMissing) {
			status, code = 400, "external_identity_invalid"
		}
		writeJSON(w, status, map[string]string{"code": code})
		return
	}
	if result.Bound {
		http.Redirect(w, r, a.Origin+"/auth/login?federation=linked", http.StatusSeeOther)
		return
	}
	if result.Completion != "" {
		http.Redirect(w, r, a.Origin+"/auth/federation?completion="+result.Completion, http.StatusSeeOther)
		return
	}
	token, err := a.Accounts.Sessions.CreateVerified(r.Context(), result.User.ID, a.Accounts.token(r.Header.Get("Cookie")), result.User.SecurityVersion, a.Accounts.Identity)
	if err != nil {
		writeJSON(w, 503, map[string]string{"code": "external_identity_unavailable"})
		return
	}
	cookie := a.Accounts.Sessions.Cookie(token)
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, a.destination(result), http.StatusSeeOther)
}
