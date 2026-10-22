package identity

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"moetopia.local/aio/api/internal/persistence/ent/registrationevent"
	"time"
)

type RegistrationDay struct {
	Day           string `json:"day"`
	Partial       bool   `json:"partial"`
	Registrations *int   `json:"registrations,omitempty"`
}
type IdentityStatistics struct {
	TotalUsers       int               `json:"total_users"`
	EnabledUsers     int               `json:"enabled_users"`
	PendingDeletions int               `json:"pending_deletions"`
	RecordingSince   time.Time         `json:"recording_since"`
	Registrations    []RegistrationDay `json:"registrations" nullable:"false"`
}

func (s *Service) Statistics(ctx context.Context) (IdentityStatistics, error) {
	result := IdentityStatistics{Registrations: []RegistrationDay{}}
	db := persistence.Client(ctx, s.store.db)
	state, err := db.RegistrationStatisticsState.Get(ctx, "registration-statistics")
	if err != nil {
		return result, err
	}
	result.RecordingSince = state.StartedAt.UTC()
	if result.TotalUsers, err = db.AccountUser.Query().Count(ctx); err != nil {
		return result, err
	}
	if result.EnabledUsers, err = db.AccountUser.Query().Where(accountuser.Enabled(true), accountuser.DeletionDueAtIsNil()).Count(ctx); err != nil {
		return result, err
	}
	if result.PendingDeletions, err = db.AccountUser.Query().Where(accountuser.DeletionDueAtNotNil()).Count(ctx); err != nil {
		return result, err
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -29)
	var groups []struct {
		Day   string
		Count int
	}
	if err := db.RegistrationEvent.Query().Where(registrationevent.DayGTE(start.Format(time.DateOnly)), registrationevent.DayLT(today.AddDate(0, 0, 1).Format(time.DateOnly))).GroupBy(registrationevent.FieldDay).Aggregate(ent.Count()).Scan(ctx, &groups); err != nil {
		return result, err
	}
	counts := map[string]int{}
	for _, group := range groups {
		counts[group.Day] = group.Count
	}
	since := state.StartedAt.UTC().Format(time.DateOnly)
	for i := 0; i < 30; i++ {
		day := start.AddDate(0, 0, i).Format(time.DateOnly)
		entry := RegistrationDay{Day: day, Partial: day == since || day == today.Format(time.DateOnly)}
		if day >= since {
			count := counts[day]
			entry.Registrations = &count
		}
		result.Registrations = append(result.Registrations, entry)
	}
	return result, nil
}

func (s *Store) cleanupRegistrationStatistics(ctx context.Context) error {
	ids, err := s.db.RegistrationEvent.Query().Where(registrationevent.DayLT(time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -29).Format(time.DateOnly))).Limit(1000).IDs(ctx)
	if err != nil || len(ids) == 0 {
		return err
	}
	_, err = s.db.RegistrationEvent.Delete().Where(registrationevent.IDIn(ids...)).Exec(ctx)
	return err
}
