package httpserver

import (
	"context"
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/sessions"
)

type AuthorizationInput struct {
	AccountCookie
	ID string `path:"id" minLength:"43" maxLength:"43"`
}
type AuthorizationOutput struct {
	Body struct {
		authorization.AuthorizationDetails
		User *identity.User `json:"user,omitempty"`
	}
}
type AuthorizationDecisionInput struct {
	AccountWrite
	ID   string `path:"id" minLength:"43" maxLength:"43"`
	Body struct {
		Decision  string `json:"decision" enum:"approve,deny"`
		SubjectID string `json:"subject_id" minLength:"1" maxLength:"64"`
	}
}
type AuthorizationDecisionOutput struct {
	Body struct {
		RedirectTo string `json:"redirect_to"`
	}
}

func authorizationError(err error) error {
	if errors.Is(err, authorization.ErrInteraction) {
		return authorizationProblem(404, "Authorization request unavailable or expired", "authorization_request_unavailable")
	}
	if errors.Is(err, authorization.ErrLoginRequired) {
		return authorizationProblem(401, "Fresh sign-in required", "authorization_login_required")
	}
	return authorizationProblem(503, "Authorization service temporarily unavailable", "authorization_unavailable")
}

func authorizationProblem(status int, title, code string) error {
	return &AccountProblem{ErrorModel: huma.ErrorModel{Status: status, Title: title, Type: "about:blank"}, Code: code}
}

func (a AccountEndpoints) registerAuthorization(api huma.API) {
	huma.Get(api, "/api/v1/account/authorizations/{id}", func(ctx context.Context, input *AuthorizationInput) (*AuthorizationOutput, error) {
		if a.Authorization == nil || a.Sessions == nil || a.Identity == nil {
			return nil, huma.Error503ServiceUnavailable("Authorization service unavailable")
		}
		var authenticated time.Time
		var user *identity.User
		session, err := a.Sessions.Find(ctx, a.token(input.Cookie))
		if err == nil {
			found, err := a.Identity.User(ctx, session.UserID)
			if err == nil {
				user, authenticated = &found, session.CreatedAt
			} else if !errors.Is(err, identity.ErrMissing) {
				return nil, accountError(err)
			}
		} else if !errors.Is(err, sessions.ErrMissing) {
			return nil, accountError(err)
		}
		details, err := a.Authorization.Details(ctx, input.ID, a.Authorization.BrowserToken(input.Cookie), authenticated)
		if err != nil {
			return nil, authorizationError(err)
		}
		result := &AuthorizationOutput{}
		result.Body.AuthorizationDetails, result.Body.User = details, user
		return result, nil
	})
	huma.Post(api, "/api/v1/account/authorizations/{id}", func(ctx context.Context, input *AuthorizationDecisionInput) (*AuthorizationDecisionOutput, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if a.Authorization == nil {
			return nil, huma.Error503ServiceUnavailable("Authorization service unavailable")
		}
		user, session, err := a.user(ctx, input.Cookie)
		if err != nil {
			return nil, err
		}
		if user.ID != input.Body.SubjectID {
			return nil, authorizationProblem(409, "The signed-in account changed; reload the authorization request", "authorization_account_changed")
		}
		if err := a.rate(ctx, user.ID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		location, err := a.Authorization.Decide(ctx, input.ID, a.Authorization.BrowserToken(input.Cookie), user, session.CreatedAt, input.Body.Decision == "approve")
		if err != nil {
			return nil, authorizationError(err)
		}
		result := &AuthorizationDecisionOutput{}
		result.Body.RedirectTo = location
		return result, nil
	})
}
