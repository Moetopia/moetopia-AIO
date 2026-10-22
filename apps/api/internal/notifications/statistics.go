package notifications

import (
	"context"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/emaildelivery"
	"time"
)

type QueueStatistics struct {
	Queued         int `json:"queued"`
	RetryExhausted int `json:"retry_exhausted"`
	InFlight       int `json:"in_flight"`
}

func (o *Outbox) Statistics(ctx context.Context) (QueueStatistics, error) {
	var result QueueStatistics
	db := persistence.Client(ctx, o.db)
	var err error
	now := time.Now()
	if result.Queued, err = db.EmailDelivery.Query().Where(emaildelivery.ExpiresAtGT(now), emaildelivery.AttemptsLT(5), emaildelivery.LeaseUntilLTE(now)).Count(ctx); err != nil {
		return result, err
	}
	if result.InFlight, err = db.EmailDelivery.Query().Where(emaildelivery.ExpiresAtGT(now), emaildelivery.LeaseUntilGT(now)).Count(ctx); err != nil {
		return result, err
	}
	result.RetryExhausted, err = db.EmailDelivery.Query().Where(emaildelivery.ExpiresAtGT(now), emaildelivery.AttemptsGTE(5), emaildelivery.LeaseUntilLTE(now)).Count(ctx)
	return result, err
}
