package identity

import (
	"context"
	"errors"
	"time"

	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/password"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountchallenge"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
)

var userColumns = []string{accountuser.FieldID, accountuser.FieldEmail, accountuser.FieldDisplayName, accountuser.FieldBio, accountuser.FieldAvatarDigest, accountuser.FieldRevision, accountuser.FieldSecurityVersion, accountuser.FieldPasswordHash, accountuser.FieldEnabled, accountuser.FieldEmailVerifiedAt, accountuser.FieldCreatedAt}

func (s *Service) VerifyVersion(ctx context.Context, id string, version int) error {
	row, err := persistence.Client(ctx, s.store.db).AccountUser.Query().Where(accountuser.ID(id), accountuser.Enabled(true)).Select(accountuser.FieldSecurityVersion).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ErrMissing
	}
	if err != nil {
		return err
	}
	if row.SecurityVersion != version {
		return ErrMissing
	}
	return nil
}

func (s *Service) credentialsByID(ctx context.Context, id, current string) (*ent.AccountUser, error) {
	row, err := s.store.db.AccountUser.Query().Where(accountuser.ID(id), accountuser.Enabled(true)).Select(userColumns...).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrMissing
	}
	if err != nil {
		return nil, err
	}
	if !password.Verify(ctx, current, row.PasswordHash) {
		return nil, ErrCredentials
	}
	return row, nil
}
func (s *Service) Reauthenticate(ctx context.Context, id, current string) error {
	_, err := s.credentialsByID(ctx, id, current)
	return err
}
func (s *Service) ChangePassword(ctx context.Context, id, current, next string) error {
	if !password.Valid(next) || next == current {
		return ErrInvalid
	}
	before, err := s.credentialsByID(ctx, id, current)
	if err != nil {
		return err
	}
	hash, err := password.Hash(ctx, next)
	if err != nil {
		return err
	}
	tx, err := s.store.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	row, err := tx.AccountUser.Query().Where(accountuser.ID(id), accountuser.Enabled(true)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	if row.SecurityVersion != before.SecurityVersion || row.PasswordHash != before.PasswordHash {
		return ErrCredentials
	}
	if err := tx.AccountUser.UpdateOneID(id).SetPasswordHash(hash).AddSecurityVersion(1).AddRevision(1).Exec(ctx); err != nil {
		return err
	}
	if err := s.securityChanged(ctx, tx.Client(), row, "password.changed"); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) securityChanged(ctx context.Context, db *ent.Client, row *ent.AccountUser, action string) error {
	if s.store.revoker == nil {
		return errors.New("account session invalidation is not configured")
	}
	if err := s.store.revoker.RevokeUser(persistence.WithClient(ctx, db), row.ID); err != nil {
		return err
	}
	if _, err := db.AccountChallenge.Update().Where(accountchallenge.UserID(row.ID)).SetConsumed(true).Save(ctx); err != nil {
		return err
	}
	if err := audit.Record(ctx, db, row.ID, action); err != nil {
		return err
	}
	message := notifications.Message{To: row.Email, Subject: "Moetopia account security update", Text: "Your Moetopia account security settings changed (" + action + "). Sign in again. If this was not you, reset your password immediately.\n你的 Moetopia 账号安全信息已变更，请重新登录。如非本人操作，请立即找回密码。"}
	return s.store.outbox.Enqueue(ctx, db, message, time.Now().Add(24*time.Hour))
}
