package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/valkey-io/valkey-go"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/sessions"
)

type AccountContextBody struct {
	CSRFToken        string         `json:"csrf_token"`
	User             *identity.User `json:"user,omitempty"`
	RegistrationMode string         `json:"registration_mode" enum:"invite,open"`
}
type AccountContextOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      AccountContextBody
}
type AccountCookie struct {
	Cookie string `header:"Cookie"`
}
type AccountWrite struct {
	Cookie string `header:"Cookie"`
	Origin string `header:"Origin"`
	CSRF   string `header:"X-Moetopia-CSRF"`
}
type LoginInput struct {
	AccountWrite
	Body struct {
		Email    string `json:"email" maxLength:"254"`
		Password string `json:"password" maxLength:"256"`
	}
}
type VerificationInput struct {
	AccountWrite
	Body struct {
		Email  string `json:"email" maxLength:"254"`
		Locale string `json:"locale" enum:"zh-CN,en"`
	}
}
type VerificationOutput struct {
	Body struct {
		ChallengeID string `json:"challenge_id"`
	}
}
type RegisterInput struct {
	AccountWrite
	Body struct {
		ChallengeID string `json:"challenge_id" maxLength:"36"`
		Code        string `json:"code" minLength:"8" maxLength:"8"`
		Invitation  string `json:"invitation" maxLength:"128"`
		DisplayName string `json:"display_name" minLength:"1" maxLength:"80"`
		Password    string `json:"password" minLength:"15" maxLength:"256"`
	}
}
type SessionsOutput struct {
	Body struct {
		Items     []sessions.Session `json:"items"`
		CurrentID string             `json:"current_id"`
	}
}
type RevokeInput struct {
	AccountWrite
	ID string `path:"id" maxLength:"64"`
}
type AccountEndpoints struct {
	Identity      *identity.Service
	Sessions      *sessions.Service
	Cache         valkey.Client
	Authorization *authorization.Provider
	RateNamespace string
}
type AccountProblem struct {
	huma.ErrorModel
	Code string `json:"code"`
}

