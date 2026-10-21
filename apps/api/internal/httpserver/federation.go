package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/federation"
	"moetopia.local/aio/api/internal/identity"
)

type FederationEndpoints struct {
	Service  *federation.Service
	Origin   string
	Accounts AccountEndpoints
	Portal   PortalEndpoints
}

type FederationStartInput struct {
	AccountWrite
	Provider string `path:"provider" enum:"github,google"`
	Body     struct {
		AuthorizationRequest string `json:"authorization_request" maxLength:"43" required:"false"`
		Continue             string `json:"continue" enum:",developer-applications" required:"false"`
	}
}
type FederationRedirect struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      struct {
		RedirectTo string `json:"redirect_to"`
	}
}
type FederationBindingInput struct {
	AccountWrite
	Provider string `path:"provider" enum:"github,google"`
	Body     struct {
		CurrentPassword string `json:"current_password" maxLength:"256"`
	}
}
type FederationRegistrationInput struct {
	AccountCookie
	Completion string `query:"completion" minLength:"43" maxLength:"43"`
}
type FederationCompleteInput struct {
	AccountWrite
	Body struct {
		Completion  string `json:"completion" minLength:"43" maxLength:"43"`
		ChallengeID string `json:"challenge_id" minLength:"36" maxLength:"36"`
		Code        string `json:"code" minLength:"8" maxLength:"8"`
		Invitation  string `json:"invitation" maxLength:"128"`
		DisplayName string `json:"display_name" minLength:"1" maxLength:"80"`
	}
}
type FederationRegistrationOutput struct {
	Body federation.RegistrationDetails
}
type FederationProviders struct {
	Body struct {
		Items []string `json:"items"`
	}
}
type FederationBindings struct {
	Body struct {
		Items []identity.ExternalBinding `json:"items"`
	}
}

func federationError(err error) error {
	code, status := "external_identity_invalid", 400
	switch {
	case errors.Is(err, federation.ErrUnavailable):
		code, status = "external_provider_unavailable", 503
	case errors.Is(err, identity.ErrExternalConflict):
		code, status = "external_identity_conflict", 409
	case errors.Is(err, federation.ErrFlow), errors.Is(err, federation.ErrProvider):
	case errors.Is(err, identity.ErrInvalid), errors.Is(err, identity.ErrCredentials), errors.Is(err, identity.ErrMissing):
		return accountError(err)
	default:
		code, status = "external_identity_unavailable", 503
	}
	return &AccountProblem{ErrorModel: huma.ErrorModel{Status: status, Title: http.StatusText(status), Detail: "External identity request could not be completed", Type: "about:blank"}, Code: code}
}

