package identity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"entgo.io/ent/dialect/sql"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountchallenge"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
)

type ManagedUser struct {
	User
	Enabled        bool      `json:"enabled"`
	DisabledReason string    `json:"disabled_reason"`
	CreatedAt      time.Time `json:"created_at"`
}
type ManagedUsers struct {
	Items      []ManagedUser `json:"items"`
	NextCursor string        `json:"next_cursor"`
}
type userCursor struct {
	Query string `json:"query"`
	ID    string `json:"id"`
}

var managedColumns = []string{accountuser.FieldID, accountuser.FieldEmail, accountuser.FieldDisplayName, accountuser.FieldBio, accountuser.FieldAvatarDigest, accountuser.FieldRevision, accountuser.FieldEnabled, accountuser.FieldDisabledReason, accountuser.FieldCreatedAt}

func managedUser(row *ent.AccountUser) ManagedUser {
	return ManagedUser{public(row), row.Enabled, row.DisabledReason, row.CreatedAt}
}
func (s *Service) AdminUsers(ctx context.Context, search, cursor string) (ManagedUsers, error) {
	result := ManagedUsers{Items: []ManagedUser{}}
	search = strings.ToLower(strings.TrimSpace(search))
	if !utf8.ValidString(search) || len(search) > 254 || search != "" && utf8.RuneCountInString(search) < 2 {
		return result, ErrInvalid
	}
	query := persistence.Client(ctx, s.store.db).AccountUser.Query()
	if search != "" {
		query.Where(func(selector *sql.Selector) {
			selector.Where(sql.Or(sql.HasPrefixFold(selector.C(accountuser.FieldEmail), search), sql.HasPrefixFold(selector.C(accountuser.FieldDisplayName), search), sql.EQ(selector.C(accountuser.FieldID), search)))
		})
	}
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(cursor) > 1000 {
			return result, ErrInvalid
		}
		var c userCursor
		if json.Unmarshal(data, &c) != nil || c.Query != search {
			return result, ErrInvalid
		}
		if _, err := uuid.Parse(c.ID); err != nil {
			return result, ErrInvalid
		}
		query.Where(accountuser.IDGT(c.ID))
	}
	rows, err := query.Order(ent.Asc(accountuser.FieldID)).Limit(51).Select(managedColumns...).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 50 {
		rows = rows[:50]
		raw, _ := json.Marshal(userCursor{search, rows[49].ID})
		result.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	for _, row := range rows {
		result.Items = append(result.Items, managedUser(row))
	}
	return result, nil
}
func (s *Service) AdminSetEnabled(ctx context.Context, actor, id string, revision int, enabled bool, reason string) (ManagedUser, error) {
	reason = strings.TrimSpace(reason)
	if actor == "" || actor == id || revision < 0 || !utf8.ValidString(reason) || utf8.RuneCountInString(reason) > 500 || !enabled && reason == "" {
		return ManagedUser{}, ErrInvalid
	}
	unit, err := persistence.Begin(ctx, s.store.db)
	if err != nil {
		return ManagedUser{}, err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	row, err := unit.AccountUser.Query().Where(accountuser.ID(id)).ForUpdate().Select(managedColumns...).Only(ctx)
	if ent.IsNotFound(err) {
		return ManagedUser{}, ErrMissing
	}
	if err != nil {
		return ManagedUser{}, err
	}
	if row.Revision != revision {
		return ManagedUser{}, ErrConflict
	}
	if enabled {
		reason = ""
	}
	if err := unit.AccountUser.UpdateOneID(id).SetEnabled(enabled).SetDisabledReason(reason).AddRevision(1).AddSecurityVersion(1).Exec(ctx); err != nil {
		return ManagedUser{}, err
	}
	if s.store.revoker == nil {
		return ManagedUser{}, ErrInvalid
	}
	if err := s.store.revoker.RevokeUser(ctx, id); err != nil {
		return ManagedUser{}, err
	}
	if err := unit.AccountChallenge.Update().Where(accountchallenge.UserID(id)).SetConsumed(true).Exec(ctx); err != nil {
		return ManagedUser{}, err
	}
	action := "account.disabled"
	if enabled {
		action = "account.enabled"
	}
	if err := audit.RecordTarget(ctx, unit.Client(), actor, action, id, "success"); err != nil {
		return ManagedUser{}, err
	}
	if err := s.store.outbox.Enqueue(ctx, unit.Client(), notifications.Message{To: row.Email, Subject: "Moetopia account status updated", Text: "Your Moetopia account status changed (" + action + "). Contact the operator if you need assistance.\n你的 Moetopia 账号状态已由平台变更，如有疑问请联系维护者。"}, time.Now().Add(24*time.Hour)); err != nil {
		return ManagedUser{}, err
	}
	row.Enabled, row.DisabledReason, row.Revision = enabled, reason, row.Revision+1
	return managedUser(row), unit.Commit()
}
