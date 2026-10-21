package authorization

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/logoutinteraction"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/sessions"
)

var ErrLogout = errors.New("logout request unavailable or invalid")
var ErrLogoutAccount = errors.New("logout account changed")

type logoutPayload struct {
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
	HintSubject string `json:"hint_subject"`
}
type LogoutDetails struct {
	ClientID              string    `json:"client_id"`
	ClientName            string    `json:"client_name"`
	PostLogoutRedirectURI string    `json:"post_logout_redirect_uri"`
	ExpiresAt             time.Time `json:"expires_at"`
	RequiresConfirmation  bool      `json:"requires_confirmation"`
}

func (p *Provider) logoutParameters(ctx context.Context, values url.Values) (string, logoutPayload, error) {
	data := logoutPayload{RedirectURI: values.Get("post_logout_redirect_uri"), State: values.Get("state")}
	for name, items := range values {
		if len(items) != 1 || !slices.Contains([]string{"client_id", "post_logout_redirect_uri", "state", "id_token_hint", "ui_locales"}, name) {
			return "", data, ErrLogout
		}
	}
	if len(data.State) > 512 || len(values.Get("client_id")) > 128 || len(data.RedirectURI) > 2048 || len(values.Get("ui_locales")) > 128 {
		return "", data, ErrLogout
	}
	clientID := values.Get("client_id")
	if raw := values.Get("id_token_hint"); raw != "" {
		if len(raw) > 16384 {
			return "", data, ErrLogout
		}
		claims := &jwt.RegisteredClaims{}
		_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
			id, ok := token.Header["kid"].(string)
			if !ok || id == "" || len(id) > 128 || token.Header["typ"] != "JWT" {
				return nil, ErrLogout
			}
			return p.keys.Public(ctx, id)
		}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(p.issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(time.Minute))
		if err != nil || claims.IssuedAt == nil || len(claims.Audience) != 1 || claims.Subject == "" || clientID != "" && clientID != claims.Audience[0] {
			return "", data, ErrLogout
		}
		clientID, data.HintSubject = claims.Audience[0], claims.Subject
	}
	if clientID == "" || len(clientID) > 128 {
		return "", data, ErrLogout
	}
	if _, err := p.logoutClient(ctx, clientID, data.RedirectURI, false); err != nil {
		return "", data, err
	}
	return clientID, data, nil
}

func (p *Provider) logoutClient(ctx context.Context, id, redirect string, lock bool) (*ent.OAuthClient, error) {
	query := persistence.Client(ctx, p.store.client).OAuthClient.Query().Where(oauthclient.ID(id))
	if lock {
		query.ForShare()
	}
	client, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrLogout
	}
	if err != nil {
		return nil, err
	}
	if !client.Enabled || !slices.Contains(client.Scopes, "openid") || redirect != "" && !slices.Contains(client.PostLogoutRedirectUris, redirect) {
		return nil, ErrLogout
	}
	if redirect != "" {
		u, err := url.Parse(redirect)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || strings.ContainsAny(redirect, "\r\n\\") || u.Scheme != "https" && !(strings.HasPrefix(p.issuer, "http://") && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1" || strings.HasSuffix(u.Hostname(), ".localhost"))) {
			return nil, ErrLogout
		}
	}
	return client, nil
}

func (p *Provider) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	values, err := url.ParseQuery(r.URL.RawQuery)
	if r.Method == http.MethodPost {
		if r.URL.RawQuery != "" || r.ParseForm() != nil || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			err = ErrLogout
		} else {
			values = r.PostForm
		}
	}
	if len(r.URL.RawQuery) > 24576 {
		err = ErrLogout
	}
	if err != nil {
		logoutProblem(w, 400)
		return
	}
	clientID, data, err := p.logoutParameters(r.Context(), values)
	if err != nil {
		status := 503
		if errors.Is(err, ErrLogout) {
			status = 400
		}
		logoutProblem(w, status)
		return
	}
	browser := p.BrowserToken(r.Header.Get("Cookie"))
	if browser == "" {
		browser, err = sessions.RandomToken()
		if err != nil {
			logoutProblem(w, 503)
			return
		}
	}
	handle, err := sessions.RandomToken()
	if err != nil {
		logoutProblem(w, 503)
		return
	}
	id := signatureID("logout-interaction", handle)
	plaintext, _ := json.Marshal(data)
	encrypted, err := p.store.box.Seal(plaintext, "logout-interaction:"+id+":"+clientID)
	if err != nil {
		logoutProblem(w, 503)
		return
	}
	err = p.store.client.LogoutInteraction.Create().SetID(id).SetClientID(clientID).SetBrowserHash(signatureID("logout-browser", browser)).SetPayload(encrypted).SetExpiresAt(time.Now().Add(5 * time.Minute)).Exec(r.Context())
	if err != nil {
		logoutProblem(w, 503)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: p.browserCookie(), Value: browser, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(p.issuer, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 3600})
	http.Redirect(w, r, p.issuer+"/auth/confirm-logout?logout_request="+handle, http.StatusSeeOther)
}

func logoutProblem(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	code := "invalid_request"
	if status == 503 {
		code = "temporarily_unavailable"
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

func (p *Provider) logoutInteraction(ctx context.Context, handle, browser string, lock bool) (*ent.LogoutInteraction, logoutPayload, error) {
	var data logoutPayload
	if !sessions.ValidToken(handle) || !sessions.ValidToken(browser) {
		return nil, data, ErrLogout
	}
	query := persistence.Client(ctx, p.store.client).LogoutInteraction.Query().Where(logoutinteraction.ID(signatureID("logout-interaction", handle)))
	if lock {
		query.ForUpdate()
	}
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, data, ErrLogout
	}
	if err != nil {
		return nil, data, err
	}
	if row.Consumed || !time.Now().Before(row.ExpiresAt) || subtle.ConstantTimeCompare([]byte(row.BrowserHash), []byte(signatureID("logout-browser", browser))) != 1 {
		return nil, data, ErrLogout
	}
	plaintext, err := p.store.box.Open(row.Payload, "logout-interaction:"+row.ID+":"+row.ClientID)
	if err != nil {
		return nil, data, err
	}
	if json.Unmarshal(plaintext, &data) != nil {
		return nil, data, ErrLogout
	}
	return row, data, nil
}
