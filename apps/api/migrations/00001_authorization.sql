-- +goose Up
-- Establish the protocol storage boundary before the API accepts credentials.
CREATE SCHEMA "authorization";

CREATE TABLE "authorization".clients (
    id text PRIMARY KEY,
    secret_hash bytea,
    name text NOT NULL,
    public boolean NOT NULL DEFAULT false,
    enabled boolean NOT NULL DEFAULT false,
    redirect_uris jsonb NOT NULL DEFAULT '[]',
    grant_types jsonb NOT NULL DEFAULT '[]',
    response_types jsonb NOT NULL DEFAULT '[]',
    scopes jsonb NOT NULL DEFAULT '[]',
    audiences jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE "authorization".protocol_sessions (
    id text PRIMARY KEY,
    kind text NOT NULL,
    request_id text NOT NULL,
    client_id text NOT NULL,
    payload bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX protocol_sessions_request_kind ON "authorization".protocol_sessions(request_id, kind);
CREATE INDEX protocol_sessions_expiry ON "authorization".protocol_sessions(expires_at);

CREATE TABLE "authorization".signing_keys (
    id text PRIMARY KEY,
    algorithm text NOT NULL DEFAULT 'RS256',
    private_key bytea NOT NULL,
    public_key bytea NOT NULL,
    active boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    retire_after timestamptz
);
CREATE UNIQUE INDEX signing_keys_one_active ON "authorization".signing_keys(active) WHERE active;

-- +goose Down
-- Only use on a disposable database: rollback permanently removes protocol state.
DROP TABLE "authorization".protocol_sessions;
DROP TABLE "authorization".signing_keys;
DROP TABLE "authorization".clients;
DROP SCHEMA "authorization";
