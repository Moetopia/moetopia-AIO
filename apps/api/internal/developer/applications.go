package developer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/applicationversion"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
	"moetopia.local/aio/api/internal/persistence/ent/predicate"
)

func (s *Service) Create(ctx context.Context, user identity.User, public bool, config Config) (Created, error) {
	var result Created
	config, err := s.validate(config)
	if err != nil {
		return result, err
	}
	secret := ""
	var hash []byte
	if !public {
		secret, hash, err = authorization.NewClientSecret()
		if err != nil {
			return result, err
		}
	}
	tx, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	ctx = persistence.WithClient(ctx, tx.Client())
	if err := s.users.VerifyVersion(ctx, user.ID, user.SecurityVersion); err != nil {
		return result, err
	}
	count, err := tx.DeveloperApplication.Query().Where(developerapplication.OwnerID(user.ID), developerapplication.Deleted(false)).Count(ctx)
	if err != nil {
		return result, err
	}
	if count >= 20 {
		return result, ErrQuota
	}
	id := uuid.NewString()
	row, err := tx.DeveloperApplication.Create().SetID(id).SetOwnerID(user.ID).SetClientID("mt_app_" + id).SetPublic(public).Save(ctx)
	if err != nil {
		return result, err
	}
	if err := tx.ApplicationVersion.Create().SetID(uuid.NewString()).SetApplicationID(id).SetVersion(1).SetConfig(configMap(config)).Exec(ctx); err != nil {
		return result, err
	}
	if err := s.protocol.InstallManagedClient(ctx, s.spec(row, config, true), hash); err != nil {
		return result, err
	}
	if err := audit.RecordTarget(ctx, tx.Client(), user.ID, "application.created", id, "success"); err != nil {
		return result, err
	}
	view, err := s.view(ctx, tx.Client(), row)
	if err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return Created{view, secret}, nil
}
func (s *Service) UpdateDraft(ctx context.Context, user identity.User, id string, revision int, config Config) (Application, error) {
	config, err := s.validate(config)
	if err != nil {
		return Application{}, err
	}
	ctx, tx, row, err := s.begin(ctx, user, id, revision)
	if err != nil {
		return Application{}, err
	}
	defer tx.Rollback()
	latest, err := tx.ApplicationVersion.Query().Where(applicationversion.ApplicationID(id), applicationversion.Version(row.LatestVersion)).Only(ctx)
	if err != nil {
		return Application{}, err
	}
	if latest.Status == applicationversion.StatusPending {
		return Application{}, ErrConflict
	}
	if latest.Status != applicationversion.StatusDraft {
		row.LatestVersion++
		if err := tx.ApplicationVersion.Create().SetID(uuid.NewString()).SetApplicationID(id).SetVersion(row.LatestVersion).SetConfig(configMap(config)).Exec(ctx); err != nil {
			return Application{}, err
		}
	} else {
		if err := tx.ApplicationVersion.UpdateOneID(latest.ID).SetConfig(configMap(config)).Exec(ctx); err != nil {
			return Application{}, err
		}
	}
	if row.ActiveVersion == 0 {
		if err := s.protocol.SetManagedClientEnabled(ctx, row.ClientID, user.ID, false); err != nil {
			return Application{}, err
		}
		if err := s.protocol.InstallManagedClient(ctx, s.spec(row, config, true), nil); err != nil {
			return Application{}, err
		}
	}
	if err := tx.DeveloperApplication.UpdateOneID(id).SetLatestVersion(row.LatestVersion).AddRevision(1).Exec(ctx); err != nil {
		return Application{}, err
	}
	row.Revision++
	if err := audit.RecordTarget(ctx, tx.Client(), user.ID, "application.draft.updated", id, "success"); err != nil {
		return Application{}, err
	}
	view, err := s.view(ctx, tx.Client(), row)
	if err != nil {
		return Application{}, err
	}
	return view, tx.Commit()
}
func (s *Service) Submit(ctx context.Context, user identity.User, id string, revision int) (Application, error) {
	ctx, tx, row, err := s.begin(ctx, user, id, revision)
	if err != nil {
		return Application{}, err
	}
	defer tx.Rollback()
	latest, err := tx.ApplicationVersion.Query().Where(applicationversion.ApplicationID(id), applicationversion.Version(row.LatestVersion)).Only(ctx)
	if err != nil {
		return Application{}, err
	}
	if latest.Status != applicationversion.StatusDraft || !row.Enabled || row.Suspended {
		return Application{}, ErrConflict
	}
	if err := tx.ApplicationVersion.UpdateOneID(latest.ID).SetStatus(applicationversion.StatusPending).SetReviewReason("").Exec(ctx); err != nil {
		return Application{}, err
	}
	if err := tx.DeveloperApplication.UpdateOneID(id).AddRevision(1).Exec(ctx); err != nil {
		return Application{}, err
	}
	row.Revision++
	if err := audit.RecordTarget(ctx, tx.Client(), user.ID, "application.review.submitted", id, "success"); err != nil {
		return Application{}, err
	}
	view, err := s.view(ctx, tx.Client(), row)
	if err != nil {
		return Application{}, err
	}
	return view, tx.Commit()
}
func (s *Service) SetEnabled(ctx context.Context, user identity.User, id string, revision int, enabled bool) (Application, error) {
	ctx, tx, row, err := s.begin(ctx, user, id, revision)
	if err != nil {
		return Application{}, err
	}
	defer tx.Rollback()
	if err := s.protocol.SetManagedClientEnabled(ctx, row.ClientID, user.ID, enabled && !row.Suspended); err != nil {
		return Application{}, err
	}
	if err := tx.DeveloperApplication.UpdateOneID(id).SetEnabled(enabled).AddRevision(1).Exec(ctx); err != nil {
		return Application{}, err
	}
	row.Enabled = enabled
	row.Revision++
	if err := audit.RecordTarget(ctx, tx.Client(), user.ID, "application.availability.changed", id, "success"); err != nil {
		return Application{}, err
	}
	view, err := s.view(ctx, tx.Client(), row)
	if err != nil {
		return Application{}, err
	}
	return view, tx.Commit()
}
func (s *Service) RotateSecret(ctx context.Context, user identity.User, id string, revision int) (string, error) {
	ctx, tx, row, err := s.begin(ctx, user, id, revision)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if row.Public {
		return "", ErrInvalid
	}
	secret, err := s.protocol.RotateManagedSecret(ctx, row.ClientID, user.ID)
	if err != nil {
		return "", err
	}
	if err := tx.DeveloperApplication.UpdateOneID(id).AddRevision(1).Exec(ctx); err != nil {
		return "", err
	}
	if err := audit.RecordTarget(ctx, tx.Client(), user.ID, "application.secret.rotated", id, "success"); err != nil {
		return "", err
	}
	if err := s.notify(ctx, user.ID, id, "application.secret.rotated", "If this was not you, disable the application and secure your account. 如非本人操作，请立即停用应用并保护账号。"); err != nil {
		return "", err
	}
	return secret, tx.Commit()
}
func (s *Service) Delete(ctx context.Context, user identity.User, id string, revision int) error {
	ctx, tx, row, err := s.begin(ctx, user, id, revision)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if row.Enabled {
		return ErrConflict
	}
	if err := tx.DeveloperApplication.UpdateOneID(id).SetDeleted(true).AddRevision(1).Exec(ctx); err != nil {
		return err
	}
	if err := audit.RecordTarget(ctx, tx.Client(), user.ID, "application.deleted", id, "success"); err != nil {
		return err
	}
	return tx.Commit()
}

