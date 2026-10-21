package httpserver

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
)

type AdminApplicationsInput struct {
	AccountCookie
	Cursor string `query:"cursor" maxLength:"300"`
	Status string `query:"status" enum:"pending,all" default:"pending"`
}
type AdminReviewInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision int    `json:"revision" minimum:"1"`
		Approve  bool   `json:"approve"`
		Reason   string `json:"reason" maxLength:"500"`
	}
}
type AdminSuspendInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision  int    `json:"revision" minimum:"1"`
		Suspended bool   `json:"suspended"`
		Reason    string `json:"reason" maxLength:"500"`
	}
}

func (a AdminEndpoints) registerApplications(api huma.API) {
	a.registerApplicationHistory(api)
	huma.Get(api, "/api/v1/admin/applications", func(ctx context.Context, input *AdminApplicationsInput) (*ApplicationsOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		if a.Developers == nil {
			return nil, huma.Error503ServiceUnavailable("Application service unavailable")
		}
		result := &ApplicationsOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			items, err := a.Developers.AdminList(ctx, input.Status == "pending", input.Cursor)
			result.Body = items
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Get(api, "/api/v1/admin/applications/{id}", func(ctx context.Context, input *ApplicationGetInput) (*ApplicationOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		if a.Developers == nil {
			return nil, huma.Error503ServiceUnavailable("Application service unavailable")
		}
		result := &ApplicationOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Developers.AdminDetail(ctx, input.ID)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Post(api, "/api/v1/admin/applications/{id}/review", func(ctx context.Context, input *AdminReviewInput) (*ApplicationOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if a.Developers == nil {
			return nil, huma.Error503ServiceUnavailable("Application service unavailable")
		}
		result := &ApplicationOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Developers.Review(ctx, user.ID, input.ID, input.Body.Revision, input.Body.Approve, input.Body.Reason)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Post(api, "/api/v1/admin/applications/{id}/suspension", func(ctx context.Context, input *AdminSuspendInput) (*ApplicationOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if a.Developers == nil {
			return nil, huma.Error503ServiceUnavailable("Application service unavailable")
		}
		result := &ApplicationOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Developers.Suspend(ctx, user.ID, input.ID, input.Body.Revision, input.Body.Suspended, input.Body.Reason)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
}
