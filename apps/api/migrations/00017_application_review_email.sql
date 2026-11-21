-- +goose Up
ALTER TABLE identity.users ADD COLUMN mute_application_review_email boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE identity.users DROP COLUMN mute_application_review_email;
