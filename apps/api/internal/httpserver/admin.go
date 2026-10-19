package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
)

type AdminEndpoints struct {
	RP         *relyingparty.Service
	Service    *administration.Service
	Accounts   AccountEndpoints
	Developers *developer.Service
}
type AdminContextBody struct {
	Authenticated bool                  `json:"authenticated"`
	CSRFToken     string                `json:"csrf_token"`
	LoginURL      string                `json:"login_url"`
	User          *identity.User        `json:"user,omitempty"`
	ExpiresAt     *time.Time            `json:"expires_at,omitempty"`
	MFA           *administration.State `json:"mfa,omitempty"`
	RecoveryCodes []string              `json:"recovery_codes,omitempty"`
}
type AdminContextOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      AdminContextBody
}

func adminError(err error) error {
	switch {
	case errors.Is(err, administration.ErrRole):
		return authorizationProblem(403, "Administrator access required", "admin_role_required")
	case errors.Is(err, administration.ErrFactor):
		return authorizationProblem(403, "Administrator MFA required", "admin_mfa_required")
	case errors.Is(err, administration.ErrProof):
		return authorizationProblem(422, "Invalid administrator verification", "admin_verification_invalid")
	case errors.Is(err, administration.ErrState):
		return authorizationProblem(409, "Administrator security state changed", "admin_security_state_changed")
	case errors.Is(err, administration.ErrProtected):
		return authorizationProblem(403, "Administrator account requires separate recovery controls", "admin_target_protected")
	case errors.Is(err, audit.ErrCursor):
		return authorizationProblem(422, "Invalid audit cursor", "cursor_invalid")
	default:
		return applicationError(err)
	}
}
func (a AdminEndpoints) context(ctx context.Context, cookies string) (*AdminContextOutput, error) {
	if a.RP == nil || a.Service == nil {
		return nil, huma.Error503ServiceUnavailable("Administrator service unavailable")
	}
	token := a.RP.CookieToken(cookies)
	result := &AdminContextOutput{Body: AdminContextBody{LoginURL: "/api/v1/admin/login"}}
	user, web, err := a.RP.User(ctx, token)
	if err == nil {
		state, err := a.Service.State(ctx, user.ID, web.ID)
		if err != nil {
			return nil, adminError(err)
		}
		result.Body.Authenticated, result.Body.User, result.Body.ExpiresAt, result.Body.MFA = true, &user, &web.ExpiresAt, &state
	} else if !errors.Is(err, sessions.ErrMissing) && !errors.Is(err, identity.ErrMissing) {
		return nil, adminError(err)
	} else if !sessions.ValidToken(token) || errors.Is(err, identity.ErrMissing) {
		token, err = sessions.RandomToken()
		if err != nil {
			return nil, adminError(err)
		}
	}
	result.SetCookie = a.RP.Sessions.Cookie(token)
	result.Body.CSRFToken = a.RP.Sessions.CSRF(token)
	return result, nil
}
func (a AdminEndpoints) actor(ctx context.Context, input AccountWrite, write bool) (identity.User, sessions.Session, error) {
	if a.RP == nil || a.Service == nil {
		return identity.User{}, sessions.Session{}, huma.Error503ServiceUnavailable("Administrator service unavailable")
	}
	if write && !a.RP.Sessions.Protect(a.RP.CookieToken(input.Cookie), input.Origin, input.CSRF) {
		return identity.User{}, sessions.Session{}, huma.Error403Forbidden("Request verification failed")
	}
	user, web, err := a.RP.User(ctx, a.RP.CookieToken(input.Cookie))
	if err != nil {
		return user, web, adminError(err)
	}
	if write {
		if err := a.Accounts.rate(ctx, "admin:"+user.ID, clientIP(ctx), false); err != nil {
			return user, web, err
		}
	}
	return user, web, nil
}
func (a AdminEndpoints) rotated(ctx context.Context, handle string, codes []string) (*AdminContextOutput, error) {
	result, err := a.context(ctx, a.RP.Sessions.CookieName()+"="+handle)
	if err != nil {
		return nil, err
	}
	result.Body.RecoveryCodes = codes
	return result, nil
}
func (a AdminEndpoints) Register(api huma.API) {
	huma.Get(api, "/api/v1/admin/context", func(ctx context.Context, input *AccountCookie) (*AdminContextOutput, error) {
		return a.context(ctx, input.Cookie)
	})
	huma.Post(api, "/api/v1/admin/logout", func(ctx context.Context, input *AccountWrite) (*AdminContextOutput, error) {
		user, web, err := a.actor(ctx, *input, true)
		if err != nil {
			return nil, err
		}
		if err := a.RP.Sessions.Revoke(ctx, user.ID, web.ID); err != nil {
			return nil, adminError(err)
		}
		return a.context(ctx, "")
	})
	a.registerMFA(api)
	a.registerApplications(api)
	a.registerSessions(api)
	a.registerIdentity(api)
}
func (a AdminEndpoints) registerSessions(api huma.API) {
	huma.Get(api, "/api/v1/admin/sessions", func(ctx context.Context, input *AccountCookie) (*SessionsOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result := &SessionsOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			items, err := a.RP.Sessions.List(ctx, user.ID)
			result.Body.Items, result.Body.CurrentID = items, web.ID
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Delete(api, "/api/v1/admin/sessions/{id}", func(ctx context.Context, input *RevokeInput) (*struct{}, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error { return a.RP.Sessions.Revoke(ctx, user.ID, input.ID) })
		if err != nil {
			return nil, adminError(err)
		}
		return nil, nil
	})
}
