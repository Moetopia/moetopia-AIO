package identity

import (
	"context"
	"encoding/hex"
	"time"

	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/invitation"
	"moetopia.local/aio/api/internal/persistence/ent/registrationpolicy"
)

type RegistrationPolicy struct {
	Mode     string `json:"mode"`
	Revision int    `json:"revision"`
}

func (s *Service) RegistrationPolicy(ctx context.Context) (RegistrationPolicy, error) {
	row, err := persistence.Client(ctx, s.store.db).RegistrationPolicy.Get(ctx, "registration")
	if err != nil {
		return RegistrationPolicy{}, err
	}
	return RegistrationPolicy{string(row.Mode), row.Revision}, nil
}
func (s *Service) SetRegistrationPolicy(ctx context.Context, actor, mode string, revision int) (RegistrationPolicy, error) {
	if actor == "" || mode != "open" && mode != "invite" || revision < 0 {
		return RegistrationPolicy{}, ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return RegistrationPolicy{}, err
	}
	defer unit.Rollback()
	db := unit.Client()
	ctx = persistence.WithClient(ctx, db)
	count, err := db.RegistrationPolicy.Update().Where(registrationpolicy.ID("registration"), registrationpolicy.Revision(revision)).SetMode(registrationpolicy.Mode(mode)).AddRevision(1).Save(ctx)
	if err != nil {
		return RegistrationPolicy{}, err
	}
	if count != 1 {
		return RegistrationPolicy{}, ErrConflict
	}
	if err := audit.RecordTarget(ctx, db, actor, "registration-policy.updated", "registration", "success"); err != nil {
		return RegistrationPolicy{}, err
	}
	return RegistrationPolicy{mode, revision + 1}, unit.Commit()
}

type ManagedInvitation struct {
	ID        string    `json:"id"`
	Consumed  bool      `json:"consumed"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
type ManagedInvitations struct {
	Items      []ManagedInvitation `json:"items"`
	NextCursor string              `json:"next_cursor"`
}
type IssuedInvitation struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) AdminInvitations(ctx context.Context, cursor string) (ManagedInvitations, error) {
	result := ManagedInvitations{Items: []ManagedInvitation{}}
	query := persistence.Client(ctx, s.store.db).Invitation.Query()
	if cursor != "" {
		if raw, err := hex.DecodeString(cursor); err != nil || len(raw) != 32 {
			return result, ErrInvalid
		}
		query.Where(invitation.IDGT(cursor))
	}
	rows, err := query.Order(ent.Asc(invitation.FieldID)).Limit(51).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 50 {
		rows = rows[:50]
		result.NextCursor = rows[49].ID
	}
	for _, row := range rows {
		result.Items = append(result.Items, ManagedInvitation{row.ID, row.Consumed, row.Revoked, row.CreatedAt, row.ExpiresAt})
	}
	return result, nil
}
func (s *Service) AdminIssueInvitation(ctx context.Context, actor string, hours int) (IssuedInvitation, error) {
	if actor == "" || hours < 1 || hours > 720 {
		return IssuedInvitation{}, ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return IssuedInvitation{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	expires := time.Now().Add(time.Duration(hours) * time.Hour)
	code, err := s.IssueInvitation(ctx, expires)
	if err != nil {
		return IssuedInvitation{}, err
	}
	id := invitationHash(code)
	if err := audit.RecordTarget(ctx, persistence.Client(ctx, s.store.db), actor, "invitation.created", id, "success"); err != nil {
		return IssuedInvitation{}, err
	}
	return IssuedInvitation{id, code, expires}, unit.Commit()
}
func (s *Service) AdminRevokeInvitation(ctx context.Context, actor, id string) error {
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	db := unit.Client()
	ctx = persistence.WithClient(ctx, db)
	row, err := db.Invitation.Query().Where(invitation.ID(id)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ErrMissing
	}
	if err != nil {
		return err
	}
	if row.Consumed {
		return ErrConflict
	}
	if row.Revoked {
		return unit.Commit()
	}
	if err := db.Invitation.UpdateOneID(id).SetRevoked(true).Exec(ctx); err != nil {
		return err
	}
	if err := audit.RecordTarget(ctx, db, actor, "invitation.revoked", id, "success"); err != nil {
		return err
	}
	return unit.Commit()
}
