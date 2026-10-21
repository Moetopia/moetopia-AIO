package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/emaildelivery"
)

type Message struct {
	UserID     string
	To         string
	Subject    string
	Text       string
	Suppressed bool
}
type Sender interface {
	Send(context.Context, Message) error
}
type Outbox struct {
	db     *ent.Client
	box    *cryptobox.Box
	sender Sender
}

func New(db *ent.Client, box *cryptobox.Box, sender Sender) *Outbox { return &Outbox{db, box, sender} }

func (o *Outbox) Enqueue(ctx context.Context, transaction *ent.Client, message Message, expires time.Time) error {
	id := uuid.NewString()
	plain, err := json.Marshal(message)
	if err != nil {
		return err
	}
	payload, err := o.box.Seal(plain, "notification:"+id)
	if err != nil {
		return err
	}
	return transaction.EmailDelivery.Create().SetID(id).SetUserID(message.UserID).SetPayload(payload).SetAvailableAt(time.Now()).SetLeaseUntil(time.Unix(0, 0)).SetExpiresAt(expires).Exec(ctx)
}

func (o *Outbox) PurgeUser(ctx context.Context, db *ent.Client, userID string) error {
	_, err := db.EmailDelivery.Delete().Where(emaildelivery.UserID(userID)).Exec(ctx)
	return err
}

func (o *Outbox) Deliver(ctx context.Context) error {
	now := time.Now()
	if _, err := o.db.EmailDelivery.Delete().Where(emaildelivery.ExpiresAtLTE(now)).Exec(ctx); err != nil {
		return err
	}
	jobs, err := o.db.EmailDelivery.Query().Where(emaildelivery.AvailableAtLTE(now), emaildelivery.LeaseUntilLTE(now), emaildelivery.AttemptsLT(5)).Order(ent.Asc(emaildelivery.FieldAvailableAt)).Limit(5).All(ctx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		lease := time.Now().Add(30 * time.Second)
		count, err := o.db.EmailDelivery.Update().Where(emaildelivery.IDEQ(job.ID), emaildelivery.LeaseUntilLTE(now), emaildelivery.AttemptsEQ(job.Attempts)).SetLeaseUntil(lease).AddAttempts(1).Save(ctx)
		if err != nil {
			return err
		}
		if count != 1 {
			continue
		}
		plain, err := o.box.Open(job.Payload, "notification:"+job.ID)
		var message Message
		if err == nil {
			err = json.Unmarshal(plain, &message)
		}
		if err == nil && !message.Suppressed {
			err = o.sender.Send(ctx, message)
		}
		if err == nil {
			_, err = o.db.EmailDelivery.Delete().Where(emaildelivery.IDEQ(job.ID), emaildelivery.LeaseUntilEQ(lease)).Exec(ctx)
		} else {
			slog.Warn("notification delivery will retry", "attempt", job.Attempts+1)
			_, err = o.db.EmailDelivery.Update().Where(emaildelivery.IDEQ(job.ID), emaildelivery.LeaseUntilEQ(lease)).SetAvailableAt(time.Now().Add(time.Duration(1<<uint(job.Attempts))*5*time.Second + time.Duration(rand.IntN(1000))*time.Millisecond)).SetLeaseUntil(time.Unix(0, 0)).Save(ctx)
		}
		if err != nil {
			return fmt.Errorf("notification lease update failed")
		}
	}
	return nil
}
