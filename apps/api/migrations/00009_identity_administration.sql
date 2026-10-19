-- +goose Up
ALTER TABLE identity.users ADD COLUMN disabled_reason text NOT NULL DEFAULT '';
ALTER TABLE identity.invitations ADD COLUMN revoked boolean NOT NULL DEFAULT false;
ALTER TABLE identity.registration_policy ADD COLUMN revision integer NOT NULL DEFAULT 0 CHECK (revision >= 0);

-- +goose Down
ALTER TABLE identity.registration_policy DROP COLUMN revision;
UPDATE identity.invitations SET consumed = true WHERE revoked;
ALTER TABLE identity.invitations DROP COLUMN revoked;
ALTER TABLE identity.users DROP COLUMN disabled_reason;
