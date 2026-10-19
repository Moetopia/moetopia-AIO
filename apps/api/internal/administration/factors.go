package administration

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/sessions"
)

type Enrollment struct {
	Secret    string    `json:"secret"`
	URI       string    `json:"otpauth_uri"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) BeginEnrollment(ctx context.Context, user identity.User, web sessions.Session, password string) (Enrollment, error) {
	if err := s.users.Reauthenticate(ctx, user.ID, password); err != nil {
		return Enrollment{}, err
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Moetopia administration", AccountName: user.Email, Period: 30, SecretSize: 20, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
	if err != nil {
		return Enrollment{}, err
	}
	ctx, unit, row, err := s.begin(ctx, user, web)
	if err != nil {
		return Enrollment{}, err
	}
	defer unit.Rollback()
	if len(row.Secret) > 0 {
		return Enrollment{}, ErrState
	}
	sealed, err := s.box.Seal([]byte(key.Secret()), "admin:pending:"+user.ID+":"+web.ID)
	if err != nil {
		return Enrollment{}, err
	}
	expires := time.Now().Add(5 * time.Minute)
	if err := unit.AdminPrincipal.UpdateOneID(user.ID).SetPendingSecret(sealed).SetPendingSession(web.ID).SetPendingExpiresAt(expires).Exec(ctx); err != nil {
		return Enrollment{}, err
	}
	if err := audit.RecordTarget(ctx, unit.Client(), user.ID, "admin.mfa.enrollment.started", user.ID, "success"); err != nil {
		return Enrollment{}, err
	}
	if err := unit.Commit(); err != nil {
		return Enrollment{}, err
	}
	return Enrollment{key.Secret(), key.URL(), expires}, nil
}
func totpCounter(secret, code string, last int64, now time.Time) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	current := now.Unix() / 30
	for counter := current + 1; counter >= current-1; counter-- {
		if counter <= last {
			continue
		}
		candidate, err := totp.GenerateCodeCustom(secret, time.Unix(counter*30, 0), totp.ValidateOpts{Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if err == nil && subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 {
			return counter, true
		}
	}
	return 0, false
}
func failedProof(ctx context.Context, unit *persistence.Unit, userID string) error {
	if err := audit.RecordTarget(ctx, unit.Client(), userID, "admin.mfa.verification", userID, "failure"); err != nil {
		return err
	}
	if err := unit.Commit(); err != nil {
		return err
	}
	return ErrProof
}
func (s *Service) ConfirmEnrollment(ctx context.Context, user identity.User, web sessions.Session, code string) (string, []string, error) {
	ctx, unit, row, err := s.begin(ctx, user, web)
	if err != nil {
		return "", nil, err
	}
	defer unit.Rollback()
	if len(row.Secret) > 0 || len(row.PendingSecret) == 0 || row.PendingSession != web.ID || row.PendingExpiresAt == nil || !time.Now().Before(*row.PendingExpiresAt) {
		return "", nil, ErrState
	}
	secret, err := s.box.Open(row.PendingSecret, "admin:pending:"+user.ID+":"+web.ID)
	if err != nil {
		return "", nil, err
	}
	counter, ok := totpCounter(string(secret), code, -1, time.Now())
	if !ok {
		return "", nil, failedProof(ctx, unit, user.ID)
	}
	sealed, err := s.box.Seal(secret, "admin:totp:"+user.ID)
	if err != nil {
		return "", nil, err
	}
	codes, hashes, err := recoveryCodes(user.ID)
	if err != nil {
		return "", nil, err
	}
	if err := unit.AdminPrincipal.UpdateOneID(user.ID).SetSecret(sealed).ClearPendingSecret().SetPendingSession("").ClearPendingExpiresAt().SetLastCounter(counter).AddFactorVersion(1).SetRecoveryHashes(hashes).Exec(ctx); err != nil {
		return "", nil, err
	}
	row.FactorVersion++
	if err := s.clearSessions(ctx, unit.Client(), user.ID); err != nil {
		return "", nil, err
	}
	handle, err := s.elevate(ctx, unit.Client(), row, user, web)
	if err != nil {
		return "", nil, err
	}
	if err := audit.RecordTarget(ctx, unit.Client(), user.ID, "admin.mfa.enrolled", user.ID, "success"); err != nil {
		return "", nil, err
	}
	if err := s.notify(ctx, unit.Client(), user, "mfa.enrolled"); err != nil {
		return "", nil, err
	}
	if err := unit.Commit(); err != nil {
		return "", nil, err
	}
	return handle, codes, nil
}
func (s *Service) Verify(ctx context.Context, user identity.User, web sessions.Session, code string) (string, error) {
	ctx, unit, row, err := s.begin(ctx, user, web)
	if err != nil {
		return "", err
	}
	defer unit.Rollback()
	if len(row.Secret) == 0 {
		return "", ErrFactor
	}
	secret, err := s.box.Open(row.Secret, "admin:totp:"+user.ID)
	if err != nil {
		return "", err
	}
	counter, ok := totpCounter(string(secret), code, row.LastCounter, time.Now())
	if !ok {
		return "", failedProof(ctx, unit, user.ID)
	}
	if err := unit.AdminPrincipal.UpdateOneID(user.ID).SetLastCounter(counter).Exec(ctx); err != nil {
		return "", err
	}
	handle, err := s.elevate(ctx, unit.Client(), row, user, web)
	if err != nil {
		return "", err
	}
	if err := audit.RecordTarget(ctx, unit.Client(), user.ID, "admin.mfa.verified", user.ID, "success"); err != nil {
		return "", err
	}
	return handle, unit.Commit()
}
func (s *Service) elevate(ctx context.Context, db *ent.Client, row *ent.AdminPrincipal, user identity.User, web sessions.Session) (string, error) {
	handle, err := s.Sessions.Rotate(ctx, web)
	if err != nil {
		return "", err
	}
	current, err := s.Sessions.Find(ctx, handle)
	if err != nil {
		return "", err
	}
	expires := time.Now().Add(15 * time.Minute)
	if web.ExpiresAt.Before(expires) {
		expires = web.ExpiresAt
	}
	if err := db.AdminElevation.Create().SetID(current.ID).SetUserID(user.ID).SetFactorVersion(row.FactorVersion).SetExpiresAt(expires).Exec(ctx); err != nil {
		return "", err
	}
	return handle, nil
}
