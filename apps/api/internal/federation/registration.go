package federation

import (
	"context"

	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
)

func (s *Service) Registration(ctx context.Context, completion, browser string) (RegistrationDetails, error) {
	row, data, err := s.read(ctx, completion, browser, "", "register", false)
	if err != nil {
		return RegistrationDetails{}, err
	}
	if data.Profile == nil || s.providers[data.Profile.Provider] == nil {
		return RegistrationDetails{}, ErrUnavailable
	}
	return RegistrationDetails{string(row.Provider), data.Profile.DisplayName, data.Profile.Email, row.ExpiresAt}, nil
}

func (s *Service) Complete(ctx context.Context, completion, browser, challenge, code, invitation, name string) (Result, error) {
	if _, err := s.Registration(ctx, completion, browser); err != nil {
		return Result{}, err
	}
	prepared, err := s.users.PrepareExternalRegistration(ctx, challenge, code, invitation, name)
	if err != nil {
		return Result{}, err
	}
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return Result{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	_, data, err := s.read(ctx, completion, browser, "", "register", true)
	if err != nil {
		return Result{}, err
	}
	if data.Profile == nil || s.providers[data.Profile.Provider] == nil {
		return Result{}, ErrUnavailable
	}
	profile := data.Profile
	user, err := s.users.RegisterExternal(ctx, prepared, identity.External{Provider: profile.Provider, Issuer: profile.Issuer, Subject: profile.Subject})
	if err != nil {
		return Result{}, err
	}
	if err := unit.Commit(); err != nil {
		return Result{}, err
	}
	return Result{User: &user, Continuation: data.Continuation}, nil
}
