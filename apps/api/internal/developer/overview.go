package developer

import (
	"context"
	"entgo.io/ent/dialect/sql"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

type Overview struct {
	Total            int `json:"total"`
	Pending          int `json:"pending"`
	PublishedEnabled int `json:"published_enabled"`
	Suspended        int `json:"suspended"`
}

func (s *Service) Overview(ctx context.Context) (Overview, error) {
	var result Overview
	db := persistence.Client(ctx, s.db)
	query := func() *ent.DeveloperApplicationQuery {
		return db.DeveloperApplication.Query().Where(developerapplication.Deleted(false))
	}
	var err error
	if result.Total, err = query().Count(ctx); err != nil {
		return result, err
	}
	if result.PublishedEnabled, err = query().Where(developerapplication.ActiveVersionGT(0), developerapplication.Enabled(true), developerapplication.Suspended(false)).Count(ctx); err != nil {
		return result, err
	}
	if result.Suspended, err = query().Where(developerapplication.Suspended(true)).Count(ctx); err != nil {
		return result, err
	}
	result.Pending, err = query().Where(func(selector *sql.Selector) {
		versions := sql.Table("application_versions").Schema("developer")
		selector.Where(sql.Exists(sql.Select(versions.C("id")).From(versions).Where(sql.And(sql.ColumnsEQ(versions.C("application_id"), selector.C("id")), sql.ColumnsEQ(versions.C("version"), selector.C("latest_version")), sql.EQ(versions.C("status"), "pending")))))
	}).Count(ctx)
	return result, err
}
