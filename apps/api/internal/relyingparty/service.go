package relyingparty

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/rplogin"
	"moetopia.local/aio/api/internal/sessions"
)

var ErrLogin = errors.New("RP login unavailable or invalid")

type SubjectSource interface {
	User(context.Context, string) (identity.User, error)
	VerifyVersion(context.Context, string, int) error
}
type Service struct {
	db       *ent.Client
	box      *cryptobox.Box
	users    SubjectSource
	issuer   string
	secure   bool
	client   *http.Client
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	Sessions *sessions.Service
	clientID string
	origin   string
	admin    bool
}

func New(db *ent.Client, box *cryptobox.Box, users SubjectSource, issuer, key string, secure bool, protocol http.Handler) (*Service, error) {
	return newService(db, box, users, issuer, issuer, key, secure, protocol, false)
}
func NewAdmin(db *ent.Client, box *cryptobox.Box, users SubjectSource, issuer, origin, key string, secure bool, protocol http.Handler) (*Service, error) {
	return newService(db, box, users, issuer, origin, key, secure, protocol, true)
}
func newService(db *ent.Client, box *cryptobox.Box, users SubjectSource, issuer, origin, key string, secure bool, protocol http.Handler, admin bool) (*Service, error) {
	id, callback := ClientID, CallbackPath
	if admin {
		id, callback = AdminClientID, AdminCallbackPath
	}
	secret, err := profileSecret(key, admin)
	if err != nil {
		return nil, err
	}
	constructor := sessions.NewPortal
	if admin {
		constructor = sessions.NewAdmin
	}
	web, err := constructor(db, key, origin, secure)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: protocolTransport{issuer, protocol}, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	ctx := oidc.ClientContext(context.Background(), client)
	verifier := oidc.NewVerifier(issuer, oidc.NewRemoteKeySet(ctx, issuer+"/oauth/jwks"), &oidc.Config{ClientID: id, SupportedSigningAlgs: []string{oidc.RS256}})
	return &Service{db: db, box: box, users: users, issuer: issuer, secure: secure, client: client, verifier: verifier, Sessions: web, clientID: id, origin: origin, admin: admin,
		oauth: oauth2.Config{ClientID: id, ClientSecret: secret, RedirectURL: origin + callback, Scopes: []string{"openid", "profile", "email"}, Endpoint: oauth2.Endpoint{AuthURL: issuer + "/oauth/authorize", TokenURL: issuer + "/oauth/token", AuthStyle: oauth2.AuthStyleInHeader}}}, nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func ReturnPath(value string) (string, error) {
	if value == "" {
		return "/profile", nil
	}
	if !slices.Contains([]string{"/profile", "/sessions", "/security", "/developer", "/developer/applications", "/developer/docs"}, value) {
		return "", ErrLogin
	}
	return value, nil
}
func (s *Service) browserCookie() string {
	if s.admin {
		if s.secure {
			return "__Host-moetopia_admin_browser"
		}
		return "moetopia_admin_browser"
	}
	if s.secure {
		return "__Host-moetopia_rp_browser"
	}
	return "moetopia_rp_browser"
}
func (s *Service) CookieToken(cookies string) string {
	r := &http.Request{Header: http.Header{"Cookie": []string{cookies}}}
	c, err := r.Cookie(s.Sessions.CookieName())
	if err != nil {
		return ""
	}
	return c.Value
}
func (s *Service) User(ctx context.Context, token string) (identity.User, sessions.Session, error) {
	web, err := s.Sessions.Find(ctx, token)
	if err != nil {
		return identity.User{}, web, err
	}
	if err := s.currentClient(ctx); err != nil {
		return identity.User{}, web, err
	}
	user, err := s.users.User(ctx, web.UserID)
	return user, web, err
}
func (s *Service) Cleanup(ctx context.Context) error {
	_, err := s.db.RPLogin.Delete().Where(rplogin.ExpiresAtLT(time.Now())).Exec(ctx)
	return err
}
func (s *Service) currentClient(ctx context.Context) error {
	row, err := s.db.OAuthClient.Get(ctx, s.clientID)
	if err != nil {
		return err
	}
	if !row.Enabled || row.Public || row.OwnerID != "" || row.TestOnly || !slices.Equal(row.RedirectUris, []string{s.oauth.RedirectURL}) || !slices.Equal(row.GrantTypes, []string{"authorization_code"}) || !slices.Equal(row.ResponseTypes, []string{"code"}) || !slices.Equal(row.Scopes, s.oauth.Scopes) || !slices.Equal(row.Audiences, []string{s.issuer + "/api"}) {
		return ErrLogin
	}
	return nil
}

func (s *Service) returnPath(value string) (string, error) {
	if !s.admin {
		return ReturnPath(value)
	}
	if value == "" {
		return "/", nil
	}
	if !slices.Contains([]string{"/", "/applications", "/users", "/invitations", "/audit", "/settings"}, value) {
		return "", ErrLogin
	}
	return value, nil
}
func (s *Service) Problem(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"title":"Sign-in could not be completed","status":` + strconv.Itoa(status) + `,"code":"` + code + `"}`))
}
