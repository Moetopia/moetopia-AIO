-- +goose Up
ALTER TABLE "authorization".clients ADD COLUMN statistics_since timestamptz NOT NULL DEFAULT now();
CREATE TABLE "authorization".protocol_events (
    id text PRIMARY KEY,
    client_id text NOT NULL REFERENCES "authorization".clients(id) ON DELETE CASCADE,
    day text NOT NULL CHECK (day ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'),
    kind text NOT NULL CHECK (kind IN ('authorization', 'code_exchange', 'refresh', 'machine_token'))
);
CREATE INDEX protocol_events_client_day_kind_idx ON "authorization".protocol_events (client_id, day, kind);
CREATE INDEX protocol_events_day_idx ON "authorization".protocol_events (day);

-- +goose Down
DROP TABLE "authorization".protocol_events;
ALTER TABLE "authorization".clients DROP COLUMN statistics_since;
