package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountchallenge"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"moetopia.local/aio/api/internal/persistence/ent/emailverification"
	"moetopia.local/aio/api/internal/persistence/ent/invitation"
)

type Store struct {
	db      *ent.Client
	outbox  *notifications.Outbox
	revoker SessionRevoker
}
type Verification struct {
	Email     string
	Hash      []byte
	ExpiresAt time.Time
	Attempts  int
	Consumed  bool
}

type SessionRevoker interface {
	RevokeUser(context.Context, string) error
}

func NewStore(db *ent.Client, outbox *notifications.Outbox) *Store {
	return &Store{db: db, outbox: outbox}
}
func (s *Store) WithSessions(revoker SessionRevoker) *Store { s.revoker = revoker; return s }
func (s *Store) Cleanup(ctx context.Context) error {
	if _, err := s.db.AccountChallenge.Delete().Where(accountchallenge.ExpiresAtLTE(time.Now())).Exec(ctx); err != nil {
		return err
	}
	if err := audit.Cleanup(ctx, s.db); err != nil {
		return err
	}
	_, err := s.db.EmailVerification.Delete().Where(emailverification.ExpiresAtLTE(time.Now())).Exec(ctx)
	return err
}
func public(user *ent.AccountUser) User {
	avatar := ""
	if user.AvatarDigest != "" {
		avatar = "/api/v1/avatars/" + user.ID + "?v=" + fmt.Sprint(user.Revision)
	}
	return User{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Bio: user.Bio, AvatarURL: avatar, Revision: user.Revision, SecurityVersion: user.SecurityVersion}
}
func (s *Store) Exists(ctx context.Context, email string) (bool, error) {
	return s.db.AccountUser.Query().Where(accountuser.EmailEQ(email)).Exist(ctx)
}
func (s *Store) Credentials(ctx context.Context, email string) (User, string, error) {
	user, err := s.db.AccountUser.Query().Where(accountuser.EmailEQ(email), accountuser.EnabledEQ(true)).Select(userColumns...).Only(ctx)
	if ent.IsNotFound(err) {
		return User{}, "", ErrMissing
	}
	if err != nil {
		return User{}, "", err
	}
	return public(user), user.PasswordHash, nil
}
func (s *Store) User(ctx context.Context, id string) (User, error) {
	user, err := persistence.Client(ctx, s.db).AccountUser.Query().Where(accountuser.IDEQ(id), accountuser.EnabledEQ(true)).Select(userColumns...).Only(ctx)
	if ent.IsNotFound(err) {
		return User{}, ErrMissing
	}
	if err != nil {
		return User{}, err
	}
	return public(user), nil
}
func (s *Store) Mode(ctx context.Context) (string, error) {
	policy, err := s.db.RegistrationPolicy.Get(ctx, "registration")
	if err != nil {
		return "", err
	}
	return string(policy.Mode), nil
}
func (s *Store) Begin(ctx context.Context, id, email string, hash []byte, code, locale string, expires time.Time) error {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.EmailVerification.Create().SetID(id).SetEmail(email).SetCodeHash(hash).SetExpiresAt(expires).Exec(ctx); err != nil {
		return err
	}
	message := notifications.Message{To: email, Subject: "Moetopia registration code", Text: "Your Moetopia registration code is: " + code + "\nIt expires in 10 minutes. If you did not request this, ignore this message."}
	if locale != "en" {
		message.Text = "你的 Moetopia 注册验证码是：" + code + "\n验证码 10 分钟内有效。如非本人操作，请忽略。"
	}
	if err := s.outbox.Enqueue(ctx, tx.Client(), message, expires); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Verification(ctx context.Context, id string) (Verification, error) {
	v, err := s.db.EmailVerification.Get(ctx, id)
	if ent.IsNotFound(err) {
		return Verification{}, ErrInvalid
	}
	if err != nil {
		return Verification{}, err
	}
	return Verification{v.Email, v.CodeHash, v.ExpiresAt, v.Attempts, v.Consumed}, nil
}
func (s *Store) FailedAttempt(ctx context.Context, id string) error {
	_, err := s.db.EmailVerification.Update().Where(emailverification.IDEQ(id), emailverification.AttemptsLT(5)).AddAttempts(1).Save(ctx)
	return err
}
func (s *Store) Register(ctx context.Context, id, email, name, hash, invite string) error {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now()
	count, err := tx.EmailVerification.Update().Where(emailverification.IDEQ(id), emailverification.ConsumedEQ(false), emailverification.AttemptsLT(5), emailverification.ExpiresAtGT(now)).SetConsumed(true).Save(ctx)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrInvalid
	}
	policy, err := tx.RegistrationPolicy.Get(ctx, "registration")
	if err != nil {
		return err
	}
	if policy.Mode == "invite" {
		count, err = tx.Invitation.Update().Where(invitation.IDEQ(invite), invitation.ConsumedEQ(false), invitation.ExpiresAtGT(now)).SetConsumed(true).Save(ctx)
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrInvalid
		}
	}
	if err := tx.AccountUser.Create().SetID(uuid.NewString()).SetEmail(email).SetDisplayName(name).SetPasswordHash(hash).SetEmailVerifiedAt(now).Exec(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return ErrInvalid
		}
		return err
	}
	return tx.Commit()
}
func (s *Store) IssueInvitation(ctx context.Context, id string, expires time.Time) error {
	if !expires.After(time.Now()) || expires.After(time.Now().Add(30*24*time.Hour)) {
		return errors.New("invitation expiry must be within thirty days")
	}
	return s.db.Invitation.Create().SetID(id).SetExpiresAt(expires).Exec(ctx)
}
