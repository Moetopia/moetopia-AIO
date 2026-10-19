package httpserver

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/administration"
)

type AdminPasswordInput struct {
	AccountWrite
	Body struct {
		CurrentPassword string `json:"current_password" minLength:"1" maxLength:"256"`
	}
}
type AdminCodeInput struct {
	AccountWrite
	Body struct {
		Code string `json:"code" minLength:"6" maxLength:"6"`
	}
}
type AdminRecoveryInput struct {
	AccountWrite
	Body struct {
		CurrentPassword string `json:"current_password" minLength:"1" maxLength:"256"`
		RecoveryCode    string `json:"recovery_code" minLength:"32" maxLength:"32"`
	}
}
type AdminEnrollmentOutput struct{ Body administration.Enrollment }
type AdminRecoveryOutput struct {
	Body struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
}

func (a AdminEndpoints) registerMFA(api huma.API) {
	huma.Post(api, "/api/v1/admin/mfa/enrollment", func(ctx context.Context, input *AdminPasswordInput) (*AdminEnrollmentOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.BeginEnrollment(ctx, user, web, input.Body.CurrentPassword)
		if err != nil {
			return nil, adminError(err)
		}
		return &AdminEnrollmentOutput{result}, nil
	})
	huma.Post(api, "/api/v1/admin/mfa/enrollment/confirm", func(ctx context.Context, input *AdminCodeInput) (*AdminContextOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		handle, codes, err := a.Service.ConfirmEnrollment(ctx, user, web, input.Body.Code)
		if err != nil {
			return nil, adminError(err)
		}
		return a.rotated(ctx, handle, codes)
	})
	huma.Post(api, "/api/v1/admin/mfa/verify", func(ctx context.Context, input *AdminCodeInput) (*AdminContextOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		handle, err := a.Service.Verify(ctx, user, web, input.Body.Code)
		if err != nil {
			return nil, adminError(err)
		}
		return a.rotated(ctx, handle, nil)
	})
	huma.Post(api, "/api/v1/admin/mfa/recover", func(ctx context.Context, input *AdminRecoveryInput) (*AdminContextOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		handle, err := a.Service.Recover(ctx, user, web, input.Body.CurrentPassword, input.Body.RecoveryCode)
		if err != nil {
			return nil, adminError(err)
		}
		return a.rotated(ctx, handle, nil)
	})
	huma.Post(api, "/api/v1/admin/mfa/recovery-codes", func(ctx context.Context, input *AdminPasswordInput) (*AdminRecoveryOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		codes, err := a.Service.RegenerateRecovery(ctx, user, web, input.Body.CurrentPassword)
		if err != nil {
			return nil, adminError(err)
		}
		result := &AdminRecoveryOutput{}
		result.Body.RecoveryCodes = codes
		return result, nil
	})
}
