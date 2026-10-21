package developer

import (
	"context"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/applicationversion"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
	"strings"
	"unicode/utf8"
)

// Review is an internal command; only an authenticated administrator adapter may expose it.
func (s *Service) Review(ctx context.Context, reviewer, id string, revision int, approve bool, reason string) (Application, error) {
	if reviewer == "" || !utf8.ValidString(reason) || utf8.RuneCountInString(reason) > 500 || !approve && strings.TrimSpace(reason) == "" {
		return Application{}, ErrInvalid
	}
	meta, err := persistence.Client(ctx, s.db).DeveloperApplication.Get(ctx, id)
	if ent.IsNotFound(err) {
		return Application{}, ErrMissing
	}
	if err != nil {
		return Application{}, err
	}
	owner, err := s.users.User(ctx, meta.OwnerID)
	if err != nil {
		return Application{}, err
	}
	ctx, tx, row, err := s.begin(ctx, owner, id, revision)
	if err != nil {
		return Application{}, err
	}
	defer tx.Rollback()
	latest, err := tx.ApplicationVersion.Query().Where(applicationversion.ApplicationID(id), applicationversion.Version(row.LatestVersion)).Only(ctx)
	if err != nil {
		return Application{}, err
	}
	if latest.Status != applicationversion.StatusPending || !row.Enabled || row.Deleted || row.Suspended {
		return Application{}, ErrConflict
	}
	status := applicationversion.StatusRejected
	action := "application.review.rejected"
	if approve {
		version, err := publicVersion(latest)
		if err != nil {
			return Application{}, err
		}
		if _, err := s.validate(version.Config); err != nil {
			return Application{}, err
		}
		if err := s.protocol.SetManagedClientEnabled(ctx, row.ClientID, row.OwnerID, false); err != nil {
			return Application{}, err
		}
		if err := s.protocol.InstallManagedClient(ctx, s.spec(row, version.Config, false), nil); err != nil {
			return Application{}, err
		}
		row.ActiveVersion = row.LatestVersion
		status = applicationversion.StatusApproved
		action = "application.review.approved"
	}
	if err := tx.ApplicationVersion.UpdateOneID(latest.ID).SetStatus(status).SetReviewReason(strings.TrimSpace(reason)).SetReviewerID(reviewer).Exec(ctx); err != nil {
		return Application{}, err
	}
	if err := tx.DeveloperApplication.UpdateOneID(id).SetActiveVersion(row.ActiveVersion).AddRevision(1).Exec(ctx); err != nil {
		return Application{}, err
	}
	row.Revision++
	if err := audit.RecordTarget(ctx, tx.Client(), reviewer, action, id, "success"); err != nil {
		return Application{}, err
	}
	if err := s.notify(ctx, row.OwnerID, id, action, strings.TrimSpace(reason)); err != nil {
		return Application{}, err
	}
	view, err := s.view(ctx, tx.Client(), row)
	if err != nil {
		return Application{}, err
	}
	return view, tx.Commit()
}
func (s *Service) HasPublishedApplications(ctx context.Context, owner string) (bool, error) {
	return persistence.Client(ctx, s.db).DeveloperApplication.Query().Where(developerapplication.OwnerID(owner), developerapplication.ActiveVersionGT(0), developerapplication.Enabled(true), developerapplication.Deleted(false)).Exist(ctx)
}
