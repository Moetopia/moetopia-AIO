-- +goose Up
CREATE TABLE "authorization".grants (
    id text PRIMARY KEY,
    client_id text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX grants_expires_at_idx ON "authorization".grants (expires_at);
CREATE INDEX grants_client_id_idx ON "authorization".grants (client_id);

INSERT INTO "authorization".grants (id, client_id, expires_at, created_at)
SELECT request_id, client_id,
       max(CASE WHEN kind = 'code' THEN expires_at + interval '10 minutes' ELSE expires_at END),
       min(created_at)
FROM "authorization".protocol_sessions
WHERE kind IN ('code', 'access')
GROUP BY request_id, client_id;

-- +goose Down
DROP TABLE "authorization".grants;
