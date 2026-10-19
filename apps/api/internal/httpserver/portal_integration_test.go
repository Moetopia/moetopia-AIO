//go:build integration

package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/persistence/ent/rplogin"
	"moetopia.local/aio/api/internal/relyingparty"
)

const portalOrigin = "https://account.moetopia.test"

type portalFixture struct {
	*accountFixture
	portal         *relyingparty.Service
	identityCookie string
	identityToken  string
	userID         string
	mutation       string
	applications   *developer.Service
}

func newPortalFixture(t *testing.T) *portalFixture {
	t.Helper()
	base := newAccountFixture(t)
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	box, err := cryptobox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := authorization.NewProvider(authorization.NewStore(base.db.Ent, box), &authorization.Key{ID: uuid.NewString(), Private: private}, portalOrigin, key, base.identity)
	if err != nil {
		t.Fatal(err)
	}
	f := &portalFixture{accountFixture: base}
	backchannel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/jwks" {
			writeJSON(w, 200, provider.JWKS())
			return
		}
		if f.mutation == "server_error" {
			writeJSON(w, 503, map[string]string{"error": "temporarily_unavailable"})
			return
		}
		response := httptest.NewRecorder()
		provider.Token(response, r)
		if response.Code != 200 || f.mutation == "" {
			for key, values := range response.Header() {
				w.Header()[key] = values
			}
			w.WriteHeader(response.Code)
			_, _ = w.Write(response.Body.Bytes())
			return
		}
		var result map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Error("invalid token response")
			return
		}
		raw, _ := result["id_token"].(string)
		claims := jwt.MapClaims{}
		if _, _, err := jwt.NewParser().ParseUnverified(raw, claims); err != nil {
			t.Error("invalid identity token fixture")
			return
		}
		switch f.mutation {
		case "issuer":
			claims["iss"] = "https://attacker.test"
		case "audience":
			claims["aud"] = []string{"other-client"}
		case "nonce":
			claims["nonce"] = "wrong-nonce"
		case "expired":
			claims["exp"] = time.Now().Add(-time.Hour).Unix()
		case "future":
			claims["iat"] = time.Now().Add(time.Hour).Unix()
		case "azp":
			claims["azp"] = "other-client"
		case "auth_time":
			claims["auth_time"] = 0
		case "at_hash":
			claims["at_hash"] = "invalid-hash"
		case "security_version":
			claims["security_version"] = 99
		}
		changed := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		changed.Header["kid"] = provider.JWKS().(map[string]any)["keys"].([]map[string]string)[0]["kid"]
		encoded, err := changed.SignedString(private)
		if err != nil {
			t.Error("could not sign fixture")
			return
		}
		if f.mutation == "signature" {
			parts := strings.Split(encoded, ".")
			parts[2] = base64.RawURLEncoding.EncodeToString(make([]byte, 256))
			encoded = strings.Join(parts, ".")
		}
		result["id_token"] = encoded
		writeJSON(w, 200, result)
	})
	if err := relyingparty.InitializeClient(t.Context(), base.db.Ent, portalOrigin, key); err != nil {
		t.Fatal(err)
	}
	if err := relyingparty.InitializeClient(t.Context(), base.db.Ent, portalOrigin, key); err != nil {
		t.Fatal("initialization must be idempotent")
	}
	portal, err := relyingparty.New(base.db.Ent, box, base.identity, portalOrigin, key, true, backchannel)
	if err != nil {
		t.Fatal(err)
	}
	f.portal = portal
	f.applications = developer.New(f.db.Ent, provider, f.identity, portalOrigin)
	f.handler = New(Dependencies{Tokens: provider, Valkey: f.cache, Portal: PortalEndpoints{Service: portal}, Developers: f.applications, Accounts: AccountEndpoints{Identity: f.identity, Sessions: f.sessions, Cache: f.cache, Authorization: provider}})
	f.userID = uuid.NewString()
	if err := f.db.Ent.AccountUser.Create().SetID(f.userID).SetEmail(f.userID + "@example.test").SetDisplayName("RP user").SetPasswordHash("unusable-test-credential").SetEmailVerifiedAt(time.Now()).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.identityToken, err = f.sessions.Create(t.Context(), f.userID, "")
	if err != nil {
		t.Fatal(err)
	}
	f.identityCookie = f.sessions.CookieName() + "=" + f.identityToken
	return f
}

