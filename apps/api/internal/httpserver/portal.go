package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
)

type PortalContextBody struct {
	Authenticated bool           `json:"authenticated"`
	CSRFToken     string         `json:"csrf_token"`
	LoginURL      string         `json:"login_url"`
	User          *identity.User `json:"user,omitempty"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
}
type PortalContextOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      PortalContextBody
}
type PortalEndpoints struct{ Service *relyingparty.Service }

func (a PortalEndpoints) context(ctx context.Context, cookies string) (*PortalContextOutput, error) {
	if a.Service == nil {
		return nil, huma.Error503ServiceUnavailable("User site sign-in unavailable")
	}
	token := a.Service.CookieToken(cookies)
	result := &PortalContextOutput{Body: PortalContextBody{LoginURL: "/api/v1/portal/login"}}
	user, web, err := a.Service.User(ctx, token)
	if err == nil {
		result.Body.Authenticated = true
		result.Body.User = &user
		result.Body.ExpiresAt = &web.ExpiresAt
	} else if !errors.Is(err, sessions.ErrMissing) && !errors.Is(err, identity.ErrMissing) {
		return nil, huma.Error503ServiceUnavailable("User site session unavailable")
	} else if !sessions.ValidToken(token) || errors.Is(err, identity.ErrMissing) {
		token, err = sessions.RandomToken()
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("User site session unavailable")
		}
	}
	result.SetCookie = a.Service.Sessions.Cookie(token)
	result.Body.CSRFToken = a.Service.Sessions.CSRF(token)
	return result, nil
}
func (a PortalEndpoints) protect(input AccountWrite) error {
	if a.Service == nil {
		return huma.Error503ServiceUnavailable("User site sign-in unavailable")
	}
	if !a.Service.Sessions.Protect(a.Service.CookieToken(input.Cookie), input.Origin, input.CSRF) {
		return huma.Error403Forbidden("Request verification failed")
	}
	return nil
}
func (a PortalEndpoints) Register(api huma.API) {
	huma.Get(api, "/api/v1/portal/context", func(ctx context.Context, input *AccountCookie) (*PortalContextOutput, error) {
		return a.context(ctx, input.Cookie)
	})
	huma.Post(api, "/api/v1/portal/logout", func(ctx context.Context, input *AccountWrite) (*PortalContextOutput, error) {
		if err := a.protect(*input); err != nil {
			return nil, err
		}
		web, err := a.Service.Sessions.Find(ctx, a.Service.CookieToken(input.Cookie))
		if err != nil && !errors.Is(err, sessions.ErrMissing) {
			return nil, accountError(err)
		}
		if web.ID != "" {
			if err := a.Service.Sessions.Revoke(ctx, web.UserID, web.ID); err != nil {
				return nil, accountError(err)
			}
		}
		return a.context(ctx, "")
	})
	huma.Get(api, "/api/v1/portal/sessions", func(ctx context.Context, input *AccountCookie) (*SessionsOutput, error) {
		if a.Service == nil {
			return nil, huma.Error503ServiceUnavailable("User site sign-in unavailable")
		}
		user, web, err := a.Service.User(ctx, a.Service.CookieToken(input.Cookie))
		if err != nil {
			return nil, accountError(err)
		}
		items, err := a.Service.Sessions.List(ctx, user.ID)
		if err != nil {
			return nil, accountError(err)
		}
		result := &SessionsOutput{}
		result.Body.Items, result.Body.CurrentID = items, web.ID
		return result, nil
	})
	huma.Delete(api, "/api/v1/portal/sessions/{id}", func(ctx context.Context, input *RevokeInput) (*struct{}, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		user, _, err := a.Service.User(ctx, a.Service.CookieToken(input.Cookie))
		if err != nil {
			return nil, accountError(err)
		}
		if err := a.Service.Sessions.Revoke(ctx, user.ID, input.ID); err != nil {
			return nil, accountError(err)
		}
		return nil, nil
	})
}
