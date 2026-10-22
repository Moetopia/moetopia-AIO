package administration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/adminprincipal"
	"moetopia.local/aio/api/internal/sessions"
)

type EmergencyTicket struct {
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Revision  int       `json:"revision"`
}

func emergencyHash(userID, token string) string {
	hash := sha256.Sum256([]byte("moetopia:admin:emergency:v1:" + userID + ":" + token))
	return hex.EncodeToString(hash[:])
}

// EmergencyReset is only called by the explicit host-operator CLI, never an HTTP adapter.
func (s *Service) EmergencyReset(ctx context.Context, target string, revision int, reason string) (EmergencyTicket, error) {
	id, err := uuid.Parse(target)
	if err != nil || id == uuid.Nil || id.String() != target || revision < 0 || !validReason(reason) {
		return EmergencyTicket{}, ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return EmergencyTicket{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if err := lockAdministration(ctx, unit.Client()); err != nil {
		return EmergencyTicket{}, err
	}
	user, err := s.users.User(ctx, target)
	if err != nil {
		return EmergencyTicket{}, err
	}
	if err := s.users.VerifyVersion(ctx, target, user.SecurityVersion); err != nil {
		return EmergencyTicket{}, err
	}
	row, err := unit.AdminPrincipal.Query().Where(adminprincipal.ID(target), adminprincipal.Enabled(true)).ForUpdate().Only(ctx)
	if err != nil {
		return EmergencyTicket{}, err
	}
	if row.FactorVersion != revision {
		return EmergencyTicket{}, ErrState
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return EmergencyTicket{}, err
	}
	ticket := EmergencyTicket{target, hex.EncodeToString(raw), time.Now().Add(15 * time.Minute), revision + 1}
	if err := unit.AdminPrincipal.UpdateOneID(target).ClearSecret().ClearPendingSecret().ClearPendingExpiresAt().SetPendingSession("").SetRecoveryHashes([]string{}).SetLastCounter(-1).AddFactorVersion(1).SetEnrollmentBlocked(true).SetEnrollmentSession("").SetEmergencyHash(emergencyHash(target, ticket.Token)).SetEmergencyExpiresAt(ticket.ExpiresAt).Exec(ctx); err != nil {
		return EmergencyTicket{}, err
	}
	if err := s.clearSessions(ctx, unit.Client(), target); err != nil {
		return EmergencyTicket{}, err
	}
	if err := audit.RecordReason(ctx, unit.Client(), "system:administrator-emergency", "admin.mfa.emergency_reset", target, strings.TrimSpace(reason)); err != nil {
		return EmergencyTicket{}, err
	}
	if err := s.notify(ctx, unit.Client(), user, "mfa.emergency_reset"); err != nil {
		return EmergencyTicket{}, err
	}
	return ticket, unit.Commit()
}

func (s *Service) AcceptEmergency(ctx context.Context, user identity.User, web sessions.Session, password, token string) (string, error) {
	if err := s.users.Reauthenticate(ctx, user.ID, password); err != nil {
		return "", err
	}
	ctx, unit, row, err := s.begin(ctx, user, web)
	if err != nil {
		return "", err
	}
	defer unit.Rollback()
	if !row.EnrollmentBlocked || row.EmergencyExpiresAt == nil || !time.Now().Before(*row.EmergencyExpiresAt) || len(token) != 64 || len(row.EmergencyHash) != 64 || subtle.ConstantTimeCompare([]byte(row.EmergencyHash), []byte(emergencyHash(user.ID, token))) != 1 {
		return "", failedProof(ctx, unit, user.ID)
	}
	if err := s.clearSessions(ctx, unit.Client(), user.ID); err != nil {
		return "", err
	}
	handle, err := s.Sessions.Rotate(ctx, web)
	if err != nil {
		return "", err
	}
	current, err := s.Sessions.Find(ctx, handle)
	if err != nil {
		return "", err
	}
	if err := unit.AdminPrincipal.UpdateOneID(user.ID).SetEnrollmentBlocked(false).SetEnrollmentSession(current.ID).SetEmergencyHash("").ClearEmergencyExpiresAt().AddFactorVersion(1).Exec(ctx); err != nil {
		return "", err
	}
	if err := audit.RecordTarget(ctx, unit.Client(), user.ID, "admin.mfa.emergency_accepted", user.ID, "success"); err != nil {
		return "", err
	}
	if err := s.notify(ctx, unit.Client(), user, "mfa.emergency_accepted"); err != nil {
		return "", err
	}
	return handle, unit.Commit()
}
