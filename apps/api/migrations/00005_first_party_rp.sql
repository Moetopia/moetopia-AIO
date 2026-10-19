-- +goose Up
ALTER TABLE sessions.web_sessions DROP CONSTRAINT web_sessions_application_check;
ALTER TABLE sessions.web_sessions ADD CONSTRAINT web_sessions_application_check
    CHECK (application IN ('account', 'admin', 'developer', 'user-site'));
CREATE TABLE sessions.rp_logins (
    id text PRIMARY KEY,
    browser_hash text NOT NULL,
    payload bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX rp_logins_expires_at ON sessions.rp_logins (expires_at);

-- +goose Down
DROP TABLE sessions.rp_logins;
DELETE FROM sessions.web_sessions WHERE application = 'user-site';
ALTER TABLE sessions.web_sessions DROP CONSTRAINT web_sessions_application_check;
ALTER TABLE sessions.web_sessions ADD CONSTRAINT web_sessions_application_check
    CHECK (application IN ('account', 'admin', 'developer'));
