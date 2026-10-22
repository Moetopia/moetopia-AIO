package developer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

type TestMembers struct {
	Revision int                         `json:"revision"`
	Limit    int                         `json:"limit"`
	Items    []authorization.TestSubject `json:"items" nullable:"false"`
}

func (s *Service) TestMembers(ctx context.Context, owner, id string) (TestMembers, error) {
	tx, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return TestMembers{}, err
	}
	defer tx.Rollback()
	ctx = persistence.WithClient(ctx, tx.Client())
	app, err := tx.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.OwnerID(owner), developerapplication.Deleted(false)).ForShare().Only(ctx)
	if ent.IsNotFound(err) {
		return TestMembers{}, ErrMissing
	}
	if err != nil {
		return TestMembers{}, err
	}
	items, err := s.protocol.ListTestSubjects(ctx, app.ClientID, owner)
	if err != nil {
		return TestMembers{}, err
	}
	return TestMembers{app.Revision, authorization.MaxTestSubjects, items}, tx.Commit()
}

func (s *Service) ChangeTestMember(ctx context.Context, user identity.User, id string, revision int, target string, add bool) (TestMembers, error) {
	parsed, err := uuid.Parse(target)
	if err != nil || parsed == uuid.Nil || parsed.String() == user.ID {
		return TestMembers{}, ErrInvalid
	}
	target = parsed.String()
	var tx *persistence.Unit
	var row *ent.DeveloperApplication
	if add {
		ctx, tx, row, err = s.beginTestMember(ctx, user, id, revision, target)
	} else {
		ctx, tx, row, err = s.begin(ctx, user, id, revision)
	}
	if err != nil {
		return TestMembers{}, err
	}
	defer tx.Rollback()
	if add && row.ActiveVersion != 0 {
		return TestMembers{}, ErrConflict
	}
	if err := s.protocol.ChangeTestSubject(ctx, row.ClientID, user.ID, target, add); err != nil {
		switch {
		case errors.Is(err, authorization.ErrTestSubjectConflict):
			return TestMembers{}, ErrConflict
		case errors.Is(err, authorization.ErrTestSubjectLimit):
			return TestMembers{}, ErrQuota
		default:
			return TestMembers{}, err
		}
	}
	action := "application.test_member.removed"
	if add {
		action = "application.test_member.added"
	}
	if err := audit.RecordTarget(ctx, tx.Client(), user.ID, action, id+":"+target, "success"); err != nil {
		return TestMembers{}, err
	}
	if err := s.notify(ctx, user.ID, id, action, ""); err != nil {
		return TestMembers{}, err
	}
	if err := s.notifyTestMember(ctx, target, id, add); err != nil {
		return TestMembers{}, err
	}
	if err := tx.DeveloperApplication.UpdateOneID(id).AddRevision(1).Exec(ctx); err != nil {
		return TestMembers{}, err
	}
	items, err := s.protocol.ListTestSubjects(ctx, row.ClientID, user.ID)
	if err != nil {
		return TestMembers{}, err
	}
	result := TestMembers{row.Revision + 1, authorization.MaxTestSubjects, items}
	return result, tx.Commit()
}

func (s *Service) beginTestMember(ctx context.Context, owner identity.User, id string, revision int, target string) (context.Context, *persistence.Unit, *ent.DeveloperApplication, error) {
	member, err := s.users.User(ctx, target)
	if errors.Is(err, identity.ErrMissing) {
		err = ErrInvalid
	}
	if err != nil {
		return ctx, nil, nil, err
	}
	tx, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return ctx, nil, nil, err
	}
	ctx = persistence.WithClient(ctx, tx.Client())
	users := []identity.User{owner, member}
	// Cross-owner additions acquire both identity locks in the same order.
	slices.SortFunc(users, func(a, b identity.User) int { return strings.Compare(a.ID, b.ID) })
	for _, user := range users {
		if err := s.users.VerifyVersion(ctx, user.ID, user.SecurityVersion); err != nil {
			_ = tx.Rollback()
			if user.ID == target && errors.Is(err, identity.ErrMissing) {
				err = ErrInvalid
			}
			return ctx, nil, nil, err
		}
	}
	row, err := tx.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.OwnerID(owner.ID), developerapplication.Deleted(false)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		err = ErrMissing
	}
	if err == nil && row.Revision != revision {
		err = ErrConflict
	}
	if err != nil {
		_ = tx.Rollback()
		return ctx, nil, nil, err
	}
	return ctx, tx, row, nil
}

func (s *Service) notifyTestMember(ctx context.Context, target, id string, add bool) error {
	email, err := s.users.VerifiedEmail(ctx, target)
	if !add && errors.Is(err, identity.ErrMissing) {
		return nil
	}
	if err != nil {
		return err
	}
	if s.outbox == nil {
		return errors.New("application notification delivery unavailable")
	}
	status := "removed / 已移除"
	if add {
		status = "added / 已添加"
	}
	message := notifications.Message{UserID: target, To: email, Subject: "Moetopia application test access / 应用测试资格变更", Text: fmt.Sprintf("Application %s: test access %s. This grants no application management permissions.\n应用测试资格已变更；此资格不包含应用管理权限。", id, status)}
	return s.outbox.Enqueue(ctx, persistence.Client(ctx, s.db), message, time.Now().Add(24*time.Hour))
}
