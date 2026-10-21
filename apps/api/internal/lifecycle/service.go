package lifecycle

import (
	"context"

	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
)

type Service struct {
	db       *ent.Client
	users    *identity.Service
	apps     *developer.Service
	protocol *authorization.Provider
}

func New(db *ent.Client, users *identity.Service, apps *developer.Service, protocol *authorization.Provider) *Service {
	return &Service{db, users, apps, protocol}
}

func (s *Service) Request(ctx context.Context, user identity.User, current string, revision int) (identity.Deletion, error) {
	if err := s.users.Reauthenticate(ctx, user.ID, current); err != nil {
		return identity.Deletion{}, err
	}
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return identity.Deletion{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := s.users.VerifyVersion(ctx, user.ID, user.SecurityVersion); err != nil {
		return identity.Deletion{}, err
	}
	if err := administration.OrdinaryTarget(ctx, s.db, user.ID); err != nil {
		return identity.Deletion{}, err
	}
	if err := s.apps.DisableOwner(ctx, user.ID); err != nil {
		return identity.Deletion{}, err
	}
	result, err := s.users.ScheduleDeletion(ctx, user, revision)
	if err != nil {
		return identity.Deletion{}, err
	}
	return result, unit.Commit()
}

func (s *Service) Cleanup(ctx context.Context) error {
	ids, err := s.users.DueDeletions(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.purge(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) purge(ctx context.Context, id string) error {
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	ready, err := s.users.LockDueDeletion(ctx, id)
	if err != nil || !ready {
		return err
	}
	if err := administration.OrdinaryTarget(ctx, s.db, id); err != nil {
		return err
	}
	if err := s.apps.PurgeOwner(ctx, id); err != nil {
		return err
	}
	if err := s.protocol.PurgeAccount(ctx, id); err != nil {
		return err
	}
	if err := s.users.PurgeDeletion(ctx, id); err != nil {
		return err
	}
	return unit.Commit()
}
