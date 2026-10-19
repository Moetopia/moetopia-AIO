package audit

import (
	"context"
	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
	"strings"
	"time"
)

type Event struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

func Record(ctx context.Context, db *ent.Client, userID, action string) error {
	return RecordTarget(ctx, db, userID, action, userID, "success")
}

type requestKey struct{}

func WithRequest(ctx context.Context, id string) context.Context {
	if len(id) > 128 || strings.ContainsAny(id, "\r\n\t ") {
		id = uuid.NewString()
	}
	return context.WithValue(ctx, requestKey{}, id)
}
func RecordTarget(ctx context.Context, db *ent.Client, userID, action, target, outcome string) error {
	id, _ := ctx.Value(requestKey{}).(string)
	return db.SecurityEvent.Create().SetID(uuid.NewString()).SetUserID(userID).SetAction(action).SetTargetID(target).SetOutcome(outcome).SetRequestID(id).Exec(ctx)
}
func List(ctx context.Context, db *ent.Client, userID string) ([]Event, error) {
	rows, err := db.SecurityEvent.Query().Where(securityevent.UserID(userID), securityevent.CreatedAtGT(time.Now().Add(-90*24*time.Hour))).Order(ent.Desc(securityevent.FieldCreatedAt), ent.Desc(securityevent.FieldID)).Limit(50).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Event, 0, len(rows))
	for _, row := range rows {
		items = append(items, Event{row.ID, row.Action, row.CreatedAt})
	}
	return items, nil
}
func Cleanup(ctx context.Context, db *ent.Client) error {
	_, err := db.SecurityEvent.Delete().Where(securityevent.CreatedAtLT(time.Now().Add(-90 * 24 * time.Hour))).Exec(ctx)
	return err
}
