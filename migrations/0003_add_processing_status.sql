-- +goose Up
-- +goose StatementBegin
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_status_check CHECK (
    status IN ('pending', 'processing', 'ai_reviewed', 'teacher_reviewed', 'resubmitted', 'accepted')
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE submissions SET status = 'pending' WHERE status = 'processing';
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_status_check CHECK (
    status IN ('pending', 'ai_reviewed', 'teacher_reviewed', 'resubmitted', 'accepted')
);
-- +goose StatementEnd
