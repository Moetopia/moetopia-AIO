package httpserver

import (
	"context"
	"errors"
	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
)

type DeveloperEndpoints struct {
	Portal   PortalEndpoints
	Service  *developer.Service
	Accounts AccountEndpoints
}
type ApplicationCreateInput struct {
	AccountWrite
	Body struct {
		Public bool             `json:"public"`
		Config developer.Config `json:"config"`
	}
}
type ApplicationCreateOutput struct{ Body developer.Created }
type ApplicationOutput struct{ Body developer.Application }
type ApplicationsOutput struct{ Body developer.Applications }
type ApplicationGetInput struct {
	AccountCookie
	ID string `path:"id" format:"uuid"`
}
type ApplicationDraftInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision int              `json:"revision" minimum:"1"`
		Config   developer.Config `json:"config"`
	}
}
type ApplicationRevisionInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision int `json:"revision" minimum:"1"`
	}
}
type ApplicationAvailabilityInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision int  `json:"revision" minimum:"1"`
		Enabled  bool `json:"enabled"`
	}
}
type ApplicationSecretInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision        int    `json:"revision" minimum:"1"`
		CurrentPassword string `json:"current_password" maxLength:"256"`
	}
}
type ApplicationSecretOutput struct {
	Body struct {
		ClientSecret string `json:"client_secret"`
	}
}

func applicationError(err error) error {
	if errors.Is(err, developer.ErrMissing) {
		return authorizationProblem(404, "Application unavailable", "application_unavailable")
	}
	if errors.Is(err, developer.ErrConflict) {
		return authorizationProblem(409, "Application state changed; reload before editing", "application_conflict")
	}
	if errors.Is(err, developer.ErrInvalid) {
		return authorizationProblem(422, "Invalid application configuration", "application_invalid")
	}
	if errors.Is(err, developer.ErrQuota) {
		return authorizationProblem(409, "Application quota reached", "application_quota_reached")
	}
	return managementError(err)
}
func (a DeveloperEndpoints) actor(ctx context.Context, input AccountWrite, write bool) (identity.User, error) {
	if a.Service == nil {
		return identity.User{}, huma.Error503ServiceUnavailable("Developer service unavailable")
	}
	user, _, err := a.Portal.actor(ctx, input, write)
	if err != nil {
		return user, err
	}
	if write {
		if err := a.Accounts.rate(ctx, user.ID, clientIP(ctx), false); err != nil {
			return user, err
		}
	}
	return user, nil
}
func (a DeveloperEndpoints) Register(api huma.API) {
	a.registerHistory(api)
	a.registerTestMembers(api)
	a.registerStatistics(api)
	huma.Get(api, "/api/v1/developer/applications", func(ctx context.Context, input *GrantListInput) (*ApplicationsOutput, error) {
		user, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		items, err := a.Service.List(ctx, user.ID, input.Cursor)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationsOutput{items}, nil
	})
	huma.Post(api, "/api/v1/developer/applications", func(ctx context.Context, input *ApplicationCreateInput) (*ApplicationCreateOutput, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.Create(ctx, user, input.Body.Public, input.Body.Config)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationCreateOutput{result}, nil
	})
	huma.Get(api, "/api/v1/developer/applications/{id}", func(ctx context.Context, input *ApplicationGetInput) (*ApplicationOutput, error) {
		user, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.Detail(ctx, user.ID, input.ID)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationOutput{result}, nil
	})
	huma.Put(api, "/api/v1/developer/applications/{id}/draft", func(ctx context.Context, input *ApplicationDraftInput) (*ApplicationOutput, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.UpdateDraft(ctx, user, input.ID, input.Body.Revision, input.Body.Config)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationOutput{result}, nil
	})
	huma.Post(api, "/api/v1/developer/applications/{id}/submit", func(ctx context.Context, input *ApplicationRevisionInput) (*ApplicationOutput, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.Submit(ctx, user, input.ID, input.Body.Revision)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationOutput{result}, nil
	})
	huma.Post(api, "/api/v1/developer/applications/{id}/availability", func(ctx context.Context, input *ApplicationAvailabilityInput) (*ApplicationOutput, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.SetEnabled(ctx, user, input.ID, input.Body.Revision, input.Body.Enabled)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationOutput{result}, nil
	})
	huma.Post(api, "/api/v1/developer/applications/{id}/secret", func(ctx context.Context, input *ApplicationSecretInput) (*ApplicationSecretOutput, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := a.Accounts.Identity.Reauthenticate(ctx, user.ID, input.Body.CurrentPassword); err != nil {
			return nil, managementError(err)
		}
		secret, err := a.Service.RotateSecret(ctx, user, input.ID, input.Body.Revision)
		if err != nil {
			return nil, applicationError(err)
		}
		result := &ApplicationSecretOutput{}
		result.Body.ClientSecret = secret
		return result, nil
	})
	huma.Delete(api, "/api/v1/developer/applications/{id}", func(ctx context.Context, input *ApplicationRevisionInput) (*struct{}, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := a.Service.Delete(ctx, user, input.ID, input.Body.Revision); err != nil {
			return nil, applicationError(err)
		}
		return nil, nil
	})
}
