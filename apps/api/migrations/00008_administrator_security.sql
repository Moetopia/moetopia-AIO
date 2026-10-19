-- +goose Up
CREATE SCHEMA administration;
CREATE TABLE administration.principals (
    id text PRIMARY KEY, enabled boolean NOT NULL DEFAULT true,
    secret bytea, pending_secret bytea, pending_session text NOT NULL DEFAULT '', pending_expires_at timestamptz,
    last_counter bigint NOT NULL DEFAULT -1, factor_version integer NOT NULL DEFAULT 0 CHECK (factor_version >= 0),
    recovery_hashes jsonb NOT NULL DEFAULT '[]', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE administration.elevations (
    id text PRIMARY KEY, user_id text NOT NULL, factor_version integer NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX elevations_expires ON administration.elevations(expires_at);
CREATE INDEX elevations_user ON administration.elevations(user_id);
CREATE TABLE administration.bootstrap (id text PRIMARY KEY, user_id text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
ALTER TABLE developer.applications ADD COLUMN suspended boolean NOT NULL DEFAULT false, ADD COLUMN suspension_reason text NOT NULL DEFAULT '';
ALTER TABLE audit.security_events ADD COLUMN target_id text NOT NULL DEFAULT '', ADD COLUMN request_id text NOT NULL DEFAULT '', ADD COLUMN outcome text NOT NULL DEFAULT 'success' CHECK (outcome IN ('success','failure'));

-- +goose Down
DELETE FROM sessions.web_sessions WHERE application = 'admin';
DELETE FROM "authorization".clients WHERE id = 'moetopia-admin';
ALTER TABLE audit.security_events DROP COLUMN target_id, DROP COLUMN request_id, DROP COLUMN outcome;
ALTER TABLE developer.applications DROP COLUMN suspended, DROP COLUMN suspension_reason;
DROP TABLE administration.elevations;
DROP TABLE administration.principals;
DROP TABLE administration.bootstrap;
DROP SCHEMA administration;
