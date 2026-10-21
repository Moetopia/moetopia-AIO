-- +goose Up
ALTER TABLE identity.users ADD COLUMN has_password boolean NOT NULL DEFAULT true;

CREATE TABLE identity.external_identities (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('github', 'google')),
    issuer text NOT NULL,
    subject text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, issuer, subject),
    UNIQUE (user_id, provider)
);

CREATE TABLE identity.external_logins (
    id text PRIMARY KEY,
    browser_hash text NOT NULL,
    provider text NOT NULL CHECK (provider IN ('github', 'google')),
    purpose text NOT NULL CHECK (purpose IN ('login', 'bind', 'register')),
    user_id text NOT NULL DEFAULT '',
    payload bytea NOT NULL,
    consumed boolean NOT NULL DEFAULT false,
    expires_at timestamptz NOT NULL
);
CREATE INDEX external_logins_expiry_idx ON identity.external_logins (expires_at);
CREATE INDEX external_logins_user_idx ON identity.external_logins (user_id);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM identity.users WHERE NOT has_password) THEN
        RAISE EXCEPTION 'Cannot remove external login support while passwordless accounts exist';
    END IF;
END $$;
-- +goose StatementEnd
DROP TABLE identity.external_logins;
DROP TABLE identity.external_identities;
ALTER TABLE identity.users DROP COLUMN has_password;
