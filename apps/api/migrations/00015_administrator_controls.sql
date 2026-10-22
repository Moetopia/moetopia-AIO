-- +goose Up
CREATE TABLE administration.controls (id text PRIMARY KEY);
INSERT INTO administration.controls (id) VALUES ('administrator-security');
ALTER TABLE administration.principals
    ADD COLUMN enrollment_blocked boolean NOT NULL DEFAULT false,
    ADD COLUMN enrollment_session text NOT NULL DEFAULT '',
    ADD COLUMN emergency_hash text NOT NULL DEFAULT '',
    ADD COLUMN emergency_expires_at timestamptz;
ALTER TABLE audit.security_events ADD COLUMN reason text NOT NULL DEFAULT '';

-- +goose Down
-- Blocked recovery must not become unrestricted enrollment in an older binary.
UPDATE administration.principals SET enabled = false WHERE enrollment_blocked OR enrollment_session <> '';
ALTER TABLE administration.principals DROP COLUMN emergency_expires_at, DROP COLUMN emergency_hash, DROP COLUMN enrollment_session, DROP COLUMN enrollment_blocked;
DROP TABLE administration.controls;
ALTER TABLE audit.security_events DROP COLUMN reason;
