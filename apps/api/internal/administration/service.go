package administration

import (
	"context"
	"errors"
	"time"

	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/adminelevation"
	"moetopia.local/aio/api/internal/persistence/ent/adminprincipal"
	"moetopia.local/aio/api/internal/sessions"
)

var ErrRole = errors.New("administrator role required")
var ErrFactor = errors.New("administrator MFA required")
var ErrProof = errors.New("invalid administrator verification")
var ErrState = errors.New("administrator security state changed")
var ErrProtected = errors.New("administrator account cannot be managed through ordinary account controls")

type Service struct {
	db       *ent.Client
	box      *cryptobox.Box
	users    *identity.Service
	outbox   *notifications.Outbox
	Sessions *sessions.Service
}
type State struct {
	Enrolled               bool       `json:"enrolled"`
	Verified               bool       `json:"verified"`
	VerifiedUntil          *time.Time `json:"verified_until,omitempty"`
	RecoveryCodesRemaining int        `json:"recovery_codes_remaining"`
}

func New(db *ent.Client, box *cryptobox.Box, users *identity.Service, outbox *notifications.Outbox, web *sessions.Service) *Service {
	return &Service{db, box, users, outbox, web}
}
func (s *Service) User(ctx context.Context, id string) (identity.User, error) {
	user, err := s.users.User(ctx, id)
	if err != nil {
		return user, err
	}
	ok, err := persistence.Client(ctx, s.db).AdminPrincipal.Query().Where(adminprincipal.ID(id), adminprincipal.Enabled(true)).Exist(ctx)
	if err != nil {
		return user, err
	}
	if !ok {
		return identity.User{}, identity.ErrMissing
	}
	return user, nil
}
func (s *Service) VerifyVersion(ctx context.Context, id string, version int) error {
	if err := s.users.VerifyVersion(ctx, id, version); err != nil {
		return err
	}
	_, err := persistence.Client(ctx, s.db).AdminPrincipal.Query().Where(adminprincipal.ID(id), adminprincipal.Enabled(true)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return identity.ErrMissing
	}
	return err
}
func (s *Service) State(ctx context.Context, userID, sessionID string) (State, error) {
	row, err := persistence.Client(ctx, s.db).AdminPrincipal.Query().Where(adminprincipal.ID(userID), adminprincipal.Enabled(true)).Only(ctx)
	if ent.IsNotFound(err) {
		return State{}, ErrRole
	}
	if err != nil {
		return State{}, err
	}
	state := State{Enrolled: len(row.Secret) > 0, RecoveryCodesRemaining: len(row.RecoveryHashes)}
	proof, err := persistence.Client(ctx, s.db).AdminElevation.Query().Where(adminelevation.ID(sessionID), adminelevation.UserID(userID), adminelevation.FactorVersion(row.FactorVersion), adminelevation.ExpiresAtGT(time.Now())).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return state, err
	}
	if proof != nil && state.Enrolled {
		state.Verified, state.VerifiedUntil = true, &proof.ExpiresAt
	}
	return state, nil
}
func (s *Service) begin(ctx context.Context, user identity.User, web sessions.Session) (context.Context, *persistence.Unit, *ent.AdminPrincipal, error) {
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return ctx, nil, nil, err
	}
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := s.users.VerifyVersion(ctx, user.ID, user.SecurityVersion); err != nil {
		_ = unit.Rollback()
		return ctx, nil, nil, err
	}
	row, err := unit.AdminPrincipal.Query().Where(adminprincipal.ID(user.ID), adminprincipal.Enabled(true)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		err = ErrRole
	}
	if err == nil {
		err = s.Sessions.AssertLive(ctx, user.ID, web.ID)
	}
	if err != nil {
		_ = unit.Rollback()
		return ctx, nil, nil, err
	}
	return ctx, unit, row, nil
}
func (s *Service) Run(ctx context.Context, user identity.User, web sessions.Session, command func(context.Context) error) error {
	ctx, unit, _, err := s.begin(ctx, user, web)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	state, err := s.State(ctx, user.ID, web.ID)
	if err != nil {
		return err
	}
	if !state.Verified {
		return ErrFactor
	}
	if err := command(ctx); err != nil {
		return err
	}
	return unit.Commit()
}
func (s *Service) notify(ctx context.Context, db *ent.Client, user identity.User, action string) error {
	return s.outbox.Enqueue(ctx, db, notifications.Message{To: user.Email, Subject: "Moetopia administrator security update", Text: "Your administrator security settings changed (" + action + "). If this was not you, contact the operator immediately.\n你的管理员安全设置已变更，如非本人操作，请立即联系维护者。"}, time.Now().Add(24*time.Hour))
}
func (s *Service) Cleanup(ctx context.Context) error {
	if _, err := s.db.AdminElevation.Delete().Where(adminelevation.ExpiresAtLTE(time.Now())).Exec(ctx); err != nil {
		return err
	}
	return s.db.AdminPrincipal.Update().Where(adminprincipal.PendingExpiresAtLTE(time.Now())).ClearPendingSecret().ClearPendingExpiresAt().SetPendingSession("").Exec(ctx)
}

func (s *Service) OrdinaryTarget(ctx context.Context, id string) error {
	exists, err := persistence.Client(ctx, s.db).AdminPrincipal.Query().Where(adminprincipal.ID(id)).Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return ErrProtected
	}
	return nil
}
func (s *Service) AuditEvents(ctx context.Context, cursor string) (audit.AdminEvents, error) {
	return audit.AdminList(ctx, persistence.Client(ctx, s.db), cursor)
}
