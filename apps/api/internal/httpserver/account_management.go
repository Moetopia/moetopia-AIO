package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/sessions"
)

type ProfileInput struct {
	AccountWrite
	Body struct {
		DisplayName string `json:"display_name" minLength:"1" maxLength:"80"`
		Bio         string `json:"bio" maxLength:"500"`
		Revision    int    `json:"revision" minimum:"0"`
	}
}
type UserOutput struct{ Body identity.User }
type AvatarInput struct {
	AccountWrite
	Body struct {
		ImageBase64 string `json:"image_base64" maxLength:"700000"`
		Revision    int    `json:"revision" minimum:"0"`
	}
}
type PasswordChangeInput struct {
	AccountWrite
	Body struct {
		CurrentPassword string `json:"current_password" maxLength:"256"`
		NewPassword     string `json:"new_password" minLength:"15" maxLength:"256"`
	}
}
type EmailChangeInput struct {
	AccountWrite
	Body struct {
		CurrentPassword string `json:"current_password" maxLength:"256"`
		Email           string `json:"email" maxLength:"254"`
		Locale          string `json:"locale" enum:"zh-CN,en"`
	}
}
type EmailConfirmInput struct {
	AccountWrite
	Body struct {
		CurrentPassword string `json:"current_password" maxLength:"256"`
		ChallengeID     string `json:"challenge_id" minLength:"36" maxLength:"36"`
		Code            string `json:"code" minLength:"8" maxLength:"8"`
	}
}
type PasswordResetInput struct {
	AccountWrite
	Body struct {
		ChallengeID string `json:"challenge_id" minLength:"36" maxLength:"36"`
		Code        string `json:"code" minLength:"8" maxLength:"8"`
		NewPassword string `json:"new_password" minLength:"15" maxLength:"256"`
	}
}
type GrantListInput struct {
	AccountCookie
	Cursor string `query:"cursor" maxLength:"300"`
}
type UserGrantsOutput struct{ Body authorization.UserGrants }
type SecurityEventsOutput struct {
	Body struct {
		Items []audit.Event `json:"items"`
	}
}

