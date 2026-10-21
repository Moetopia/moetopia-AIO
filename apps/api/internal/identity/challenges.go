package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/password"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountchallenge"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
)

func (s *Service) challengeHash(id, purpose, userID, email, code string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = fmt.Fprintf(mac, "moetopia:account-challenge:v1:%s:%s:%s:%s:%s", id, purpose, userID, email, code)
	return mac.Sum(nil)
}
func (s *Service) BeginReset(ctx context.Context, inputEmail, locale string) (string, error) {
	email, err := NormalizeEmail(inputEmail)
	if err != nil {
		return "", err
	}
	row, err := s.store.db.AccountUser.Query().Where(accountuser.Email(email), accountuser.Enabled(true)).Select(userColumns...).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}
	if row == nil {
		row = &ent.AccountUser{}
	}
	return s.beginChallenge(ctx, "reset", row, email, "", locale, row.ID == "")
}
func (s *Service) BeginEmailChange(ctx context.Context, id, binding, current, target, locale string) (string, error) {
	email, err := NormalizeEmail(target)
	if err != nil {
		return "", err
	}
	row, err := s.credentialsByID(ctx, id, current)
	if err != nil {
		return "", err
	}
	if row.Email == email || binding == "" {
		return "", ErrInvalid
	}
	exists, err := s.store.Exists(ctx, email)
	if err != nil {
		return "", err
	}
	return s.beginChallenge(ctx, "email", row, email, binding, locale, exists)
}
func (s *Service) beginChallenge(ctx context.Context, purpose string, row *ent.AccountUser, email, binding, locale string, suppressed bool) (string, error) {
	id := uuid.NewString()
	number, err := rand.Int(rand.Reader, big.NewInt(100000000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%08d", number.Int64())
	expires := time.Now().Add(10 * time.Minute)
	tx, err := s.store.db.Tx(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := tx.AccountChallenge.Create().SetID(id).SetPurpose(accountchallenge.Purpose(purpose)).SetUserID(row.ID).SetEmail(email).SetBinding(binding).SetSecurityVersion(row.SecurityVersion).SetCodeHash(s.challengeHash(id, purpose, row.ID, email, code)).SetExpiresAt(expires).Exec(ctx); err != nil {
		return "", err
	}
	message := notifications.Message{UserID: row.ID, To: email, Subject: "Moetopia account verification", Text: "Your Moetopia " + purpose + " code is: " + code + "\nChallenge: " + id + "\nExpires in 10 minutes. If you did not request this, ignore this message.", Suppressed: suppressed}
	if locale != "en" {
		message.Text = "你的 Moetopia " + map[string]string{"reset": "密码恢复", "email": "邮箱变更", "deletion-cancel": "取消注销"}[purpose] + "验证码是：" + code + "\n验证事务：" + id + "\n10 分钟内有效。如非本人操作，请忽略。"
	}
	if err := s.store.outbox.Enqueue(ctx, tx.Client(), message, expires); err != nil {
		return "", err
	}
	return id, tx.Commit()
}
func (s *Service) ResetPassword(ctx context.Context, id, code, next string) error {
	if !password.Valid(next) || len(code) != 8 {
		return ErrInvalid
	}
	row, err := s.store.db.AccountChallenge.Get(ctx, id)
	if ent.IsNotFound(err) {
		return ErrInvalid
	}
	if err != nil {
		return err
	}
	hash := ""
	if row.Purpose == accountchallenge.PurposeReset && !row.Consumed && row.Attempts < 5 && time.Now().Before(row.ExpiresAt) && subtle.ConstantTimeCompare(row.CodeHash, s.challengeHash(id, "reset", row.UserID, row.Email, code)) == 1 {
		hash, err = password.Hash(ctx, next)
		if err != nil {
			return err
		}
	}
	return s.finishChallenge(ctx, id, code, "reset", "", "", "", hash)
}
func (s *Service) ConfirmEmailChange(ctx context.Context, userID, binding, id, code, current string) error {
	row, err := s.credentialsByID(ctx, userID, current)
	if err != nil {
		return err
	}
	return s.finishChallenge(ctx, id, code, "email", userID, binding, row.PasswordHash, "")
}
func (s *Service) finishChallenge(ctx context.Context, id, code, purpose, userID, binding, currentHash, newHash string) error {
	meta, err := s.store.db.AccountChallenge.Get(ctx, id)
	if ent.IsNotFound(err) {
		return ErrInvalid
	}
	if err != nil {
		return err
	}
	if string(meta.Purpose) != purpose || meta.UserID == "" || userID != "" && meta.UserID != userID || meta.Binding != binding {
		return ErrInvalid
	}
	tx, err := s.store.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock the user before challenges so all credential updates share one lock order.
	user, err := tx.AccountUser.Query().Where(accountuser.ID(meta.UserID), accountuser.Enabled(true)).ForUpdate().Only(ctx)
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
	if row.Consumed || row.Attempts >= 5 || !time.Now().Before(row.ExpiresAt) || row.SecurityVersion != user.SecurityVersion || purpose == "reset" && row.Email != user.Email {
		return ErrInvalid
	}
	if len(code) != 8 || subtle.ConstantTimeCompare(row.CodeHash, s.challengeHash(id, purpose, user.ID, row.Email, code)) != 1 {
		if err := tx.AccountChallenge.UpdateOneID(id).AddAttempts(1).Exec(ctx); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrInvalid
	}
	update := tx.AccountUser.UpdateOneID(user.ID).AddSecurityVersion(1).AddRevision(1)
	action := "password.reset"
	if purpose == "reset" {
		if newHash == "" {
			return ErrInvalid
		}
		update.SetPasswordHash(newHash).SetHasPassword(true)
	} else {
		if user.PasswordHash != currentHash {
			return ErrCredentials
		}
		update.SetEmail(row.Email).SetEmailVerifiedAt(time.Now())
		action = "email.changed"
	}
	if err := update.Exec(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return ErrInvalid
		}
		return err
	}
	if err := s.securityChanged(ctx, tx.Client(), user, action); err != nil {
		return err
	}
	if purpose == "email" {
		if err := s.store.outbox.Enqueue(ctx, tx.Client(), notifications.Message{UserID: user.ID, To: row.Email, Subject: "Moetopia email updated", Text: "Your Moetopia email has been updated. Sign in using this address.\n你的 Moetopia 邮箱已更新，请使用此邮箱重新登录。"}, time.Now().Add(24*time.Hour)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