func (a AccountEndpoints) token(cookies string) string {
	req := &http.Request{Header: http.Header{"Cookie": []string{cookies}}}
	cookie, err := req.Cookie(a.Sessions.CookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}
func accountError(err error) error {
	if errors.Is(err, identity.ErrInvalid) {
		return &AccountProblem{ErrorModel: huma.ErrorModel{Status: 422, Title: "Registration failed", Detail: "Invalid or expired registration details", Type: "about:blank"}, Code: "registration_invalid"}
	}
	if errors.Is(err, identity.ErrCredentials) {
		return &AccountProblem{ErrorModel: huma.ErrorModel{Status: 401, Title: "Sign-in failed", Detail: "Invalid email or password", Type: "about:blank"}, Code: "credentials_invalid"}
	}
	if errors.Is(err, identity.ErrMissing) || errors.Is(err, sessions.ErrMissing) {
		return huma.Error401Unauthorized("Authentication required")
	}
	return huma.Error503ServiceUnavailable("Account service temporarily unavailable")
}
func (a AccountEndpoints) protect(input AccountWrite) error {
	if a.Identity == nil || a.Sessions == nil {
		return huma.Error503ServiceUnavailable("Account service unavailable")
	}
	if !a.Sessions.Protect(a.token(input.Cookie), input.Origin, input.CSRF) {
		return huma.Error403Forbidden("Request verification failed")
	}
	return nil
}
func (a AccountEndpoints) user(ctx context.Context, cookies string) (identity.User, sessions.Session, error) {
	if a.Identity == nil || a.Sessions == nil {
		return identity.User{}, sessions.Session{}, huma.Error503ServiceUnavailable("Account service unavailable")
	}
	session, err := a.Sessions.Find(ctx, a.token(cookies))
	if err != nil {
		return identity.User{}, session, accountError(err)
	}
	user, err := a.Identity.User(ctx, session.UserID)
	if err != nil {
		return identity.User{}, session, accountError(err)
	}
	return user, session, nil
}
func (a AccountEndpoints) context(ctx context.Context, cookies string) (*AccountContextOutput, error) {
	if a.Identity == nil || a.Sessions == nil {
		return nil, huma.Error503ServiceUnavailable("Account service unavailable")
	}
	mode, err := a.Identity.Mode(ctx)
	if err != nil {
		return nil, accountError(err)
	}
	token := a.token(cookies)
	result := &AccountContextOutput{Body: AccountContextBody{RegistrationMode: mode}}
	session, err := a.Sessions.Find(ctx, token)
	if err == nil {
		user, err := a.Identity.User(ctx, session.UserID)
		if err == nil {
			result.Body.User = &user
		} else if !errors.Is(err, identity.ErrMissing) {
			return nil, accountError(err)
		}
	} else if !errors.Is(err, sessions.ErrMissing) {
		return nil, accountError(err)
	}
	if !sessions.ValidToken(token) || result.Body.User == nil && err == nil {
		token, err = sessions.RandomToken()
		if err != nil {
			return nil, accountError(err)
		}
	}
	result.SetCookie = a.Sessions.Cookie(token)
	result.Body.CSRFToken = a.Sessions.CSRF(token)
	return result, nil
}
func (a AccountEndpoints) rate(ctx context.Context, email string, clientIP string, sending bool) error {
	if a.Cache == nil {
		return huma.Error503ServiceUnavailable("Rate limiter unavailable")
	}
	script := valkey.NewLuaScript("local n = redis.call('INCR', KEYS[1]); if n == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end; return n")
	for _, limit := range []struct {
		key     string
		max     int64
		seconds string
	}{{"ip:" + clientIP, 30, "60"}, {"email:" + strings.ToLower(strings.TrimSpace(email)), 10, "600"}} {
		if sending && strings.HasPrefix(limit.key, "email:") {
			limit.max = 3
		}
		sum := sha256.Sum256([]byte(limit.key))
		count, err := script.Exec(ctx, a.Cache, []string{a.RateNamespace + "limit:account:" + hex.EncodeToString(sum[:])}, []string{limit.seconds}).AsInt64()
		if err != nil {
			return huma.Error503ServiceUnavailable("Rate limiter unavailable")
		}
		if count > limit.max {
			return huma.Error429TooManyRequests("Please retry later")
		}
	}
	return nil
}

type accountIPKey struct{}

func clientIP(ctx context.Context) string {
	value, _ := ctx.Value(accountIPKey{}).(string)
	return value
}

func (a AccountEndpoints) Register(api huma.API) {
	huma.Get(api, "/api/v1/account/context", func(ctx context.Context, input *AccountCookie) (*AccountContextOutput, error) {
		return a.context(ctx, input.Cookie)
	})
	huma.Post(api, "/api/v1/account/login", func(ctx context.Context, input *LoginInput) (*AccountContextOutput, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.rate(ctx, input.Body.Email, clientIP(ctx), false); err != nil {
			return nil, err
		}
		user, err := a.Identity.Login(ctx, input.Body.Email, input.Body.Password)
		if err != nil {
			return nil, accountError(err)
		}
		token, err := a.Sessions.CreateVerified(ctx, user.ID, a.token(input.Cookie), user.SecurityVersion, a.Identity)
		if err != nil {
			return nil, accountError(err)
		}
		return a.context(ctx, a.Sessions.CookieName()+"="+token)
	})
	huma.Post(api, "/api/v1/account/verification", func(ctx context.Context, input *VerificationInput) (*VerificationOutput, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.rate(ctx, input.Body.Email, clientIP(ctx), true); err != nil {
			return nil, err
		}
		id, err := a.Identity.Begin(ctx, input.Body.Email, input.Body.Locale)
		if err != nil {
			return nil, accountError(err)
		}
		result := &VerificationOutput{}
		result.Body.ChallengeID = id
		return result, nil
	})
	huma.Post(api, "/api/v1/account/register", func(ctx context.Context, input *RegisterInput) (*struct{}, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.rate(ctx, input.Body.ChallengeID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		if err := a.Identity.Register(ctx, input.Body.ChallengeID, input.Body.Code, input.Body.Invitation, input.Body.DisplayName, input.Body.Password); err != nil {
			return nil, accountError(err)
		}
		return nil, nil
	})
	huma.Get(api, "/api/v1/account/sessions", func(ctx context.Context, input *AccountCookie) (*SessionsOutput, error) {
		user, session, err := a.user(ctx, input.Cookie)
		if err != nil {
			return nil, err
		}
		items, err := a.Sessions.List(ctx, user.ID)
		if err != nil {
			return nil, accountError(err)
		}
		result := &SessionsOutput{}
		result.Body.Items, result.Body.CurrentID = items, session.ID
		return result, nil
	})
	huma.Delete(api, "/api/v1/account/sessions/{id}", func(ctx context.Context, input *RevokeInput) (*struct{}, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		user, _, err := a.user(ctx, input.Cookie)
		if err != nil {
			return nil, err
		}
		if err := a.Sessions.Revoke(ctx, user.ID, input.ID); err != nil {
			return nil, accountError(err)
		}
		return nil, nil
	})
	huma.Post(api, "/api/v1/account/logout", func(ctx context.Context, input *AccountWrite) (*AccountContextOutput, error) {
		if err := a.protect(*input); err != nil {
			return nil, err
		}
		user, session, err := a.user(ctx, input.Cookie)
		if err != nil {
			return nil, err
		}
		if err := a.Sessions.Revoke(ctx, user.ID, session.ID); err != nil {
			return nil, accountError(err)
		}
		return a.context(ctx, "")
	})
}
