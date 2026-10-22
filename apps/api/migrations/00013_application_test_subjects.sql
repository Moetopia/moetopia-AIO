-- +goose Up
CREATE TABLE "authorization".client_test_subjects (
    id text PRIMARY KEY,
    client_id text NOT NULL REFERENCES "authorization".clients(id) ON DELETE CASCADE,
    user_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, user_id)
);
CREATE INDEX client_test_subjects_user_idx ON "authorization".client_test_subjects (user_id);

-- +goose Down
DROP TABLE "authorization".client_test_subjects;
