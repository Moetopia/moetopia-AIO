package httpserver

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type AdminApplicationDeleteInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision int    `json:"revision" minimum:"1"`
		Reason   string `json:"reason" minLength:"1" maxLength:"500"`
	}
}

func (a AdminEndpoints) registerApplicationDeletion(api huma.API) {
	huma.Delete(api, "/api/v1/admin/applications/{id}", func(ctx context.Context, input *AdminApplicationDeleteInput) (*struct{}, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if a.Developers == nil {
			return nil, huma.Error503ServiceUnavailable("Application service unavailable")
		}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			return a.Developers.AdminDelete(ctx, user.ID, input.ID, input.Body.Revision, input.Body.Reason)
		})
		if err != nil {
			return nil, adminError(err)
		}
		return nil, nil
	})
}
