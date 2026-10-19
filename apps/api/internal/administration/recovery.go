package administration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/adminelevation"
	"moetopia.local/aio/api/internal/sessions"
)

func recoveryHash(userID, code string) string {
	hash := sha256.Sum256([]byte("moetopia:admin:recovery:v1:" + userID + ":" + strings.ToLower(strings.TrimSpace(code))))
	return hex.EncodeToString(hash[:])
}
func recoveryCodes(userID string) ([]string, []string, error) {
	codes, hashes := make([]string, 10), make([]string, 10)
	for i := range codes {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		codes[i] = hex.EncodeToString(raw)
		hashes[i] = recoveryHash(userID, codes[i])
	}
	return codes, hashes, nil
}
func (s *Service) clearSessions(ctx context.Context, db *ent.Client, userID string) error {
	if err := s.Sessions.RevokeApplicationUser(ctx, userID); err != nil {
		return err
	}
	_, err := db.AdminElevation.Delete().Where(adminelevation.UserID(userID)).Exec(ctx)
	return err
}
func (s *Service) Recover(ctx context.Context, user identity.User, web sessions.Session, password, code string) (string, error) {
	if err := s.users.Reauthenticate(ctx, user.ID, password); err != nil {
		return "", err
	}
	ctx, unit, row, err := s.begin(ctx, user, web)
	if err != nil {
		return "", err
	}
	defer unit.Rollback()
	valid := false
	for _, hash := range row.RecoveryHashes {
		if subtle.ConstantTimeCompare([]byte(hash), []byte(recoveryHash(user.ID, code))) == 1 {
			valid = true
		}
	}
	if !valid || len(row.Secret) == 0 {
		return "", failedProof(ctx, unit, user.ID)
	}
	if err := unit.AdminPrincipal.UpdateOneID(user.ID).ClearSecret().ClearPendingSecret().ClearPendingExpiresAt().SetPendingSession("").SetLastCounter(-1).SetRecoveryHashes([]string{}).AddFactorVersion(1).Exec(ctx); err != nil {
		return "", err
	}
	if err := s.clearSessions(ctx, unit.Client(), user.ID); err != nil {
		return "", err
	}
	handle, err := s.Sessions.Rotate(ctx, web)
	if err != nil {
		return "", err
	}
	if err := audit.RecordTarget(ctx, unit.Client(), user.ID, "admin.mfa.recovered", user.ID, "success"); err != nil {
		return "", err
	}
	if err := s.notify(ctx, unit.Client(), user, "mfa.recovered"); err != nil {
		return "", err
	}
	return handle, unit.Commit()
}
func (s *Service) RegenerateRecovery(ctx context.Context, user identity.User, web sessions.Session, password string) ([]string, error) {
	if err := s.users.Reauthenticate(ctx, user.ID, password); err != nil {
		return nil, err
	}
	var result []string
	err := s.Run(ctx, user, web, func(ctx context.Context) error {
		db := persistence.Client(ctx, s.db)
		codes, hashes, err := recoveryCodes(user.ID)
		if err != nil {
			return err
		}
		if err := db.AdminPrincipal.UpdateOneID(user.ID).SetRecoveryHashes(hashes).Exec(ctx); err != nil {
			return err
		}
		if err := audit.RecordTarget(ctx, db, user.ID, "admin.recovery-codes.regenerated", user.ID, "success"); err != nil {
			return err
		}
		if err := s.notify(ctx, db, user, "recovery-codes.regenerated"); err != nil {
			return err
		}
		result = codes
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
