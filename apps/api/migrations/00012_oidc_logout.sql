-- +goose Up
ALTER TABLE "authorization".clients ADD COLUMN post_logout_redirect_uris jsonb NOT NULL DEFAULT '[]';

UPDATE "authorization".clients
SET post_logout_redirect_uris = jsonb_build_array(regexp_replace(redirect_uris->>0, '/api/v1/portal/callback$', '/api/v1/portal/logout/callback'))
WHERE id = 'moetopia-user-site' AND owner_id = '' AND NOT public
  AND jsonb_array_length(redirect_uris) = 1 AND redirect_uris->>0 LIKE '%/api/v1/portal/callback';
UPDATE "authorization".clients
SET post_logout_redirect_uris = jsonb_build_array(regexp_replace(redirect_uris->>0, '/api/v1/admin/callback$', '/api/v1/admin/logout/callback'))
WHERE id = 'moetopia-admin' AND owner_id = '' AND NOT public
  AND jsonb_array_length(redirect_uris) = 1 AND redirect_uris->>0 LIKE '%/api/v1/admin/callback';

CREATE TABLE "authorization".logout_interactions (
    id text PRIMARY KEY,
    client_id text NOT NULL,
    browser_hash text NOT NULL,
    payload bytea NOT NULL,
    consumed boolean NOT NULL DEFAULT false,
    expires_at timestamptz NOT NULL
);
CREATE INDEX logout_interactions_expiry_idx ON "authorization".logout_interactions (expires_at);

-- +goose Down
DROP TABLE "authorization".logout_interactions;
ALTER TABLE "authorization".clients DROP COLUMN post_logout_redirect_uris;