func (f *portalFixture) callback(t *testing.T, decision string) (string, string) {
	t.Helper()
	f.ip = "[2001:db8:" + uuid.NewString()[:4] + "::1]:1234"
	start := f.request("GET", "/api/v1/portal/login?return_to=/developer/applications", "", "", "", nil)
	if start.Code != 303 {
		t.Fatalf("login start HTTP %d", start.Code)
	}
	browser := cookieValue(t, start)
	authorize, _ := url.Parse(start.Header().Get("Location"))
	response := f.request("GET", authorize.RequestURI(), browser+"; "+f.identityCookie, "", "", nil)
	if response.Code != 303 {
		t.Fatalf("authorize HTTP %d", response.Code)
	}
	page, _ := url.Parse(response.Header().Get("Location"))
	if page.Path != "/auth/authorize" {
		t.Fatal("authorization must use the dedicated identity UI")
	}
	cookies := browser + "; " + cookieValue(t, response) + "; " + f.identityCookie
	result := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, f.request("POST", "/api/v1/account/authorizations/"+page.Query().Get("authorization_request"), cookies, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"decision": decision, "subject_id": f.userID}), 200)
	callback, err := url.Parse(result.RedirectTo)
	if err != nil || callback.Path != relyingparty.CallbackPath {
		t.Fatal("invalid RP callback")
	}
	return callback.RequestURI(), cookies
}

