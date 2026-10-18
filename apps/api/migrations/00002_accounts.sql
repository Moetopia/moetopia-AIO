-- +goose Up
CREATE SCHEMA identity;
CREATE SCHEMA sessions;
CREATE SCHEMA notifications;

CREATE TABLE identity.users (
    id text PRIMARY KEY,
    email text NOT NULL UNIQUE,
    display_name text NOT NULL,
    password_hash text NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    email_verified_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE identity.invitations (
    id text PRIMARY KEY,
    expires_at timestamptz NOT NULL,
    consumed boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE identity.email_verifications (
    id text PRIMARY KEY,
    email text NOT NULL,
    code_hash bytea NOT NULL,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    consumed boolean NOT NULL DEFAULT false,
    expires_at timestamptz NOT NULL
);
CREATE INDEX email_verifications_expiry ON identity.email_verifications(expires_at);
CREATE TABLE identity.registration_policy (
    id text PRIMARY KEY CHECK (id = 'registration'),
    mode text NOT NULL DEFAULT 'invite' CHECK (mode IN ('invite', 'open'))
);
INSERT INTO identity.registration_policy(id, mode) VALUES ('registration', 'invite');

CREATE TABLE sessions.web_sessions (
    id text PRIMARY KEY,
    user_id text NOT NULL,
    application text NOT NULL CHECK (application IN ('account', 'admin', 'developer')),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX web_sessions_user ON sessions.web_sessions(user_id);
CREATE INDEX web_sessions_expiry ON sessions.web_sessions(expires_at);

CREATE TABLE notifications.email_deliveries (
    id text PRIMARY KEY,
    payload bytea NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    available_at timestamptz NOT NULL,
    lease_until timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX email_deliveries_ready ON notifications.email_deliveries(available_at, lease_until);

-- +goose Down
-- Destructive rollback is only suitable for disposable test data.
DROP TABLE notifications.email_deliveries;
DROP TABLE sessions.web_sessions;
DROP TABLE identity.registration_policy;
DROP TABLE identity.email_verifications;
DROP TABLE identity.invitations;
DROP TABLE identity.users;
DROP SCHEMA notifications;
DROP SCHEMA sessions;
DROP SCHEMA identity;
