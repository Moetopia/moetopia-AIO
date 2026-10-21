package federation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var ErrProvider = errors.New("external identity verification failed")

type Profile struct {
	Provider    string `json:"provider"`
	Issuer      string `json:"issuer"`
	Subject     string `json:"subject"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type Provider struct {
	name     string
	oauth    oauth2.Config
	client   *http.Client
	verifier *oidc.IDTokenVerifier
}

type Credentials struct{ ClientID, ClientSecret string }

func New(name, callback string, credentials Credentials) (*Provider, error) {
	return newProvider(name, callback, credentials, http.DefaultTransport)
}

func newProvider(name, callback string, credentials Credentials, transport http.RoundTripper) (*Provider, error) {
	u, err := url.Parse(callback)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || strings.HasSuffix(u.Hostname(), ".localhost"))) || strings.TrimSpace(credentials.ClientID) == "" || strings.TrimSpace(credentials.ClientSecret) == "" {
		return nil, errors.New("invalid external provider configuration")
	}
	p := &Provider{name: name, client: &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	p.oauth = oauth2.Config{ClientID: credentials.ClientID, ClientSecret: credentials.ClientSecret, RedirectURL: callback}
	switch name {
	case "github":
		p.oauth.Endpoint = oauth2.Endpoint{AuthURL: "https://github.com/login/oauth/authorize", TokenURL: "https://github.com/login/oauth/access_token", AuthStyle: oauth2.AuthStyleInParams}
		p.oauth.Scopes = []string{"read:user", "user:email"}
	case "google":
		p.oauth.Endpoint = oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token", AuthStyle: oauth2.AuthStyleInParams}
		p.oauth.Scopes = []string{oidc.ScopeOpenID, "email", "profile"}
		ctx := oidc.ClientContext(context.Background(), p.client)
		p.verifier = oidc.NewVerifier("https://accounts.google.com", oidc.NewRemoteKeySet(ctx, "https://www.googleapis.com/oauth2/v3/certs"), &oidc.Config{ClientID: credentials.ClientID, SupportedSigningAlgs: []string{oidc.RS256}})
	default:
		return nil, errors.New("unsupported external identity provider")
	}
	return p, nil
}

func (p *Provider) AuthorizationURL(state, nonce, verifier string) string {
	return p.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce))
}

func (p *Provider) Exchange(ctx context.Context, code, nonce, verifier string) (Profile, error) {
	if code == "" || len(code) > 4096 || nonce == "" || len(verifier) < 43 {
		return Profile{}, ErrProvider
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ctx = oidc.ClientContext(ctx, p.client)
	token, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil || token.AccessToken == "" || !strings.EqualFold(token.TokenType, "bearer") {
		return Profile{}, ErrProvider
	}
	if p.name == "google" {
		return p.google(ctx, token, nonce)
	}
	return p.github(ctx, token.AccessToken)
}

func (p *Provider) get(ctx context.Context, path, accessToken string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+path, nil)
	if err != nil {
		return ErrProvider
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "Moetopia-Identity")
	response, err := p.client.Do(req)
	if err != nil {
		return ErrProvider
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ErrProvider
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil || len(data) > 256<<10 || json.Unmarshal(data, target) != nil {
		return ErrProvider
	}
	return nil
}
