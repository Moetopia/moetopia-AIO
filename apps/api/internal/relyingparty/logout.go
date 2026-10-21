package relyingparty

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"time"

	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/persistence/ent/rplogin"
	"moetopia.local/aio/api/internal/sessions"
)

type LogoutStart struct {
	RedirectTo string
	Cookies    []http.Cookie
}
type logoutPayload struct {
	ClientID string `json:"client_id"`
}

func (s *Service) logoutCallback() string {
	if s.admin {
		return s.origin + AdminLogoutCallbackPath
	}
	return s.origin + LogoutCallbackPath
}

func (s *Service) BeginLogout(ctx context.Context, token, cookies string) (LogoutStart, error) {
	web, err := s.Sessions.Find(ctx, token)
	if err != nil {
		return LogoutStart{}, err
	}
	state, err := sessions.RandomToken()
	if err != nil {
		return LogoutStart{}, err
	}
	browser := ""
	req := &http.Request{Header: http.Header{"Cookie": []string{cookies}}}
	if cookie, err := req.Cookie(s.browserCookie()); err == nil && sessions.ValidToken(cookie.Value) {
		browser = cookie.Value
	}
	if browser == "" {
		browser, err = sessions.RandomToken()
		if err != nil {
			return LogoutStart{}, err
		}
	}
	id := digest("rp-logout-state:" + state)
	plaintext, _ := json.Marshal(logoutPayload{s.clientID})
	encrypted, err := s.box.Seal(plaintext, "rp-logout:"+id)
	if err != nil {
		return LogoutStart{}, err
	}
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return LogoutStart{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := s.Sessions.RevokeLive(ctx, web.UserID, web.ID); err != nil {
		return LogoutStart{}, err
	}
	client, err := unit.OAuthClient.Query().Where(oauthclient.ID(s.clientID)).ForShare().Only(ctx)
	if err != nil {
		return LogoutStart{}, err
	}
	if err := s.currentClient(ctx); err != nil {
		return LogoutStart{}, err
	}
	if !slices.Equal(client.PostLogoutRedirectUris, []string{s.logoutCallback()}) {
		return LogoutStart{}, ErrLogin
	}
	if err := unit.RPLogin.Create().SetID(id).SetBrowserHash(digest("rp-logout-browser:" + browser)).SetPayload(encrypted).SetExpiresAt(time.Now().Add(5 * time.Minute)).Exec(ctx); err != nil {
		return LogoutStart{}, err
	}
	if err := unit.Commit(); err != nil {
		return LogoutStart{}, err
	}
	cleared := s.Sessions.Cookie("")
	cleared.MaxAge = -1
	proof := http.Cookie{Name: s.browserCookie(), Value: browser, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 3600}
	parameters := url.Values{"client_id": {s.clientID}, "post_logout_redirect_uri": {s.logoutCallback()}, "state": {state}}
	return LogoutStart{RedirectTo: s.issuer + "/oauth/logout?" + parameters.Encode(), Cookies: []http.Cookie{cleared, proof}}, nil
}

func (s *Service) LogoutCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	query, err := parseQuery(r)
	if err != nil || len(query) != 1 || !sessions.ValidToken(query.Get("state")) {
		s.Problem(w, 400, "rp_logout_invalid")
		return
	}
	proof, err := r.Cookie(s.browserCookie())
	if err != nil || !sessions.ValidToken(proof.Value) {
		s.Problem(w, 400, "rp_logout_invalid")
		return
	}
	id := digest("rp-logout-state:" + query.Get("state"))
	unit, err := persistence.Begin(r.Context(), s.db)
	if err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	defer unit.Rollback()
	row, err := unit.RPLogin.Query().Where(rplogin.ID(id)).ForUpdate().Only(r.Context())
	if ent.IsNotFound(err) {
		s.Problem(w, 400, "rp_logout_invalid")
		return
	}
	if err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	if row.Consumed || !time.Now().Before(row.ExpiresAt) || subtle.ConstantTimeCompare([]byte(row.BrowserHash), []byte(digest("rp-logout-browser:"+proof.Value))) != 1 {
		s.Problem(w, 400, "rp_logout_invalid")
		return
	}
	plaintext, err := s.box.Open(row.Payload, "rp-logout:"+id)
	var data logoutPayload
	if err != nil || json.Unmarshal(plaintext, &data) != nil || data.ClientID != s.clientID {
		s.Problem(w, 400, "rp_logout_invalid")
		return
	}
	if err := unit.RPLogin.UpdateOneID(id).SetConsumed(true).SetPayload([]byte{}).Exec(r.Context()); err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	if err := unit.Commit(); err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	path := "/auth/logout?central=1"
	if s.admin {
		path = "/"
	}
	http.Redirect(w, r, s.origin+path, http.StatusSeeOther)
}
