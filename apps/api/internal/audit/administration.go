package audit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/securityevent"
)

var ErrCursor = errors.New("invalid audit cursor")

type AdminEvent struct {
	ID        string    `json:"id"`
	ActorID   string    `json:"actor_id"`
	TargetID  string    `json:"target_id"`
	Action    string    `json:"action"`
	Outcome   string    `json:"outcome"`
	RequestID string    `json:"request_id"`
	CreatedAt time.Time `json:"created_at"`
	Reason    string    `json:"reason"`
}
type AdminEvents struct {
	Items      []AdminEvent `json:"items"`
	NextCursor string       `json:"next_cursor"`
}
type cursor struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func AdminList(ctx context.Context, db *ent.Client, value string) (AdminEvents, error) {
	result := AdminEvents{Items: []AdminEvent{}}
	query := db.SecurityEvent.Query().Where(securityevent.CreatedAtGT(time.Now().Add(-90 * 24 * time.Hour)))
	if value != "" {
		if len(value) > 300 {
			return result, ErrCursor
		}
		data, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return result, ErrCursor
		}
		var c cursor
		if json.Unmarshal(data, &c) != nil || c.At.IsZero() {
			return result, ErrCursor
		}
		if _, err := uuid.Parse(c.ID); err != nil {
			return result, ErrCursor
		}
		query.Where(securityevent.Or(securityevent.CreatedAtLT(c.At), securityevent.And(securityevent.CreatedAtEQ(c.At), securityevent.IDLT(c.ID))))
	}
	rows, err := query.Order(ent.Desc(securityevent.FieldCreatedAt), ent.Desc(securityevent.FieldID)).Limit(51).All(ctx)
	if err != nil {
		return result, err
	}
	if len(rows) > 50 {
		rows = rows[:50]
		raw, _ := json.Marshal(cursor{rows[49].CreatedAt, rows[49].ID})
		result.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	for _, row := range rows {
		result.Items = append(result.Items, AdminEvent{row.ID, row.UserID, row.TargetID, row.Action, row.Outcome, row.RequestID, row.CreatedAt, row.Reason})
	}
	return result, nil
}
