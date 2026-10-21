package httpserver

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/sessions"
)

type LogoutDetailsOutput struct {
	Body struct {
		authorization.LogoutDetails
		User *identity.User `json:"user,omitempty"`
	}
}
type LogoutDecisionInput struct {
	AccountWrite
	ID   string `path:"id" minLength:"43" maxLength:"43"`
	Body struct {
		Decision  string `json:"decision" enum:"approve,deny"`
		SubjectID string `json:"subject_id" maxLength:"64"`
	}
}

func logoutError(err error) error {
	if errors.Is(err, authorization.ErrLogout) {
		return authorizationProblem(404, "Logout request unavailable or expired", "logout_request_unavailable")
	}
	if errors.Is(err, authorization.ErrLogoutAccount) {
		return authorizationProblem(409, "Signed-in account changed; reload the logout request", "logout_account_changed")
	}
	return accountError(err)
}

func (a AccountEndpoints) logoutSubject(ctx context.Context, cookies string) (*identity.User, sessions.Session, error) {
	if a.Identity == nil || a.Sessions == nil || a.Authorization == nil {
		return nil, sessions.Session{}, huma.Error503ServiceUnavailable("Logout service unavailable")
	}
	web, err := a.Sessions.Find(ctx, a.token(cookies))
	if errors.Is(err, sessions.ErrMissing) {
		return nil, sessions.Session{}, nil
	}
	if err != nil {
		return nil, web, accountError(err)
	}
	user, err := a.Identity.User(ctx, web.UserID)
	if errors.Is(err, identity.ErrMissing) {
		return nil, sessions.Session{}, nil
	}
	if err != nil {
		return nil, web, accountError(err)
	}
	return &user, web, nil
}

func (a AccountEndpoints) registerLogout(api huma.API) {
	huma.Get(api, "/api/v1/account/logout-requests/{id}", func(ctx context.Context, input *AuthorizationInput) (*LogoutDetailsOutput, error) {
		user, _, err := a.logoutSubject(ctx, input.Cookie)
		if err != nil {
			return nil, err
		}
		subject := ""
		if user != nil {
			subject = user.ID
		}
		details, err := a.Authorization.LogoutDetails(ctx, input.ID, a.Authorization.BrowserToken(input.Cookie), subject)
		if err != nil {
			return nil, logoutError(err)
		}
		result := &LogoutDetailsOutput{}
		result.Body.LogoutDetails, result.Body.User = details, user
		return result, nil
	})
	huma.Post(api, "/api/v1/account/logout-requests/{id}", func(ctx context.Context, input *LogoutDecisionInput) (*AuthorizationDecisionOutput, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		user, web, err := a.logoutSubject(ctx, input.Cookie)
		if err != nil {
			return nil, err
		}
		subject := ""
		if user != nil {
			subject = user.ID
		}
		if subject != input.Body.SubjectID {
			return nil, logoutError(authorization.ErrLogoutAccount)
		}
		if err := a.rate(ctx, "logout:"+a.token(input.Cookie), clientIP(ctx), false); err != nil {
			return nil, err
		}
		redirect, err := a.Authorization.ConfirmLogout(ctx, input.ID, a.Authorization.BrowserToken(input.Cookie), subject, web.ID, input.Body.Decision == "approve", a.Sessions)
		if err != nil {
			return nil, logoutError(err)
		}
		result := &AuthorizationDecisionOutput{}
		result.Body.RedirectTo = redirect
		return result, nil
	})
}
