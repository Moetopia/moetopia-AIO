package notifications

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/emaildelivery"
	"time"
)

var ErrDeliveryState = errors.New("notification delivery state changed")

type FailedDelivery struct {
	ID          string    `json:"id"`
	Attempts    int       `json:"attempts"`
	AvailableAt time.Time `json:"available_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}
type FailedDeliveries struct {
	Items     []FailedDelivery `json:"items" nullable:"false"`
	NextAfter string           `json:"next_after"`
}

func (o *Outbox) Failed(ctx context.Context, after string) (FailedDeliveries, error) {
	result := FailedDeliveries{Items: []FailedDelivery{}}
	query := persistence.Client(ctx, o.db).EmailDelivery.Query().Where(emaildelivery.AttemptsGTE(5), emaildelivery.ExpiresAtGT(time.Now()), emaildelivery.LeaseUntilLTE(time.Now()))
	if after != "" {
		if id, err := uuid.Parse(after); err != nil || id.String() != after {
			return result, ErrDeliveryState
		}
		query.Where(emaildelivery.IDGT(after))
	}
	rows, err := query.Select(emaildelivery.FieldID, emaildelivery.FieldAttempts, emaildelivery.FieldAvailableAt, emaildelivery.FieldExpiresAt).Order(ent.Asc(emaildelivery.FieldID)).Limit(51).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 50 {
		rows = rows[:50]
		result.NextAfter = rows[49].ID
	}
	for _, row := range rows {
		result.Items = append(result.Items, FailedDelivery{row.ID, row.Attempts, row.AvailableAt, row.ExpiresAt})
	}
	return result, nil
}

func (o *Outbox) Retry(ctx context.Context, id string, attempts int) error {
	if parsed, err := uuid.Parse(id); err != nil || parsed.String() != id || attempts < 5 {
		return ErrDeliveryState
	}
	now := time.Now()
	count, err := persistence.Client(ctx, o.db).EmailDelivery.Update().Where(emaildelivery.ID(id), emaildelivery.Attempts(attempts), emaildelivery.AttemptsGTE(5), emaildelivery.ExpiresAtGT(now), emaildelivery.LeaseUntilLTE(now)).SetAttempts(0).SetAvailableAt(now).SetLeaseUntil(time.Unix(0, 0)).Save(ctx)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrDeliveryState
	}
	return nil
}