func TestFirstPartyOIDCLoginAndSessionIsolation(t *testing.T) {
	f := newPortalFixture(t)
	anonymous := decodeAccount[PortalContextBody](t, f.request("GET", "/api/v1/portal/context", f.identityCookie, "", "", nil), 200)
	if anonymous.Authenticated || anonymous.User != nil {
		t.Fatal("identity session must not authenticate the RP")
	}
	for _, value := range []string{"//evil.test", "https://evil.test", "/auth/login", "/profile?next=evil"} {
		if got := f.request("GET", "/api/v1/portal/login?return_to="+url.QueryEscape(value), "", "", "", nil).Code; got != 400 {
			t.Fatal("unapproved return path accepted")
		}
	}
	callback, cookies := f.callback(t, "approve")
	u, _ := url.Parse(callback)
	bad := *u
	q := bad.Query()
	q.Set("state", strings.Repeat("x", 43))
	bad.RawQuery = q.Encode()
	if got := f.request("GET", bad.RequestURI(), cookies, "", "", nil).Code; got != 400 {
		t.Fatal("wrong state accepted")
	}
	if got := f.request("GET", callback, f.identityCookie, "", "", nil).Code; got != 400 {
		t.Fatal("callback browser binding missing")
	}
	response := f.request("GET", callback, cookies, "", "", nil)
	if response.Code != 303 || response.Header().Get("Location") != portalOrigin+"/developer/applications" {
		t.Fatalf("callback HTTP %d: %s", response.Code, response.Body.String())
	}
	portalCookie := cookieValue(t, response)
	if strings.HasPrefix(portalCookie, f.sessions.CookieName()+"=") {
		t.Fatal("RP reused identity cookie")
	}
	if got := f.request("GET", callback, cookies, "", "", nil).Code; got != 400 {
		t.Fatal("callback replay accepted")
	}
	portalState := decodeAccount[PortalContextBody](t, f.request("GET", "/api/v1/portal/context", portalCookie, "", "", nil), 200)
	if !portalState.Authenticated || portalState.User == nil || portalState.User.ID != f.userID {
		t.Fatal("verified RP session missing")
	}
	if got := f.request("GET", "/api/v1/account/sessions", portalCookie, "", "", nil).Code; got != 401 {
		t.Fatal("RP session accepted as identity session")
	}
	if got := f.request("POST", "/api/v1/portal/logout", portalCookie, f.sessions.CSRF(f.identityToken), portalOrigin, nil).Code; got != 403 {
		t.Fatal("identity CSRF accepted by RP")
	}
	if got := f.request("POST", "/api/v1/portal/logout", portalCookie, portalState.CSRFToken, "https://evil.test", nil).Code; got != 403 {
		t.Fatal("foreign Origin accepted")
	}
	list := decodeAccount[struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}](t, f.request("GET", "/api/v1/portal/sessions", portalCookie, "", "", nil), 200)
	if len(list.Items) != 1 {
		t.Fatal("RP session list mixed identity sessions")
	}
	loggedOut := decodeAccount[PortalContextBody](t, f.request("POST", "/api/v1/portal/logout", portalCookie, portalState.CSRFToken, portalOrigin, nil), 200)
	if loggedOut.Authenticated {
		t.Fatal("RP logout failed")
	}
	if got := f.request("GET", "/api/v1/portal/sessions", portalCookie, "", "", nil).Code; got != 401 {
		t.Fatal("revoked RP session remains valid")
	}
	if _, err := f.sessions.Find(t.Context(), f.identityToken); err != nil {
		t.Fatal("local RP logout unexpectedly removed identity session")
	}

	t.Run("denial is bound and single-use", func(t *testing.T) {
		callback, cookies := f.callback(t, "deny")
		for range 2 {
			if got := f.request("GET", callback, cookies, "", "", nil).Code; got != 400 {
				t.Fatal("denied callback accepted")
			}
		}
	})
	t.Run("expired transaction", func(t *testing.T) {
		callback, cookies := f.callback(t, "approve")
		if _, err := f.db.Ent.RPLogin.Update().Where(rplogin.Consumed(false)).SetExpiresAt(time.Now().Add(-time.Minute)).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if got := f.request("GET", callback, cookies, "", "", nil).Code; got != 400 {
			t.Fatal("expired transaction accepted")
		}
	})
	for _, mutation := range []string{"issuer", "audience", "nonce", "expired", "future", "azp", "auth_time", "at_hash", "signature"} {
		t.Run("reject "+mutation, func(t *testing.T) {
			f := newPortalFixture(t)
			callback, cookies := f.callback(t, "approve")
			f.mutation = mutation
			response := f.request("GET", callback, cookies, "", "", nil)
			f.mutation = ""
			if response.Code != 400 || response.Header().Get("Set-Cookie") != "" {
				t.Fatalf("invalid identity accepted: HTTP %d", response.Code)
			}
		})
	}
	t.Run("concurrent callback is single-use", func(t *testing.T) {
		callback, cookies := f.callback(t, "approve")
		codes := make(chan int, 2)
		var group sync.WaitGroup
		for range 2 {
			group.Go(func() { codes <- f.request("GET", callback, cookies, "", "", nil).Code })
		}
		group.Wait()
		close(codes)
		counts := map[int]int{}
		for code := range codes {
			counts[code]++
		}
		if counts[303] != 1 || counts[400] != 1 {
			t.Fatal("callback was not single-use")
		}
	})
	t.Run("token service failure remains unavailable", func(t *testing.T) {
		f := newPortalFixture(t)
		callback, cookies := f.callback(t, "approve")
		f.mutation = "server_error"
		response := f.request("GET", callback, cookies, "", "", nil)
		if response.Code != 503 || response.Header().Get("Set-Cookie") != "" {
			t.Fatal("token failure became successful or anonymous login")
		}
		f.mutation = ""
		if got := f.request("GET", callback, cookies, "", "", nil).Code; got != 400 {
			t.Fatal("failed consumed callback was reusable")
		}
	})
	t.Run("stale signed credential version cannot establish a session", func(t *testing.T) {
		f := newPortalFixture(t)
		callback, cookies := f.callback(t, "approve")
		f.mutation = "security_version"
		response := f.request("GET", callback, cookies, "", "", nil)
		if response.Code != 401 || response.Header().Get("Set-Cookie") != "" {
			t.Fatal("stale credential version created an RP session")
		}
	})
	t.Run("disabled subject cannot establish an RP session", func(t *testing.T) {
		f := newPortalFixture(t)
		callback, cookies := f.callback(t, "approve")
		if err := f.db.Ent.AccountUser.UpdateOneID(f.userID).SetEnabled(false).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		response := f.request("GET", callback, cookies, "", "", nil)
		if response.Code != 400 || response.Header().Get("Set-Cookie") != "" {
			t.Fatal("disabled user obtained an RP session")
		}
	})
	t.Run("disabled controlled client cannot be reenabled by initialization", func(t *testing.T) {
		t.Cleanup(func() {
			if err := f.db.Ent.OAuthClient.UpdateOneID(relyingparty.ClientID).SetEnabled(true).Exec(context.Background()); err != nil {
				t.Error("could not restore dedicated fixture client")
			}
		})
		token, err := f.portal.Sessions.Create(t.Context(), f.userID, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := f.db.Ent.OAuthClient.UpdateOneID(relyingparty.ClientID).SetEnabled(false).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
		key := base64.StdEncoding.EncodeToString(make([]byte, 32))
		if err := relyingparty.InitializeClient(t.Context(), f.db.Ent, portalOrigin, key); err == nil {
			t.Fatal("disabled client silently restored")
		}
		if got := f.request("GET", "/api/v1/portal/login", "", "", "", nil).Code; got != 503 {
			t.Fatal("disabled RP login accepted")
		}
		cookie := f.portal.Sessions.CookieName() + "=" + token
		if got := f.request("GET", "/api/v1/portal/context", cookie, "", "", nil).Code; got != 503 {
			t.Fatal("disabled client retained authenticated context")
		}
		if got := f.request("POST", "/api/v1/portal/logout", cookie, f.portal.Sessions.CSRF(token), portalOrigin, nil).Code; got != 200 {
			t.Fatal("disabled client prevented local logout")
		}
	})
}
