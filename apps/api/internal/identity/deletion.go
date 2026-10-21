package identity

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountchallenge"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"moetopia.local/aio/api/internal/persistence/ent/emailverification"
)

const deletionCooldown = 7 * 24 * time.Hour

type Deletion struct {
	RequestedAt time.Time `json:"requested_at"`
	DueAt       time.Time `json:"due_at"`
}

func (s *Service) ScheduleDeletion(ctx context.Context, user User, revision int) (Deletion, error) {
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return Deletion{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	db := unit.Client()
	row, err := db.AccountUser.Query().Where(accountuser.ID(user.ID), accountuser.Enabled(true), accountuser.DeletionDueAtIsNil()).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return Deletion{}, ErrMissing
	}
	if err != nil {
		return Deletion{}, err
	}
	if row.Revision != revision || row.SecurityVersion != user.SecurityVersion {
		return Deletion{}, ErrConflict
	}
	result := Deletion{RequestedAt: time.Now().UTC()}
	result.DueAt = result.RequestedAt.Add(deletionCooldown)
	if err := db.AccountUser.UpdateOneID(user.ID).SetDeletionRequestedAt(result.RequestedAt).SetDeletionDueAt(result.DueAt).AddRevision(1).AddSecurityVersion(1).Exec(ctx); err != nil {
		return Deletion{}, err
	}
	if err := s.revokeSecurity(ctx, db, row, "account.deletion.requested"); err != nil {
		return Deletion{}, err
	}
	message := notifications.Message{UserID: row.ID, To: row.Email, Subject: "Moetopia account deletion requested / 账号注销申请", Text: fmt.Sprintf("Your account is scheduled for deletion at %s (UTC). Sign-in is blocked and existing sessions have ended. Cancel before this deadline using your password and an email verification code. If you forgot your password, reset it first; resetting does not cancel deletion.\n你的账号将在上述 UTC 时间到期清理，当前已禁止登录并结束会话。请在到期前使用当前密码和邮箱验证码取消注销；忘记密码可先重置，重置密码不会自动取消注销。", result.DueAt.Format(time.RFC3339))}
	if err := s.store.outbox.Enqueue(ctx, db, message, time.Now().Add(24*time.Hour)); err != nil {
		return Deletion{}, err
	}
	return result, unit.Commit()
}

func (s *Service) DueDeletions(ctx context.Context) ([]string, error) {
	return s.store.db.AccountUser.Query().Where(accountuser.DeletionDueAtLTE(time.Now())).Order(ent.Asc(accountuser.FieldDeletionDueAt), ent.Asc(accountuser.FieldID)).Limit(10).IDs(ctx)
}

func (s *Service) LockDueDeletion(ctx context.Context, id string) (bool, error) {
	_, err := persistence.Client(ctx, s.store.db).AccountUser.Query().Where(accountuser.ID(id), accountuser.DeletionDueAtLTE(time.Now())).ForUpdate(sql.WithLockAction(sql.SkipLocked)).Only(ctx)
	if ent.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) PurgeDeletion(ctx context.Context, id string) error {
	db := persistence.Client(ctx, s.store.db)
	row, err := db.AccountUser.Query().Where(accountuser.ID(id), accountuser.DeletionDueAtLTE(time.Now())).ForUpdate().Select(accountuser.FieldID, accountuser.FieldEmail).Only(ctx)
	if ent.IsNotFound(err) {
		return ErrMissing
	}
	if err != nil {
		return err
	}
	if err := s.store.revoker.RevokeUser(ctx, id); err != nil {
		return err
	}
	if _, err := db.AccountChallenge.Delete().Where(accountchallenge.UserID(id)).Exec(ctx); err != nil {
		return err
	}
	if _, err := db.EmailVerification.Delete().Where(emailverification.Email(row.Email)).Exec(ctx); err != nil {
		return err
	}
	if err := s.store.outbox.PurgeUser(ctx, db, id); err != nil {
		return err
	}
	if err := audit.Record(ctx, db, id, "account.deletion.completed"); err != nil {
		return err
	}
	return db.AccountUser.DeleteOneID(id).Exec(ctx)
}