type Applications struct {
	Items      []Application `json:"items"`
	NextCursor string        `json:"next_cursor"`
}
type listCursor struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func (s *Service) List(ctx context.Context, owner, cursor string) (Applications, error) {
	db := persistence.Client(ctx, s.db)
	return s.list(ctx, db.DeveloperApplication.Query().Where(developerapplication.OwnerID(owner), developerapplication.Deleted(false)), cursor)
}
func (s *Service) list(ctx context.Context, query *ent.DeveloperApplicationQuery, cursor string) (Applications, error) {
	db := persistence.Client(ctx, s.db)
	result := Applications{Items: []Application{}}
	if cursor != "" {
		if len(cursor) > 300 {
			return result, ErrInvalid
		}
		value, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return result, ErrInvalid
		}
		var c listCursor
		if json.Unmarshal(value, &c) != nil || c.At.IsZero() {
			return result, ErrInvalid
		}
		if _, err := uuid.Parse(c.ID); err != nil {
			return result, ErrInvalid
		}
		query.Where(developerapplication.Or(developerapplication.CreatedAtLT(c.At), developerapplication.And(developerapplication.CreatedAtEQ(c.At), developerapplication.IDLT(c.ID))))
	}
	rows, err := query.Order(ent.Desc(developerapplication.FieldCreatedAt), ent.Desc(developerapplication.FieldID)).Limit(21).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 20 {
		rows = rows[:20]
		last := rows[len(rows)-1]
		raw, _ := json.Marshal(listCursor{last.CreatedAt, last.ID})
		result.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	if len(rows) == 0 {
		return result, nil
	}
	conditions := make([]predicate.ApplicationVersion, 0, len(rows))
	for _, row := range rows {
		conditions = append(conditions, applicationversion.And(applicationversion.ApplicationID(row.ID), applicationversion.VersionIn(row.LatestVersion, row.ActiveVersion)))
	}
	versions, err := db.ApplicationVersion.Query().Where(applicationversion.Or(conditions...)).All(ctx)
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		view, err := assemble(row, versions)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, view)
	}
	return result, nil
}
