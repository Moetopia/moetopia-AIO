package httpserver

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/lifecycle"
)

type DeletionInput struct {
	AccountWrite
	Body struct {
		CurrentPassword string `json:"current_password" maxLength:"256"`
		Revision        int    `json:"revision" minimum:"0"`
	}
}
type DeletionOutput struct{ Body identity.Deletion }
type DeletionCancelInput struct {
	AccountWrite
	Body struct {
		ChallengeID     string `json:"challenge_id" format:"uuid"`
		Code            string `json:"code" minLength:"8" maxLength:"8"`
		CurrentPassword string `json:"current_password" maxLength:"256"`
	}
}

func registerDeletion(api huma.API, portal PortalEndpoints, accounts AccountEndpoints, service *lifecycle.Service) {
	huma.Post(api, "/api/v1/portal/deletion", func(ctx context.Context, input *DeletionInput) (*DeletionOutput, error) {
		if service == nil {
			return nil, huma.Error503ServiceUnavailable("Account lifecycle unavailable")
		}
		user, _, err := portal.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, user.ID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		result, err := service.Request(ctx, user, input.Body.CurrentPassword, input.Body.Revision)
		if errors.Is(err, administration.ErrProtected) {
			return nil, adminError(err)
		}
		if errors.Is(err, developer.ErrPublished) {
			return nil, authorizationProblem(409, "Disable published applications before deleting the account", "account_published_applications")
		}
		if err != nil {
			return nil, managementError(err)
		}
		return &DeletionOutput{result}, nil
	})
	huma.Post(api, "/api/v1/account/deletion-cancellations", func(ctx context.Context, input *VerificationInput) (*VerificationOutput, error) {
		if err := accounts.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, input.Body.Email, clientIP(ctx), true); err != nil {
			return nil, err
		}
		id, err := accounts.Identity.BeginDeletionCancel(ctx, input.Body.Email, input.Body.Locale)
		if err != nil {
			return nil, managementError(err)
		}
		result := &VerificationOutput{}
		result.Body.ChallengeID = id
		return result, nil
	})
	huma.Post(api, "/api/v1/account/deletion-cancellations/confirm", func(ctx context.Context, input *DeletionCancelInput) (*struct{}, error) {
		if err := accounts.protect(input.AccountWrite); err != nil {
			return nil, err
		}
		if err := accounts.rate(ctx, input.Body.ChallengeID, clientIP(ctx), false); err != nil {
			return nil, err
		}
		if err := accounts.Identity.CancelDeletion(ctx, input.Body.ChallengeID, input.Body.Code, input.Body.CurrentPassword); err != nil {
			return nil, managementError(err)
		}
		return nil, nil
	})
}
