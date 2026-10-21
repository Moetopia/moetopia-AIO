package identity

import (
	"context"
	"crypto/subtle"
	"time"

	"moetopia.local/aio/api/internal/password"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountchallenge"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
)

func (s *Service) BeginDeletionCancel(ctx context.Context, inputEmail, locale string) (string, error) {
	email, err := NormalizeEmail(inputEmail)
	if err != nil {
		return "", err
	}
	row, err := s.store.db.AccountUser.Query().Where(accountuser.Email(email), accountuser.Enabled(true), accountuser.DeletionDueAtGT(time.Now())).Select(userColumns...).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}
	if row == nil {
		row = &ent.AccountUser{}
	}
	return s.beginChallenge(ctx, "deletion-cancel", row, email, "", locale, row.ID == "")
}

func (s *Service) CancelDeletion(ctx context.Context, id, code, current string) error {
	meta, err := s.store.db.AccountChallenge.Get(ctx, id)
	if ent.IsNotFound(err) {
		return ErrInvalid
	}
	if err != nil {
		return err
	}
	if meta.Purpose != accountchallenge.PurposeDeletionCancel || meta.UserID == "" || len(code) != 8 {
		return ErrInvalid
	}
	before, err := s.store.db.AccountUser.Query().Where(accountuser.ID(meta.UserID), accountuser.Enabled(true), accountuser.DeletionDueAtGT(time.Now())).Select(userColumns...).Only(ctx)
	if ent.IsNotFound(err) {
		return ErrInvalid
	}
	if err != nil {
		return err
	}
	if !password.Verify(ctx, current, before.PasswordHash) {
		return ErrCredentials
	}
	tx, err := s.store.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user, err := tx.AccountUser.Query().Where(accountuser.ID(meta.UserID), accountuser.Enabled(true), accountuser.DeletionDueAtGT(time.Now())).ForUpdate().Select(append(append([]string{}, userColumns...), accountuser.FieldDeletionDueAt)...).Only(ctx)
	if ent.IsNotFound(err) {
		return ErrInvalid
	}
	if err != nil {
		return err
	}
	row, err := tx.AccountChallenge.Query().Where(accountchallenge.ID(id)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	if user.DeletionDueAt == nil || !time.Now().Before(*user.DeletionDueAt) || row.Consumed || row.Attempts >= 5 || !time.Now().Before(row.ExpiresAt) || row.SecurityVersion != user.SecurityVersion || row.Email != user.Email || user.PasswordHash != before.PasswordHash {
		return ErrInvalid
	}
	if subtle.ConstantTimeCompare(row.CodeHash, s.challengeHash(id, "deletion-cancel", user.ID, user.Email, code)) != 1 {
		if err := tx.AccountChallenge.UpdateOneID(id).AddAttempts(1).Exec(ctx); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrInvalid
	}
	if err := tx.AccountUser.UpdateOneID(user.ID).ClearDeletionRequestedAt().ClearDeletionDueAt().AddRevision(1).AddSecurityVersion(1).Exec(ctx); err != nil {
		return err
	}
	if err := s.securityChanged(ctx, tx.Client(), user, "account.deletion.cancelled"); err != nil {
		return err
	}
	return tx.Commit()
}