func (a FederationEndpoints) cookieName() string {
	if strings.HasPrefix(a.Origin, "https://") {
		return "__Host-moetopia_federation"
	}
	return "moetopia_federation"
}
func (a FederationEndpoints) browser(cookies string) string {
	req := &http.Request{Header: http.Header{"Cookie": []string{cookies}}}
	cookie, err := req.Cookie(a.cookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}
func (a FederationEndpoints) startOutput(started federation.Started) *FederationRedirect {
	result := &FederationRedirect{SetCookie: http.Cookie{Name: a.cookieName(), Value: started.Browser, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(a.Origin, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 3600}}
	result.Body.RedirectTo = started.RedirectURL
	return result
}
func (a FederationEndpoints) destination(result federation.Result) string {
	if result.Continuation.AuthorizationRequest != "" {
		return a.Origin + "/auth/authorize?authorization_request=" + result.Continuation.AuthorizationRequest
	}
	path := "/profile"
	if result.Continuation.Developer {
		path = "/developer/applications"
	}
	return a.Origin + "/api/v1/portal/login?return_to=" + path
}
func (a FederationEndpoints) available() error {
	if a.Service == nil {
		return federationError(federation.ErrUnavailable)
	}
	return nil
}

func (a FederationEndpoints) Register(api huma.API) {
	huma.Get(api, "/api/v1/account/federation/providers", func(context.Context, *struct{}) (*FederationProviders, error) {
		result := &FederationProviders{}
		result.Body.Items = []string{}
		if a.Service != nil {
			result.Body.Items = a.Service.Providers()
		}
		return result, nil
	})
	huma.Post(api, "/api/v1/account/federation/{provider}", func(ctx context.Context, input *FederationStartInput) (*FederationRedirect, error) {
		if err := a.Accounts.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.available(); err != nil {
			return nil, err
		}
		if err := a.Accounts.rate(ctx, "federation:"+input.Provider+":"+a.Accounts.token(input.Cookie), clientIP(ctx), false); err != nil {
			return nil, err
		}
		started, err := a.Service.Begin(ctx, input.Provider, a.browser(input.Cookie), federation.Continuation{AuthorizationRequest: input.Body.AuthorizationRequest, Developer: input.Body.Continue == "developer-applications"}, nil)
		if err != nil {
			return nil, federationError(err)
		}
		return a.startOutput(started), nil
	})
	huma.Get(api, "/api/v1/account/federation/registration", func(ctx context.Context, input *FederationRegistrationInput) (*FederationRegistrationOutput, error) {
		if err := a.available(); err != nil {
			return nil, err
		}
		details, err := a.Service.Registration(ctx, input.Completion, a.browser(input.Cookie))
		if err != nil {
			return nil, federationError(err)
		}
		return &FederationRegistrationOutput{Body: details}, nil
	})
	huma.Post(api, "/api/v1/account/federation/registration", func(ctx context.Context, input *FederationCompleteInput) (*FederationRedirect, error) {
		if err := a.Accounts.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.available(); err != nil {
			return nil, err
		}
		if err := a.Accounts.rate(ctx, input.Body.ChallengeID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		result, err := a.Service.Complete(ctx, input.Body.Completion, a.browser(input.Cookie), input.Body.ChallengeID, input.Body.Code, input.Body.Invitation, input.Body.DisplayName)
		if err != nil {
			return nil, federationError(err)
		}
		token, err := a.Accounts.Sessions.CreateVerified(ctx, result.User.ID, a.Accounts.token(input.Cookie), result.User.SecurityVersion, a.Accounts.Identity)
		if err != nil {
			return nil, accountError(err)
		}
		output := &FederationRedirect{SetCookie: a.Accounts.Sessions.Cookie(token)}
		output.Body.RedirectTo = a.destination(result)
		return output, nil
	})
	a.registerBindings(api)
}

func (a FederationEndpoints) registerBindings(api huma.API) {
	huma.Get(api, "/api/v1/portal/identities", func(ctx context.Context, input *AccountCookie) (*FederationBindings, error) {
		if a.Portal.Service == nil {
			return nil, accountError(identity.ErrMissing)
		}
		user, _, err := a.Portal.Service.User(ctx, a.Portal.Service.CookieToken(input.Cookie))
		if err != nil {
			return nil, accountError(err)
		}
		items, err := a.Accounts.Identity.ExternalBindings(ctx, user.ID)
		if err != nil {
			return nil, accountError(err)
		}
		output := &FederationBindings{}
		output.Body.Items = items
		return output, nil
	})
	huma.Post(api, "/api/v1/portal/identities/{provider}", func(ctx context.Context, input *FederationBindingInput) (*FederationRedirect, error) {
		if err := a.Portal.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.available(); err != nil {
			return nil, err
		}
		user, web, err := a.Portal.Service.User(ctx, a.Portal.Service.CookieToken(input.Cookie))
		if err != nil {
			return nil, accountError(err)
		}
		if err := a.Accounts.rate(ctx, user.ID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		if err := a.Accounts.Identity.Reauthenticate(ctx, user.ID, input.Body.CurrentPassword); err != nil {
			return nil, accountError(err)
		}
		started, err := a.Service.Begin(ctx, input.Provider, a.browser(input.Cookie), federation.Continuation{}, &federation.Binding{User: user, SessionID: web.ID})
		if err != nil {
			return nil, federationError(err)
		}
		return a.startOutput(started), nil
	})
	huma.Delete(api, "/api/v1/portal/identities/{provider}", func(ctx context.Context, input *FederationBindingInput) (*struct{}, error) {
		if err := a.Portal.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		user, _, err := a.Portal.Service.User(ctx, a.Portal.Service.CookieToken(input.Cookie))
		if err != nil {
			return nil, accountError(err)
		}
		if err := a.Accounts.rate(ctx, user.ID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		if err := a.Accounts.Identity.UnbindExternal(ctx, user, input.Provider, input.Body.CurrentPassword); err != nil {
			return nil, federationError(err)
		}
		return nil, nil
	})
}
