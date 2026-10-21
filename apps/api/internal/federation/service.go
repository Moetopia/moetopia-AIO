package federation

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"golang.org/x/oauth2"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/externallogin"
	"moetopia.local/aio/api/internal/sessions"
)

var ErrFlow = errors.New("external login is invalid or expired")
var ErrUnavailable = errors.New("external provider is not configured")

type upstream interface {
	AuthorizationURL(state, nonce, verifier string) string
	Exchange(context.Context, string, string, string) (Profile, error)
}

type Service struct {
	db              *ent.Client
	box             *cryptobox.Box
	users           *identity.Service
	providers       map[string]upstream
	bindingSessions *sessions.Service
}

func NewService(db *ent.Client, box *cryptobox.Box, users *identity.Service, providers map[string]*Provider, bindingSessions *sessions.Service) *Service {
	configured := make(map[string]upstream, len(providers))
	for name, provider := range providers {
		if provider != nil {
			configured[name] = provider
		}
	}
	return &Service{db, box, users, configured, bindingSessions}
}

func (s *Service) Providers() []string {
	names := make([]string, 0, 2)
	for _, name := range []string{"github", "google"} {
		if s.providers[name] != nil {
			names = append(names, name)
		}
	}
	return names
}

type Continuation struct {
	AuthorizationRequest string `json:"authorization_request"`
	Developer            bool   `json:"developer"`
}
type Binding struct {
	User      identity.User
	SessionID string
}
type payload struct {
	Verifier        string       `json:"verifier,omitempty"`
	Nonce           string       `json:"nonce,omitempty"`
	SessionID       string       `json:"session_id,omitempty"`
	SecurityVersion int          `json:"security_version,omitempty"`
	Continuation    Continuation `json:"continuation"`
	Profile         *Profile     `json:"profile,omitempty"`
}
type Started struct{ RedirectURL, Browser string }
type Result struct {
	User         *identity.User
	Completion   string
	Bound        bool
	Continuation Continuation
}
type RegistrationDetails struct {
	Provider    string    `json:"provider"`
	DisplayName string    `json:"display_name"`
	EmailHint   string    `json:"email_hint"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (s *Service) Begin(ctx context.Context, provider, browser string, continuation Continuation, binding *Binding) (Started, error) {
	upstream := s.providers[provider]
	if upstream == nil {
		return Started{}, ErrUnavailable
	}
	if continuation.AuthorizationRequest != "" && !sessions.ValidToken(continuation.AuthorizationRequest) {
		return Started{}, ErrFlow
	}
	if !sessions.ValidToken(browser) {
		var err error
		browser, err = sessions.RandomToken()
		if err != nil {
			return Started{}, err
		}
	}
	nonce, err := sessions.RandomToken()
	if err != nil {
		return Started{}, err
	}
	data := payload{Verifier: oauth2.GenerateVerifier(), Nonce: nonce, Continuation: continuation}
	purpose, userID := "login", ""
	if binding != nil {
		if binding.User.ID == "" || !binding.User.HasPassword || binding.SessionID == "" {
			return Started{}, identity.ErrCredentials
		}
		purpose, userID = "bind", binding.User.ID
		data.SessionID, data.SecurityVersion = binding.SessionID, binding.User.SecurityVersion
	}
	state, err := s.save(ctx, provider, purpose, userID, browser, data)
	if err != nil {
		return Started{}, err
	}
	return Started{upstream.AuthorizationURL(state, nonce, data.Verifier), browser}, nil
}

func (s *Service) save(ctx context.Context, provider, purpose, userID, browser string, data payload) (string, error) {
	handle, err := sessions.RandomToken()
	if err != nil {
		return "", err
	}
	id := digest("external-state:" + handle)
	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	encrypted, err := s.box.Seal(plaintext, "external-login:"+id)
	if err != nil {
		return "", err
	}
	err = persistence.Client(ctx, s.db).ExternalLogin.Create().SetID(id).SetBrowserHash(digest("external-browser:" + browser)).SetProvider(externallogin.Provider(provider)).SetPurpose(externallogin.Purpose(purpose)).SetUserID(userID).SetPayload(encrypted).SetExpiresAt(time.Now().Add(5 * time.Minute)).Exec(ctx)
	return handle, err
}

func (s *Service) read(ctx context.Context, handle, browser, provider, purpose string, consume bool) (*ent.ExternalLogin, payload, error) {
	var data payload
	if !sessions.ValidToken(handle) || !sessions.ValidToken(browser) {
		return nil, data, ErrFlow
	}
	query := persistence.Client(ctx, s.db).ExternalLogin.Query().Where(externallogin.ID(digest("external-state:" + handle)))
	if consume {
		query.ForUpdate()
	}
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, data, ErrFlow
	}
	if err != nil {
		return nil, data, err
	}
	if row.Consumed || !time.Now().Before(row.ExpiresAt) || provider != "" && string(row.Provider) != provider || purpose != "" && string(row.Purpose) != purpose || subtle.ConstantTimeCompare([]byte(row.BrowserHash), []byte(digest("external-browser:"+browser))) != 1 {
		return nil, data, ErrFlow
	}
	plaintext, err := s.box.Open(row.Payload, "external-login:"+row.ID)
	if err != nil {
		return nil, data, err
	}
	if json.Unmarshal(plaintext, &data) != nil {
		return nil, data, ErrFlow
	}
	if consume {
		if err := persistence.Client(ctx, s.db).ExternalLogin.UpdateOneID(row.ID).SetConsumed(true).SetPayload([]byte{}).Exec(ctx); err != nil {
			return nil, data, err
		}
	}
	return row, data, nil
}

func (s *Service) Callback(ctx context.Context, provider, state, browser, code string, denied bool, binding *Binding) (Result, error) {
	upstream := s.providers[provider]
	if upstream == nil {
		return Result{}, ErrUnavailable
	}
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return Result{}, err
	}
	defer unit.Rollback()
	row, data, err := s.read(persistence.WithClient(ctx, unit.Client()), state, browser, provider, "", true)
	if err != nil {
		return Result{}, err
	}
	if row.Purpose != externallogin.PurposeLogin && row.Purpose != externallogin.PurposeBind {
		return Result{}, ErrFlow
	}
	if err := unit.Commit(); err != nil {
		return Result{}, err
	}
	if denied {
		return Result{}, ErrProvider
	}
	profile, err := upstream.Exchange(ctx, code, data.Nonce, data.Verifier)
	if err != nil {
		return Result{}, err
	}
	if profile.Provider != provider {
		return Result{}, ErrProvider
	}
	external := identity.External{Provider: profile.Provider, Issuer: profile.Issuer, Subject: profile.Subject}
	if row.Purpose == externallogin.PurposeBind {
		if binding == nil || binding.User.ID != row.UserID || binding.SessionID != data.SessionID || binding.User.SecurityVersion != data.SecurityVersion {
			return Result{}, ErrFlow
		}
		if s.bindingSessions == nil {
			return Result{}, ErrFlow
		}
		unit, err := persistence.Begin(ctx, s.db)
		if err != nil {
			return Result{}, err
		}
		defer unit.Rollback()
		ctx = persistence.WithClient(ctx, unit.Client())
		if err := s.users.VerifyVersion(ctx, binding.User.ID, binding.User.SecurityVersion); err != nil {
			return Result{}, err
		}
		if err := s.bindingSessions.AssertLive(ctx, binding.User.ID, binding.SessionID); err != nil {
			return Result{}, err
		}
		if err := s.users.BindExternal(ctx, binding.User, external); err != nil {
			return Result{}, err
		}
		if err := unit.Commit(); err != nil {
			return Result{}, err
		}
		return Result{Bound: true}, nil
	}
	user, err := s.users.ExternalUser(ctx, external)
	if err == nil {
		return Result{User: &user, Continuation: data.Continuation}, nil
	}
	if !errors.Is(err, identity.ErrMissing) {
		return Result{}, err
	}
	completion, err := s.save(ctx, provider, "register", "", browser, payload{Profile: &profile, Continuation: data.Continuation})
	return Result{Completion: completion}, err
}
