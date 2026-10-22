package httpserver

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/identity"
)

type AdminUsersInput struct {
	AccountCookie
	Query  string `query:"query" maxLength:"254"`
	Cursor string `query:"cursor" maxLength:"1000"`
}
type AdminUserAvailabilityInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Revision int    `json:"revision" minimum:"0"`
		Enabled  bool   `json:"enabled"`
		Reason   string `json:"reason" maxLength:"500"`
	}
}
type AdminPolicyInput struct {
	AccountWrite
	Body struct {
		Mode     string `json:"mode" enum:"invite,open"`
		Revision int    `json:"revision" minimum:"0"`
	}
}
type AdminInviteInput struct {
	AccountWrite
	Body struct {
		ExpiresInHours int `json:"expires_in_hours" minimum:"1" maximum:"720"`
	}
}
type AdminInviteRevokeInput struct {
	AccountWrite
	ID string `path:"id" minLength:"64" maxLength:"64"`
}
type AdminUsersOutput struct{ Body identity.ManagedUsers }
type AdminUserOutput struct{ Body identity.ManagedUser }
type AdminPolicyOutput struct{ Body identity.RegistrationPolicy }
type AdminInvitationsOutput struct{ Body identity.ManagedInvitations }
type AdminInvitationOutput struct{ Body identity.IssuedInvitation }
type AdminAuditOutput struct{ Body audit.AdminEvents }

func (a AdminEndpoints) registerIdentity(api huma.API) {
	huma.Get(api, "/api/v1/admin/users", func(ctx context.Context, input *AdminUsersInput) (*AdminUsersOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result := &AdminUsersOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Accounts.Identity.AdminUsers(ctx, input.Query, input.Cursor)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Post(api, "/api/v1/admin/users/{id}/availability", func(ctx context.Context, input *AdminUserAvailabilityInput) (*AdminUserOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result := &AdminUserOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			if err := a.Service.OrdinaryTarget(ctx, input.ID); err != nil {
				return err
			}
			var err error
			result.Body, err = a.Accounts.Identity.AdminSetEnabled(ctx, user.ID, input.ID, input.Body.Revision, input.Body.Enabled, input.Body.Reason)
			return err
		}, input.ID)
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Get(api, "/api/v1/admin/registration-policy", func(ctx context.Context, input *AccountCookie) (*AdminPolicyOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result := &AdminPolicyOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Accounts.Identity.RegistrationPolicy(ctx)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Put(api, "/api/v1/admin/registration-policy", func(ctx context.Context, input *AdminPolicyInput) (*AdminPolicyOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result := &AdminPolicyOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Accounts.Identity.SetRegistrationPolicy(ctx, user.ID, input.Body.Mode, input.Body.Revision)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Get(api, "/api/v1/admin/invitations", func(ctx context.Context, input *GrantListInput) (*AdminInvitationsOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result := &AdminInvitationsOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Accounts.Identity.AdminInvitations(ctx, input.Cursor)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Post(api, "/api/v1/admin/invitations", func(ctx context.Context, input *AdminInviteInput) (*AdminInvitationOutput, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		result := &AdminInvitationOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Accounts.Identity.AdminIssueInvitation(ctx, user.ID, input.Body.ExpiresInHours)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Delete(api, "/api/v1/admin/invitations/{id}", func(ctx context.Context, input *AdminInviteRevokeInput) (*struct{}, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			return a.Accounts.Identity.AdminRevokeInvitation(ctx, user.ID, input.ID)
		})
		if err != nil {
			return nil, adminError(err)
		}
		return nil, nil
	})
	huma.Get(api, "/api/v1/admin/audit-events", func(ctx context.Context, input *GrantListInput) (*AdminAuditOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result := &AdminAuditOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Service.AuditEvents(ctx, input.Cursor)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
}
