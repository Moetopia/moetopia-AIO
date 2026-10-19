-- +goose Up
CREATE SCHEMA developer;
CREATE TABLE developer.applications (
    id text PRIMARY KEY, owner_id text NOT NULL, client_id text NOT NULL UNIQUE,
    public boolean NOT NULL, enabled boolean NOT NULL DEFAULT true, deleted boolean NOT NULL DEFAULT false,
    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    active_version integer NOT NULL DEFAULT 0, latest_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX applications_owner_created ON developer.applications(owner_id,created_at);
CREATE TABLE developer.application_versions (
    id text PRIMARY KEY, application_id text NOT NULL, version integer NOT NULL CHECK (version > 0),
    config jsonb NOT NULL, status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','pending','approved','rejected')),
    review_reason text NOT NULL DEFAULT '', reviewer_id text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(application_id,version)
);
CREATE INDEX application_versions_status_created ON developer.application_versions(status,created_at);
ALTER TABLE "authorization".clients ADD COLUMN owner_id text NOT NULL DEFAULT '', ADD COLUMN test_only boolean NOT NULL DEFAULT false;

-- +goose Down
DELETE FROM "authorization".clients WHERE id IN (SELECT client_id FROM developer.applications);
ALTER TABLE "authorization".clients DROP COLUMN owner_id, DROP COLUMN test_only;
DROP TABLE developer.application_versions;
DROP TABLE developer.applications;
DROP SCHEMA developer;
