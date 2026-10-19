package relyingparty

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/rplogin"
	"moetopia.local/aio/api/internal/sessions"
)

type loginPayload struct {
	Verifier  string `json:"verifier"`
	Nonce     string `json:"nonce"`
	ReturnTo  string `json:"return_to"`
	ClientID  string `json:"client_id"`
	StartedAt int64  `json:"started_at"`
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	query, err := parseQuery(r)
	if err != nil || len(query) > 1 {
		s.Problem(w, 400, "rp_return_path_invalid")
		return
	}
	for name := range query {
		if name != "return_to" {
			s.Problem(w, 400, "rp_return_path_invalid")
			return
		}
	}
	path, err := s.returnPath(query.Get("return_to"))
	if err != nil {
		s.Problem(w, 400, "rp_return_path_invalid")
		return
	}
	if err := s.currentClient(r.Context()); err != nil {
		s.Problem(w, 503, "rp_client_unavailable")
		return
	}
	state, err := sessions.RandomToken()
	if err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	nonce, err := sessions.RandomToken()
	if err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	browser := ""
	if cookie, err := r.Cookie(s.browserCookie()); err == nil && sessions.ValidToken(cookie.Value) {
		browser = cookie.Value
	}
	if browser == "" {
		browser, err = sessions.RandomToken()
		if err != nil {
			s.Problem(w, 503, "rp_unavailable")
			return
		}
	}
	verifier := oauth2.GenerateVerifier()
	payload, _ := json.Marshal(loginPayload{verifier, nonce, path, s.clientID, time.Now().Unix()})
	id := digest("rp-state:" + state)
	encrypted, err := s.box.Seal(payload, "rp-login:"+id)
	if err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	if err := s.db.RPLogin.Create().SetID(id).SetBrowserHash(digest("rp-browser:" + browser)).SetPayload(encrypted).SetExpiresAt(time.Now().Add(5 * time.Minute)).Exec(r.Context()); err != nil {
		s.Problem(w, 503, "rp_unavailable")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: s.browserCookie(), Value: browser, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 3600})
	options := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce), oauth2.SetAuthURLParam("audience", s.issuer+"/api")}
	if s.admin {
		options = append(options, oauth2.SetAuthURLParam("prompt", "login"), oauth2.SetAuthURLParam("max_age", "0"))
	}
	http.Redirect(w, r, s.oauth.AuthCodeURL(state, options...), http.StatusSeeOther)
}

func (s *Service) consume(ctx context.Context, r *http.Request, state string) (loginPayload, error) {
	var payload loginPayload
	if !sessions.ValidToken(state) {
		return payload, ErrLogin
	}
	cookie, err := r.Cookie(s.browserCookie())
	if err != nil || !sessions.ValidToken(cookie.Value) {
		return payload, ErrLogin
	}
	id := digest("rp-state:" + state)
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return payload, err
	}
	defer tx.Rollback()
	row, err := tx.RPLogin.Query().Where(rplogin.ID(id)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return payload, ErrLogin
	}
	if err != nil {
		return payload, err
	}
	if row.Consumed || !time.Now().Before(row.ExpiresAt) {
		return payload, ErrLogin
	}
	if subtle.ConstantTimeCompare([]byte(row.BrowserHash), []byte(digest("rp-browser:"+cookie.Value))) != 1 {
		return payload, ErrLogin
	}
	plaintext, err := s.box.Open(row.Payload, "rp-login:"+id)
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return payload, err
	}
	if payload.ClientID != s.clientID || payload.StartedAt <= 0 {
		return payload, ErrLogin
	}
	if _, err := s.returnPath(payload.ReturnTo); err != nil {
		return payload, err
	}
	if err := tx.RPLogin.UpdateOneID(id).SetConsumed(true).SetPayload([]byte{}).Exec(ctx); err != nil {
		return payload, err
	}
	return payload, tx.Commit()
}

func (s *Service) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	query, err := parseQuery(r)
	if err != nil {
		s.Problem(w, 400, "rp_callback_invalid")
		return
	}
	for name := range query {
		if name != "state" && name != "code" && name != "scope" && name != "error" && name != "error_description" && name != "error_uri" && name != "iss" {
			s.Problem(w, 400, "rp_callback_invalid")
			return
		}
	}
	if query.Get("iss") != "" && query.Get("iss") != s.issuer || (query.Get("code") == "") == (query.Get("error") == "") {
		s.Problem(w, 400, "rp_callback_invalid")
		return
	}
	payload, err := s.consume(r.Context(), r, query.Get("state"))
	if err != nil {
		if errors.Is(err, ErrLogin) {
			s.Problem(w, 400, "rp_login_invalid")
		} else {
			s.Problem(w, 503, "rp_unavailable")
		}
		return
	}
	if query.Get("error") != "" {
		s.Problem(w, 400, "rp_authorization_denied")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	ctx = oidc.ClientContext(ctx, s.client)
	token, err := s.oauth.Exchange(ctx, query.Get("code"), oauth2.VerifierOption(payload.Verifier))
	if err != nil {
		var failed *oauth2.RetrieveError
		if ctx.Err() != nil || errors.As(err, &failed) && failed.Response != nil && failed.Response.StatusCode >= 500 {
			s.Problem(w, 503, "rp_unavailable")
			return
		}
		s.Problem(w, 400, "rp_exchange_failed")
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || token.AccessToken == "" {
		s.Problem(w, 400, "rp_identity_invalid")
		return
	}
	idToken, err := s.verifier.Verify(ctx, raw)
	if err != nil {
		s.Problem(w, 400, "rp_identity_invalid")
		return
	}
	var claims struct {
		AuthorizedParty string `json:"azp"`
		AuthTime        int64  `json:"auth_time"`
		SecurityVersion int    `json:"security_version"`
	}
	if err := idToken.Claims(&claims); err != nil || len(idToken.Audience) != 1 || claims.AuthorizedParty != "" && claims.AuthorizedParty != s.clientID || idToken.Subject == "" || idToken.Nonce != payload.Nonce || idToken.IssuedAt.IsZero() || idToken.IssuedAt.After(time.Now().Add(time.Minute)) || idToken.IssuedAt.Before(time.Now().Add(-6*time.Minute)) || claims.AuthTime <= 0 || claims.AuthTime > time.Now().Add(time.Minute).Unix() || s.admin && claims.AuthTime < payload.StartedAt-1 || idToken.AccessTokenHash == "" || idToken.VerifyAccessToken(token.AccessToken) != nil {
		s.Problem(w, 400, "rp_identity_invalid")
		return
	}
	if err := s.currentClient(ctx); err != nil {
		s.Problem(w, 503, "rp_client_unavailable")
		return
	}
	if _, err := s.users.User(ctx, idToken.Subject); err != nil {
		if errors.Is(err, identity.ErrMissing) {
			s.Problem(w, 401, "rp_identity_unavailable")
		} else {
			s.Problem(w, 503, "rp_unavailable")
		}
		return
	}
	handle, err := s.Sessions.CreateVerified(ctx, idToken.Subject, s.CookieToken(r.Header.Get("Cookie")), claims.SecurityVersion, s.users)
	if err != nil {
		if errors.Is(err, identity.ErrMissing) {
			s.Problem(w, 401, "rp_identity_unavailable")
		} else {
			s.Problem(w, 503, "rp_unavailable")
		}
		return
	}
	cookie := s.Sessions.Cookie(handle)
	http.SetCookie(w, &cookie)
	// OAuth credentials are discarded after verified login; only the server-side RP session persists.
	http.Redirect(w, r, s.origin+payload.ReturnTo, http.StatusSeeOther)
}
