package httpserver

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/developer"
)

type ApplicationTestMemberInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision int    `json:"revision" minimum:"1"`
		UserID   string `json:"user_id" format:"uuid"`
	}
}

type ApplicationTestMemberRemoveInput struct {
	ApplicationRevisionInput
	UserID string `path:"user_id" format:"uuid"`
}

type ApplicationTestMembersOutput struct{ Body developer.TestMembers }

func (a DeveloperEndpoints) registerTestMembers(api huma.API) {
	huma.Get(api, "/api/v1/developer/applications/{id}/test-members", func(ctx context.Context, input *ApplicationGetInput) (*ApplicationTestMembersOutput, error) {
		user, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.TestMembers(ctx, user.ID, input.ID)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationTestMembersOutput{result}, nil
	})
	huma.Post(api, "/api/v1/developer/applications/{id}/test-members", func(ctx context.Context, input *ApplicationTestMemberInput) (*ApplicationTestMembersOutput, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.ChangeTestMember(ctx, user, input.ID, input.Body.Revision, input.Body.UserID, true)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationTestMembersOutput{result}, nil
	})
	huma.Delete(api, "/api/v1/developer/applications/{id}/test-members/{user_id}", func(ctx context.Context, input *ApplicationTestMemberRemoveInput) (*ApplicationTestMembersOutput, error) {
		user, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.ChangeTestMember(ctx, user, input.ID, input.Body.Revision, input.UserID, false)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationTestMembersOutput{result}, nil
	})
}