func managementError(err error) error {
	if errors.Is(err, identity.ErrConflict) {
		return authorizationProblem(409, "Account changed; reload before saving", "account_revision_conflict")
	}
	if errors.Is(err, identity.ErrInvalid) {
		return authorizationProblem(422, "Invalid or expired account details", "account_details_invalid")
	}
	if errors.Is(err, identity.ErrCredentials) {
		return authorizationProblem(401, "Current password verification failed", "reauthentication_failed")
	}
	if errors.Is(err, authorization.ErrGrantMissing) {
		return authorizationProblem(404, "Authorization unavailable", "authorization_unavailable")
	}
	if errors.Is(err, authorization.ErrGrantCursor) {
		return authorizationProblem(422, "Invalid cursor", "cursor_invalid")
	}
	return accountError(err)
}
func (a PortalEndpoints) actor(ctx context.Context, input AccountWrite, write bool) (identity.User, sessions.Session, error) {
	if a.Service == nil {
		return identity.User{}, sessions.Session{}, huma.Error503ServiceUnavailable("User site unavailable")
	}
	if write {
		if err := a.protect(input); err != nil {
			return identity.User{}, sessions.Session{}, err
		}
	}
	user, web, err := a.Service.User(ctx, a.Service.CookieToken(input.Cookie))
	if err != nil {
		return user, web, accountError(err)
	}
	return user, web, nil
}
func (a PortalEndpoints) registerManagement(api huma.API, accounts AccountEndpoints) {
	huma.Patch(api, "/api/v1/portal/profile", func(ctx context.Context, input *ProfileInput) (*UserOutput, error) {
		user, _, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, user.ID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		updated, err := accounts.Identity.UpdateProfile(ctx, user.ID, input.Body.DisplayName, input.Body.Bio, input.Body.Revision)
		if err != nil {
			return nil, managementError(err)
		}
		return &UserOutput{updated}, nil
	})
	huma.Put(api, "/api/v1/portal/avatar", func(ctx context.Context, input *AvatarInput) (*UserOutput, error) {
		user, _, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, user.ID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		updated, err := accounts.Identity.UpdateAvatar(ctx, user.ID, input.Body.ImageBase64, input.Body.Revision)
		if err != nil {
			return nil, managementError(err)
		}
		return &UserOutput{updated}, nil
	})
	huma.Post(api, "/api/v1/portal/password", func(ctx context.Context, input *PasswordChangeInput) (*struct{}, error) {
		user, _, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, user.Email, clientIP(ctx), false); err != nil {
			return nil, err
		}
		if err := accounts.Identity.ChangePassword(ctx, user.ID, input.Body.CurrentPassword, input.Body.NewPassword); err != nil {
			return nil, managementError(err)
		}
		return nil, nil
	})
	huma.Post(api, "/api/v1/portal/email-changes", func(ctx context.Context, input *EmailChangeInput) (*VerificationOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, input.Body.Email, clientIP(ctx), true); err != nil {
			return nil, err
		}
		id, err := accounts.Identity.BeginEmailChange(ctx, user.ID, web.ID, input.Body.CurrentPassword, input.Body.Email, input.Body.Locale)
		if err != nil {
			return nil, managementError(err)
		}
		result := &VerificationOutput{}
		result.Body.ChallengeID = id
		return result, nil
	})
	huma.Post(api, "/api/v1/portal/email-changes/confirm", func(ctx context.Context, input *EmailConfirmInput) (*struct{}, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, input.Body.ChallengeID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		if err := accounts.Identity.ConfirmEmailChange(ctx, user.ID, web.ID, input.Body.ChallengeID, input.Body.Code, input.Body.CurrentPassword); err != nil {
			return nil, managementError(err)
		}
		return nil, nil
	})
	huma.Get(api, "/api/v1/portal/authorizations", func(ctx context.Context, input *GrantListInput) (*UserGrantsOutput, error) {
		user, _, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		if accounts.Authorization == nil {
			return nil, huma.Error503ServiceUnavailable("Authorization service unavailable")
		}
		result, err := accounts.Authorization.UserGrants(ctx, user.ID, user.SecurityVersion, input.Cursor)
		if err != nil {
			return nil, managementError(err)
		}
		return &UserGrantsOutput{result}, nil
	})
	huma.Delete(api, "/api/v1/portal/authorizations/{id}", func(ctx context.Context, input *RevokeInput) (*struct{}, error) {
		user, _, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if accounts.Authorization == nil {
			return nil, huma.Error503ServiceUnavailable("Authorization service unavailable")
		}
		if err := accounts.Authorization.RevokeUserGrant(ctx, user.ID, input.ID); err != nil {
			return nil, managementError(err)
		}
		return nil, nil
	})
	huma.Get(api, "/api/v1/portal/security-events", func(ctx context.Context, input *AccountCookie) (*SecurityEventsOutput, error) {
		user, _, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		items, err := accounts.Identity.SecurityEvents(ctx, user.ID)
		if err != nil {
			return nil, accountError(err)
		}
		result := &SecurityEventsOutput{}
		result.Body.Items = items
		return result, nil
	})
}
func (a AccountEndpoints) registerRecovery(api huma.API) {
	huma.Post(api, "/api/v1/account/password-resets", func(ctx context.Context, input *VerificationInput) (*VerificationOutput, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.rate(ctx, input.Body.Email, clientIP(ctx), true); err != nil {
			return nil, err
		}
		id, err := a.Identity.BeginReset(ctx, input.Body.Email, input.Body.Locale)
		if err != nil {
			return nil, managementError(err)
		}
		result := &VerificationOutput{}
		result.Body.ChallengeID = id
		return result, nil
	})
	huma.Post(api, "/api/v1/account/password-resets/confirm", func(ctx context.Context, input *PasswordResetInput) (*struct{}, error) {
		if err := a.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := a.rate(ctx, input.Body.ChallengeID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		if err := a.Identity.ResetPassword(ctx, input.Body.ChallengeID, input.Body.Code, input.Body.NewPassword); err != nil {
			return nil, managementError(err)
		}
		return nil, nil
	})
}
func (a AccountEndpoints) avatar(w http.ResponseWriter, r *http.Request) {
	if a.Identity == nil {
		http.Error(w, "Unavailable", 503)
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		http.NotFound(w, r)
		return
	}
	data, err := a.Identity.Avatar(r.Context(), id)
	if errors.Is(err, identity.ErrMissing) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Unavailable", 503)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	_, _ = w.Write(data)
}
