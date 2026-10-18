-- +goose Up
CREATE TABLE "authorization".interactions (
    id text PRIMARY KEY,
    client_id text NOT NULL,
    browser_hash text NOT NULL,
    payload bytea NOT NULL,
    consumed boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX authorization_interactions_expiry ON "authorization".interactions(expires_at);

-- +goose Down
DROP TABLE "authorization".interactions;
