package authorization

import (
	"context"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/persistence/ent/protocolevent"
)

const StatisticsDays = 30

type ProtocolCounts struct {
	Authorizations int `json:"authorizations"`
	CodeExchanges  int `json:"code_exchanges"`
	Refreshes      int `json:"refreshes"`
	MachineTokens  int `json:"machine_tokens"`
}

type ProtocolDay struct {
	Day     string          `json:"day"`
	Partial bool            `json:"partial"`
	Counts  *ProtocolCounts `json:"counts,omitempty" doc:"Omitted before recording began; present zero counts mean no committed events."`
}

type ProtocolStatistics struct {
	Timezone       string         `json:"timezone"`
	RecordingSince time.Time      `json:"recording_since"`
	WindowStart    string         `json:"window_start"`
	WindowEnd      string         `json:"window_end"`
	GeneratedAt    time.Time      `json:"generated_at"`
	Totals         ProtocolCounts `json:"totals"`
	Days           []ProtocolDay  `json:"days" nullable:"false"`
}

func (s *Store) recordProtocolEvent(ctx context.Context, clientID string, kind protocolevent.Kind) error {
	return persistence.Client(ctx, s.database(ctx)).ProtocolEvent.Create().SetID(uuid.NewString()).SetClientID(clientID).SetDay(time.Now().UTC().Format(time.DateOnly)).SetKind(kind).Exec(ctx)
}

func (p *Provider) Statistics(ctx context.Context, clientID, owner string) (ProtocolStatistics, error) {
	db := persistence.Client(ctx, p.store.client)
	client, err := db.OAuthClient.Query().Where(oauthclient.ID(clientID), oauthclient.OwnerID(owner)).Only(ctx)
	if ent.IsNotFound(err) || owner == "" {
		return ProtocolStatistics{}, ErrGrantMissing
	}
	if err != nil {
		return ProtocolStatistics{}, err
	}
	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, 1-StatisticsDays)
	end := today.AddDate(0, 0, 1)
	var groups []struct {
		Day   string
		Kind  protocolevent.Kind
		Count int
	}
	if err := db.ProtocolEvent.Query().Where(protocolevent.ClientID(clientID), protocolevent.DayGTE(start.Format(time.DateOnly)), protocolevent.DayLT(end.Format(time.DateOnly))).GroupBy(protocolevent.FieldDay, protocolevent.FieldKind).Aggregate(ent.Count()).Scan(ctx, &groups); err != nil {
		return ProtocolStatistics{}, err
	}
	counts := map[string]*ProtocolCounts{}
	for _, group := range groups {
		day := counts[group.Day]
		if day == nil {
			day = &ProtocolCounts{}
			counts[group.Day] = day
		}
		switch group.Kind {
		case protocolevent.KindAuthorization:
			day.Authorizations += group.Count
		case protocolevent.KindCodeExchange:
			day.CodeExchanges += group.Count
		case protocolevent.KindRefresh:
			day.Refreshes += group.Count
		case protocolevent.KindMachineToken:
			day.MachineTokens += group.Count
		}
	}
	result := ProtocolStatistics{Timezone: "UTC", RecordingSince: client.StatisticsSince.UTC(), WindowStart: start.Format(time.DateOnly), WindowEnd: end.Format(time.DateOnly), Days: make([]ProtocolDay, 0, StatisticsDays)}
	since := client.StatisticsSince.UTC().Format(time.DateOnly)
	for i := 0; i < StatisticsDays; i++ {
		day := start.AddDate(0, 0, i).Format(time.DateOnly)
		item := ProtocolDay{Day: day, Partial: day == today.Format(time.DateOnly) || day == since}
		if day >= since {
			item.Counts = counts[day]
			if item.Counts == nil {
				item.Counts = &ProtocolCounts{}
			}
			result.Totals.Authorizations += item.Counts.Authorizations
			result.Totals.CodeExchanges += item.Counts.CodeExchanges
			result.Totals.Refreshes += item.Counts.Refreshes
			result.Totals.MachineTokens += item.Counts.MachineTokens
		}
		result.Days = append(result.Days, item)
	}
	result.GeneratedAt = time.Now().UTC()
	return result, nil
}

func (p *Provider) cleanupStatistics(ctx context.Context) error {
	db := p.store.database(ctx)
	cutoff := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1-StatisticsDays).Format(time.DateOnly)
	ids, err := db.ProtocolEvent.Query().Where(protocolevent.DayLT(cutoff)).Order(ent.Asc(protocolevent.FieldDay), ent.Asc(protocolevent.FieldID)).Limit(1000).IDs(ctx)
	if err != nil || len(ids) == 0 {
		return err
	}
	_, err = db.ProtocolEvent.Delete().Where(protocolevent.IDIn(ids...)).Exec(ctx)
	return err
}
