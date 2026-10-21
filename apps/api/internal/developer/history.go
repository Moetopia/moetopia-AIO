package developer

import (
	"context"
	"time"

	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/applicationversion"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

type VersionRecord struct {
	Version
	CreatedAt time.Time `json:"created_at"`
}

type VersionHistory struct {
	Items         []VersionRecord `json:"items"`
	NextBefore    int             `json:"next_before"`
	ActiveVersion int             `json:"active_version"`
	LatestVersion int             `json:"latest_version"`
}

func (s *Service) History(ctx context.Context, owner, id string, before int) (VersionHistory, error) {
	if owner == "" {
		return VersionHistory{}, ErrMissing
	}
	return s.history(ctx, owner, id, before)
}

func (s *Service) AdminHistory(ctx context.Context, id string, before int) (VersionHistory, error) {
	return s.history(ctx, "", id, before)
}

func (s *Service) history(ctx context.Context, owner, id string, before int) (VersionHistory, error) {
	result := VersionHistory{Items: []VersionRecord{}}
	if before < 0 {
		return result, ErrInvalid
	}
	db := persistence.Client(ctx, s.db)
	query := db.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.Deleted(false))
	if owner != "" {
		query.Where(developerapplication.OwnerID(owner))
	}
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return result, ErrMissing
	}
	if err != nil {
		return result, err
	}
	versions := db.ApplicationVersion.Query().Where(applicationversion.ApplicationID(id), applicationversion.VersionLTE(row.LatestVersion))
	if before > 0 {
		versions.Where(applicationversion.VersionLT(before))
	}
	rows, err := versions.Order(ent.Desc(applicationversion.FieldVersion)).Limit(21).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 20 {
		rows = rows[:20]
		result.NextBefore = rows[19].Version
	}
	result.ActiveVersion, result.LatestVersion = row.ActiveVersion, row.LatestVersion
	for _, version := range rows {
		view, err := publicVersion(version)
		if err != nil {
			return VersionHistory{}, err
		}
		result.Items = append(result.Items, VersionRecord{view, version.CreatedAt})
	}
	return result, nil
}
