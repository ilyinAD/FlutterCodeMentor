-- +goose Up
-- +goose StatementBegin
ALTER TABLE build_results ADD COLUMN IF NOT EXISTS build_success BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE build_results ADD COLUMN IF NOT EXISTS build_output TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE build_results DROP COLUMN IF EXISTS build_output;
ALTER TABLE build_results DROP COLUMN IF EXISTS build_success;
-- +goose StatementEnd
