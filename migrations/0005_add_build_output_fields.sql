-- +goose Up
-- +goose StatementBegin
ALTER TABLE build_results ADD COLUMN build_output TEXT NOT NULL DEFAULT '';
ALTER TABLE build_results ADD COLUMN build_success BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE build_results DROP COLUMN IF EXISTS build_output;
ALTER TABLE build_results DROP COLUMN IF EXISTS build_success;
-- +goose StatementEnd
