-- +goose Up
ALTER TABLE identity.users ADD COLUMN deletion_requested_at timestamptz,
    ADD COLUMN deletion_due_at timestamptz,
    ADD CONSTRAINT users_deletion_period CHECK (
        (deletion_requested_at IS NULL AND deletion_due_at IS NULL) OR
        (deletion_requested_at IS NOT NULL AND deletion_due_at IS NOT NULL AND deletion_due_at > deletion_requested_at)
    );
CREATE INDEX users_deletion_due ON identity.users(deletion_due_at);
ALTER TABLE identity.account_challenges DROP CONSTRAINT account_challenges_purpose_check;
ALTER TABLE identity.account_challenges ADD CONSTRAINT account_challenges_purpose_check CHECK (purpose IN ('reset','email','deletion-cancel'));
ALTER TABLE notifications.email_deliveries ADD COLUMN user_id text NOT NULL DEFAULT '';
CREATE INDEX email_deliveries_user ON notifications.email_deliveries(user_id);
CREATE INDEX clients_owner ON "authorization".clients(owner_id);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM identity.users WHERE deletion_due_at IS NOT NULL) THEN
        RAISE EXCEPTION 'Pending account deletions prevent downgrade';
    END IF;
END $$;
-- +goose StatementEnd
DROP INDEX "authorization".clients_owner;
DELETE FROM identity.account_challenges WHERE purpose = 'deletion-cancel';
ALTER TABLE identity.account_challenges DROP CONSTRAINT account_challenges_purpose_check;
ALTER TABLE identity.account_challenges ADD CONSTRAINT account_challenges_purpose_check CHECK (purpose IN ('reset','email'));
DROP INDEX notifications.email_deliveries_user;
ALTER TABLE notifications.email_deliveries DROP COLUMN user_id;
ALTER TABLE identity.users DROP CONSTRAINT users_deletion_period;
DROP INDEX identity.users_deletion_due;
ALTER TABLE identity.users DROP COLUMN deletion_requested_at, DROP COLUMN deletion_due_at;
