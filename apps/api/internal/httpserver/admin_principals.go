package httpserver

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/administration"
)

type AdminPrincipalsInput struct {
	AccountCookie
	After string `query:"after" maxLength:"36"`
}
type AdminPrincipalsOutput struct{ Body administration.Principals }
type AdminPrincipalOutput struct{ Body administration.Principal }
type AdminPrincipalChangeInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision        int    `json:"revision" minimum:"0"`
		Enabled         bool   `json:"enabled"`
		CurrentPassword string `json:"current_password" maxLength:"256"`
		Reason          string `json:"reason" minLength:"1" maxLength:"500"`
	}
}
type AdminEmergencyInput struct {
	AccountWrite
	Body struct {
		CurrentPassword string `json:"current_password" maxLength:"256"`
		Token           string `json:"token" minLength:"64" maxLength:"64"`
	}
}

func (a AdminEndpoints) registerPrincipals(api huma.API) {
	huma.Get(api, "/api/v1/admin/principals", func(ctx context.Context, input *AdminPrincipalsInput) (*AdminPrincipalsOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result := &AdminPrincipalsOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Service.Principals(ctx, input.After)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Post(api, "/api/v1/admin/principals/{id}/availability", func(ctx context.Context, input *AdminPrincipalChangeInput) (*AdminPrincipalOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.ChangePrincipal(ctx, user, web, input.ID, input.Body.Revision, input.Body.Enabled, input.Body.CurrentPassword, input.Body.Reason)
		if err != nil {
			return nil, adminError(err)
		}
		return &AdminPrincipalOutput{result}, nil
	})
	huma.Post(api, "/api/v1/admin/mfa/emergency", func(ctx context.Context, input *AdminEmergencyInput) (*AdminContextOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		handle, err := a.Service.AcceptEmergency(ctx, user, web, input.Body.CurrentPassword, input.Body.Token)
		if err != nil {
			return nil, adminError(err)
		}
		return a.rotated(ctx, handle, nil)
	})
}
