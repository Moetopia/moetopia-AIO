package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestGitHubProfileAndPKCE(t *testing.T) {
	verifier := oauth2.GenerateVerifier()
	requests := 0
	p, err := newProvider("github", "https://account.moetopia.test/api/v1/account/federation/github/callback", Credentials{"client", "private-secret"}, transportFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		switch r.URL.String() {
		case "https://github.com/login/oauth/access_token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("code_verifier") != verifier || r.Form.Get("client_secret") != "private-secret" || r.Form.Get("code") != "authorization-code" {
				t.Fatal("missing server-side token exchange binding")
			}
			return response(200, `{"access_token":"upstream-secret","token_type":"bearer"}`), nil
		case "https://api.github.com/user":
			if r.Header.Get("Authorization") != "Bearer upstream-secret" {
				t.Fatal("missing authenticated profile request")
			}
			return response(200, `{"id":12345,"login":"renamable-login","name":"Display"}`), nil
		case "https://api.github.com/user/emails":
			return response(200, `[{"email":"unverified@example.test","primary":true,"verified":false},{"email":"verified@example.test","verified":true}]`), nil
		default:
			t.Fatal("unexpected upstream request")
			return nil, errors.New("unexpected request")
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(p.AuthorizationURL("bound-state", "bound-nonce", verifier))
	if u.Host != "github.com" || u.Query().Get("code_challenge_method") != "S256" || u.Query().Get("code_challenge") != oauth2.S256ChallengeFromVerifier(verifier) || u.Query().Get("state") != "bound-state" || u.Query().Get("client_secret") != "" {
		t.Fatal("unsafe authorization request")
	}
	profile, err := p.Exchange(t.Context(), "authorization-code", "bound-nonce", verifier)
	if err != nil || profile.Subject != "12345" || profile.Issuer != "https://github.com" || profile.Email != "verified@example.test" || profile.DisplayName != "Display" || requests != 3 {
		t.Fatalf("unexpected verified profile: %v", err)
	}
}

func TestProviderFailuresDoNotFollowRedirectsOrExposeTokens(t *testing.T) {
	for _, status := range []int{302, 400, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			p, err := newProvider("github", "https://account.moetopia.test/callback", Credentials{"client", "private-secret"}, transportFunc(func(*http.Request) (*http.Response, error) {
				calls++
				r := response(status, `{"error":"private-secret upstream-secret"}`)
				r.Header.Set("Location", "https://attacker.example/collect")
				return r, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.Exchange(t.Context(), "code", "nonce", oauth2.GenerateVerifier())
			if !errors.Is(err, ErrProvider) || calls != 1 || strings.Contains(err.Error(), "secret") {
				t.Fatal("provider failure leaked data or followed redirect")
			}
		})
	}
}

func TestGoogleVerifiedIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	jwks, _ := json.Marshal(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256", "n": encode(key.N.Bytes()), "e": encode(big.NewInt(int64(key.E)).Bytes())}}})
	hash := sha256.Sum256([]byte("access-token"))
	for _, scenario := range []string{"valid", "wrong-nonce", "wrong-issuer", "wrong-audience", "wrong-party", "expired", "future-issued", "wrong-token-hash", "unverified-email", "wrong-signature"} {
		t.Run(scenario, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": "https://accounts.google.com", "sub": "stable-subject", "aud": "client", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(), "nonce": "nonce", "email": "hint@example.test", "email_verified": true, "name": "Display", "at_hash": encode(hash[:16])}
			switch scenario {
			case "wrong-nonce":
				claims["nonce"] = "other"
			case "wrong-issuer":
				claims["iss"] = "https://attacker.example"
			case "wrong-audience":
				claims["aud"] = "another-client"
			case "wrong-party":
				claims["azp"] = "another-client"
			case "expired":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "future-issued":
				claims["iat"] = time.Now().Add(time.Hour).Unix()
			case "wrong-token-hash":
				claims["at_hash"] = "invalid"
			case "unverified-email":
				claims["email_verified"] = false
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test-key"
			signed, err := token.SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "wrong-signature" {
				parts := strings.Split(signed, ".")
				first := "A"
				if strings.HasPrefix(parts[2], first) {
					first = "B"
				}
				parts[2] = first + parts[2][1:]
				signed = strings.Join(parts, ".")
			}
			p, err := newProvider("google", "https://account.moetopia.test/callback", Credentials{"client", "secret"}, transportFunc(func(r *http.Request) (*http.Response, error) {
				switch r.URL.String() {
				case "https://oauth2.googleapis.com/token":
					body, _ := json.Marshal(map[string]any{"access_token": "access-token", "token_type": "Bearer", "id_token": signed})
					return response(200, string(body)), nil
				case "https://www.googleapis.com/oauth2/v3/certs":
					return response(200, string(jwks)), nil
				default:
					t.Fatal("unexpected identity endpoint")
					return nil, ErrProvider
				}
			}))
			if err != nil {
				t.Fatal(err)
			}
			profile, err := p.Exchange(t.Context(), "code", "nonce", oauth2.GenerateVerifier())
			valid := scenario == "valid" || scenario == "unverified-email"
			if valid && (err != nil || profile.Subject != "stable-subject") {
				t.Fatalf("valid Google identity rejected: %v", err)
			}
			if !valid && !errors.Is(err, ErrProvider) {
				t.Fatal("invalid identity accepted")
			}
			if scenario == "unverified-email" && profile.Email != "" {
				t.Fatal("unverified email exposed as verified hint")
			}
		})
	}
}
