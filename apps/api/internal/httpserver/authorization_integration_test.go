//go:build integration

package httpserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/cryptobox"
)

func TestAuthorizationConsentContract(t *testing.T) {
	f := newAccountFixture(t)
	encoded := base64.StdEncoding.EncodeToString(make([]byte, 32))
	box, err := cryptobox.New(encoded)
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := authorization.NewProvider(authorization.NewStore(f.db.Ent, box), &authorization.Key{ID: uuid.NewString(), Private: key}, "https://account.moetopia.test", encoded, f.identity)
	if err != nil {
		t.Fatal(err)
	}
	f.handler = New(Dependencies{Tokens: provider, Valkey: f.cache, Accounts: AccountEndpoints{Identity: f.identity, Sessions: f.sessions, Cache: f.cache, Authorization: provider}})
	metadata := decodeAccount[struct {
		Issuer     string   `json:"issuer"`
		Grants     []string `json:"grant_types_supported"`
		PKCE       []string `json:"code_challenge_methods_supported"`
		Revocation string   `json:"revocation_endpoint"`
	}](t, f.request("GET", "/.well-known/openid-configuration", "", "", "", nil), 200)
	if metadata.Issuer != "https://account.moetopia.test" || !slices.Contains(metadata.Grants, "authorization_code") || !slices.Contains(metadata.Grants, "refresh_token") || !slices.Equal(metadata.PKCE, []string{"S256"}) || metadata.Revocation != "https://account.moetopia.test/oauth/revoke" {
		t.Fatal("metadata advertised unsupported capabilities")
	}
	id := uuid.NewString()
	_, err = f.db.Ent.AccountUser.Create().SetID(id).SetEmail(id + "@example.test").SetDisplayName("Consent user").SetPasswordHash("unusable-test-credential").SetEmailVerifiedAt(time.Now()).Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	token, err := f.sessions.Create(t.Context(), id, "")
	if err != nil {
		t.Fatal(err)
	}
	client := uuid.NewString()
	_, err = f.db.Ent.OAuthClient.Create().SetID(client).SetName("Real test app").SetEnabled(true).SetPublic(true).SetRedirectUris([]string{"https://client.moetopia.test/callback"}).SetGrantTypes([]string{"authorization_code"}).SetResponseTypes([]string{"code"}).SetScopes([]string{"openid"}).SetAudiences([]string{"https://api.moetopia.test"}).Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	challenge := sha256.Sum256([]byte(strings.Repeat("v", 43)))
	query := url.Values{"client_id": {client}, "response_type": {"code"}, "redirect_uri": {"https://client.moetopia.test/callback"}, "scope": {"openid"}, "state": {"test-state-value"}, "nonce": {"test-nonce-value"}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}}
	start := f.request("GET", "/oauth/authorize?"+query.Encode(), "", "", "", nil)
	if start.Code != 303 {
		t.Fatalf("authorize HTTP %d", start.Code)
	}
	location, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/account/authorizations/" + location.Query().Get("authorization_request")
	browser := cookieValue(t, start)
	preview := decodeAccount[struct {
		ClientName    string `json:"client_name"`
		LoginRequired bool   `json:"login_required"`
	}](t, f.request("GET", path, browser, "", "", nil), 200)
	if preview.ClientName != "Real test app" || !preview.LoginRequired {
		t.Fatal("incorrect anonymous preview")
	}
	cookie := browser + "; " + f.sessions.CookieName() + "=" + token
	csrf := f.sessions.CSRF(token)
	body := map[string]string{"decision": "approve", "subject_id": id}
	for _, bad := range []struct {
		cookie, csrf, origin string
		status               int
	}{{cookie, "wrong", "https://account.moetopia.test", 403}, {cookie, csrf, "https://attacker.test", 403}, {browser, csrf, "https://account.moetopia.test", 403}, {f.sessions.CookieName() + "=" + token, csrf, "https://account.moetopia.test", 404}} {
		if got := f.request("POST", path, bad.cookie, bad.csrf, bad.origin, body).Code; got != bad.status {
			t.Fatalf("consent protection HTTP %d expected %d", got, bad.status)
		}
	}
	wrongUser := map[string]string{"decision": "approve", "subject_id": "other-user"}
	if got := f.request("POST", path, cookie, csrf, "https://account.moetopia.test", wrongUser).Code; got != 409 {
		t.Fatalf("account change HTTP %d", got)
	}
	result := decodeAccount[struct {
		RedirectTo string `json:"redirect_to"`
	}](t, f.request("POST", path, cookie, csrf, "https://account.moetopia.test", body), 200)
	callback, err := url.Parse(result.RedirectTo)
	if err != nil || callback.Host != "client.moetopia.test" || callback.Query().Get("code") == "" || callback.Query().Get("state") != "test-state-value" {
		t.Fatal("incorrect consent callback contract")
	}
	if got := f.request("POST", path, cookie, csrf, "https://account.moetopia.test", body).Code; got != 404 {
		t.Fatalf("repeated consent HTTP %d", got)
	}
}
