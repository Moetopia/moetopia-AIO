package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"moetopia.local/aio/api/internal/persistence/ent/externalidentity"
)

var ErrExternalConflict = errors.New("external identity is already linked")

type External struct{ Provider, Issuer, Subject string }
type ExternalBinding struct {
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}
type ExternalRegistration struct{ verificationID, email, name, invite string }

func validExternal(value External) bool {
	return value.Subject != "" && len(value.Subject) <= 255 && (value.Provider == "github" && value.Issuer == "https://github.com" || value.Provider == "google" && value.Issuer == "https://accounts.google.com")
}

func (s *Service) ExternalUser(ctx context.Context, binding External) (User, error) {
	if !validExternal(binding) {
		return User{}, ErrInvalid
	}
	row, err := persistence.Client(ctx, s.store.db).ExternalIdentity.Query().Where(externalidentity.ProviderEQ(externalidentity.Provider(binding.Provider)), externalidentity.Issuer(binding.Issuer), externalidentity.Subject(binding.Subject)).Only(ctx)
	if ent.IsNotFound(err) {
		return User{}, ErrMissing
	}
	if err != nil {
		return User{}, err
	}
	user, err := s.User(ctx, row.UserID)
	if errors.Is(err, ErrMissing) {
		return User{}, ErrCredentials
	}
	return user, err
}

func (s *Service) ExternalBindings(ctx context.Context, id string) ([]ExternalBinding, error) {
	rows, err := persistence.Client(ctx, s.store.db).ExternalIdentity.Query().Where(externalidentity.UserID(id)).Order(ent.Asc(externalidentity.FieldProvider)).Select(externalidentity.FieldProvider, externalidentity.FieldCreatedAt).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ExternalBinding, 0, len(rows))
	for _, row := range rows {
		items = append(items, ExternalBinding{string(row.Provider), row.CreatedAt})
	}
	return items, nil
}

func (s *Service) PrepareExternalRegistration(ctx context.Context, id, code, invite, name string) (ExternalRegistration, error) {
	verification, name, err := s.registration(ctx, id, code, name)
	if err != nil {
		return ExternalRegistration{}, err
	}
	return ExternalRegistration{id, verification.Email, name, invitationHash(strings.TrimSpace(invite))}, nil
}

func (s *Service) RegisterExternal(ctx context.Context, input ExternalRegistration, binding External) (User, error) {
	if input.verificationID == "" || !validExternal(binding) {
		return User{}, ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return User{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	id, err := s.store.register(ctx, input.verificationID, input.email, input.name, "!external-only", input.invite, false)
	if err != nil {
		return User{}, err
	}
	if err := s.createExternal(ctx, unit.Client(), id, binding); err != nil {
		return User{}, err
	}
	user, err := s.User(ctx, id)
	if err != nil {
		return User{}, err
	}
	return user, unit.Commit()
}

func (s *Service) createExternal(ctx context.Context, db *ent.Client, id string, binding External) error {
	err := db.ExternalIdentity.Create().SetID(uuid.NewString()).SetUserID(id).SetProvider(externalidentity.Provider(binding.Provider)).SetIssuer(binding.Issuer).SetSubject(binding.Subject).Exec(ctx)
	if ent.IsConstraintError(err) {
		return ErrExternalConflict
	}
	return err
}

func (s *Service) BindExternal(ctx context.Context, user User, binding External) error {
	if !validExternal(binding) {
		return ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := s.VerifyVersion(ctx, user.ID, user.SecurityVersion); err != nil {
		return err
	}
	row, err := unit.AccountUser.Get(ctx, user.ID)
	if err != nil {
		return err
	}
	if !row.HasPassword {
		return ErrCredentials
	}
	if err := s.createExternal(ctx, unit.Client(), user.ID, binding); err != nil {
		return err
	}
	if err := unit.AccountUser.UpdateOneID(user.ID).AddSecurityVersion(1).AddRevision(1).Exec(ctx); err != nil {
		return err
	}
	if err := s.securityChanged(ctx, unit.Client(), row, "external_identity.bound"); err != nil {
		return err
	}
	return unit.Commit()
}

func (s *Service) UnbindExternal(ctx context.Context, user User, provider, current string) error {
	if provider != "github" && provider != "google" {
		return ErrInvalid
	}
	before, err := s.credentialsByID(ctx, user.ID, current)
	if err != nil {
		return err
	}
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := s.VerifyVersion(ctx, user.ID, before.SecurityVersion); err != nil {
		return err
	}
	count, err := unit.ExternalIdentity.Delete().Where(externalidentity.UserID(user.ID), externalidentity.ProviderEQ(externalidentity.Provider(provider))).Exec(ctx)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrMissing
	}
	if err := unit.AccountUser.UpdateOneID(user.ID).Where(accountuser.HasPassword(true)).AddSecurityVersion(1).AddRevision(1).Exec(ctx); err != nil {
		return err
	}
	if err := s.securityChanged(ctx, unit.Client(), before, "external_identity.unbound"); err != nil {
		return err
	}
	return unit.Commit()
}
