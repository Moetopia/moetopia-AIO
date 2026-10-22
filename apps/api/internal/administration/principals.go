package administration

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/admincontrol"
	"moetopia.local/aio/api/internal/persistence/ent/adminprincipal"
	"moetopia.local/aio/api/internal/sessions"
)

type Principal struct {
	ID                string    `json:"id"`
	Enabled           bool      `json:"enabled"`
	Enrolled          bool      `json:"enrolled"`
	EmergencyRequired bool      `json:"emergency_required"`
	Revision          int       `json:"revision"`
	CreatedAt         time.Time `json:"created_at"`
}

type Principals struct {
	Items     []Principal `json:"items" nullable:"false"`
	NextAfter string      `json:"next_after"`
}

func principalView(row *ent.AdminPrincipal) Principal {
	return Principal{row.ID, row.Enabled, len(row.Secret) > 0, row.EnrollmentBlocked, row.FactorVersion, row.CreatedAt}
}
func (s *Service) PrincipalStatus(ctx context.Context, id string) (Principal, error) {
	if parsed, err := uuid.Parse(id); err != nil || parsed.String() != id {
		return Principal{}, ErrInvalid
	}
	row, err := persistence.Client(ctx, s.db).AdminPrincipal.Get(ctx, id)
	if err != nil {
		return Principal{}, err
	}
	return principalView(row), nil
}

func lockAdministration(ctx context.Context, db *ent.Client) error {
	// Acquire before identity/session locks for all administrator control transactions.
	_, err := db.AdminControl.Query().Where(admincontrol.ID("administrator-security")).ForUpdate().Only(ctx)
	return err
}

func (s *Service) Principals(ctx context.Context, after string) (Principals, error) {
	result := Principals{Items: []Principal{}}
	query := persistence.Client(ctx, s.db).AdminPrincipal.Query()
	if after != "" {
		id, err := uuid.Parse(after)
		if err != nil || id.String() != after {
			return result, ErrInvalid
		}
		query.Where(adminprincipal.IDGT(after))
	}
	rows, err := query.Order(ent.Asc(adminprincipal.FieldID)).Limit(51).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 50 {
		rows = rows[:50]
		result.NextAfter = rows[49].ID
	}
	for _, row := range rows {
		result.Items = append(result.Items, principalView(row))
	}
	return result, nil
}

func validReason(reason string) bool {
	return strings.TrimSpace(reason) != "" && utf8.ValidString(reason) && utf8.RuneCountInString(reason) <= 500 && !strings.ContainsAny(reason, "\x00\r")
}

func (s *Service) ChangePrincipal(ctx context.Context, actor identity.User, web sessions.Session, target string, revision int, enabled bool, password, reason string) (Principal, error) {
	id, err := uuid.Parse(target)
	if err != nil || id == uuid.Nil || id.String() != target || revision < 0 || !validReason(reason) {
		return Principal{}, ErrInvalid
	}
	if err := s.users.Reauthenticate(ctx, actor.ID, password); err != nil {
		return Principal{}, err
	}
	var recipient identity.User
	var subjects []identity.User
	if enabled {
		recipient, err = s.users.User(ctx, target)
		if err != nil {
			return Principal{}, err
		}
		subjects = append(subjects, recipient)
	}
	ctx, unit, _, err := s.beginWithUsers(ctx, actor, web, subjects...)
	if err != nil {
		return Principal{}, err
	}
	defer unit.Rollback()
	state, err := s.State(ctx, actor.ID, web.ID)
	if err != nil {
		return Principal{}, err
	}
	if !state.Verified {
		return Principal{}, ErrFactor
	}
	email, err := s.users.VerifiedEmail(ctx, target)
	if err != nil {
		return Principal{}, err
	}
	recipient = identity.User{ID: target, Email: email}
	row, err := unit.AdminPrincipal.Query().Where(adminprincipal.ID(target)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		if !enabled || revision != 0 {
			return Principal{}, ErrState
		}
		row, err = unit.AdminPrincipal.Create().SetID(target).Save(ctx)
	} else if err == nil {
		if row.FactorVersion != revision || row.Enabled == enabled {
			return Principal{}, ErrState
		}
		if !enabled {
			others, err := unit.AdminPrincipal.Query().Where(adminprincipal.Enabled(true), adminprincipal.IDNEQ(target)).All(ctx)
			if err != nil {
				return Principal{}, err
			}
			ready := false
			for _, other := range others {
				if len(other.Secret) == 0 || other.EnrollmentBlocked {
					continue
				}
				if _, err := s.users.User(ctx, other.ID); err == nil {
					ready = true
					break
				} else if !errors.Is(err, identity.ErrMissing) {
					return Principal{}, err
				}
			}
			if !ready {
				return Principal{}, ErrLastAdministrator
			}
		}
		row, err = unit.AdminPrincipal.UpdateOneID(target).SetEnabled(enabled).ClearSecret().ClearPendingSecret().ClearPendingExpiresAt().SetPendingSession("").SetRecoveryHashes([]string{}).SetLastCounter(-1).AddFactorVersion(1).SetEnrollmentBlocked(false).SetEnrollmentSession("").SetEmergencyHash("").ClearEmergencyExpiresAt().Save(ctx)
	}
	if err != nil {
		return Principal{}, err
	}
	if err := s.clearSessions(ctx, unit.Client(), target); err != nil {
		return Principal{}, err
	}
	action := "admin.role.revoked"
	if enabled {
		action = "admin.role.granted"
	}
	if err := audit.RecordReason(ctx, unit.Client(), actor.ID, action, target, strings.TrimSpace(reason)); err != nil {
		return Principal{}, err
	}
	if err := s.notify(ctx, unit.Client(), recipient, action); err != nil {
		return Principal{}, err
	}
	return principalView(row), unit.Commit()
}
