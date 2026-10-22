-- +goose Up
CREATE TABLE identity.registration_statistics_state (
    id text PRIMARY KEY,
    started_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO identity.registration_statistics_state (id) VALUES ('registration-statistics');
CREATE TABLE identity.registration_events (
    id text PRIMARY KEY,
    day text NOT NULL CHECK (day ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$')
);
CREATE INDEX registration_events_day_idx ON identity.registration_events (day);

-- +goose Down
DROP TABLE identity.registration_events;
DROP TABLE identity.registration_statistics_state;
