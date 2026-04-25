-- +goose Up
-- +goose StatementBegin
ALTER TABLE submissions ALTER COLUMN status TYPE VARCHAR(50);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE submissions SET status = 'pending' WHERE status = 'teacher_reviewed';
ALTER TABLE submissions ALTER COLUMN status TYPE VARCHAR(15);
-- +goose StatementEnd
