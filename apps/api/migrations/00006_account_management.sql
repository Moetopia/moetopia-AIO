-- +goose Up
ALTER TABLE identity.users ADD COLUMN bio text NOT NULL DEFAULT '',
    ADD COLUMN avatar bytea,
    ADD COLUMN avatar_digest text NOT NULL DEFAULT '',
    ADD COLUMN revision integer NOT NULL DEFAULT 0 CHECK (revision >= 0),
    ADD COLUMN security_version integer NOT NULL DEFAULT 0 CHECK (security_version >= 0);
ALTER TABLE "authorization".grants ADD COLUMN user_id text NOT NULL DEFAULT '',
    ADD COLUMN security_version integer NOT NULL DEFAULT 0,
    ADD COLUMN scopes jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN audiences jsonb NOT NULL DEFAULT '[]';
CREATE INDEX grants_user_created ON "authorization".grants(user_id, created_at);
CREATE TABLE identity.account_challenges (
    id text PRIMARY KEY,
    purpose text NOT NULL CHECK (purpose IN ('reset','email')),
    user_id text NOT NULL,
    email text NOT NULL,
    binding text NOT NULL,
    security_version integer NOT NULL,
    code_hash bytea NOT NULL,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
    consumed boolean NOT NULL DEFAULT false,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX account_challenges_user ON identity.account_challenges(user_id);
CREATE INDEX account_challenges_expiry ON identity.account_challenges(expires_at);
CREATE SCHEMA audit;
CREATE TABLE audit.security_events (
    id text PRIMARY KEY, user_id text NOT NULL, action text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX security_events_user_created ON audit.security_events(user_id, created_at);
CREATE INDEX security_events_created ON audit.security_events(created_at);

-- +goose Down
DROP TABLE audit.security_events;
DROP SCHEMA audit;
DROP TABLE identity.account_challenges;
DROP INDEX "authorization".grants_user_created;
ALTER TABLE "authorization".grants DROP COLUMN user_id, DROP COLUMN security_version, DROP COLUMN scopes, DROP COLUMN audiences;
ALTER TABLE identity.users DROP COLUMN bio, DROP COLUMN avatar, DROP COLUMN avatar_digest, DROP COLUMN revision, DROP COLUMN security_version;
