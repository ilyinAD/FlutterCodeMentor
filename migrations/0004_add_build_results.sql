-- +goose Up
-- +goose StatementBegin
CREATE TABLE build_results (
    id SERIAL PRIMARY KEY,
    submission_id INT NOT NULL UNIQUE REFERENCES submissions(id) ON DELETE CASCADE,
    compile_success BOOLEAN NOT NULL DEFAULT false,
    analyze_output TEXT NOT NULL DEFAULT '',
    test_output TEXT NOT NULL DEFAULT '',
    tests_passed BOOLEAN NOT NULL DEFAULT false,
    format_output TEXT NOT NULL DEFAULT '',
    format_correct BOOLEAN NOT NULL DEFAULT false,
    execution_time_ms INT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_build_results_submission ON build_results(submission_id);

ALTER TABLE review_feedback DROP CONSTRAINT IF EXISTS review_feedback_feedback_type_check;
ALTER TABLE review_feedback ADD CONSTRAINT review_feedback_feedback_type_check CHECK (
    feedback_type IN (
        'critical_error', 'logic_error', 'style_issue',
        'performance', 'security_risk', 'improvement', 'criterion_check'
    )
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS build_results;

ALTER TABLE review_feedback DROP CONSTRAINT IF EXISTS review_feedback_feedback_type_check;
ALTER TABLE review_feedback ADD CONSTRAINT review_feedback_feedback_type_check CHECK (
    feedback_type IN (
        'critical_error', 'logic_error', 'style_issue',
        'performance', 'security_risk', 'improvement'
    )
);
-- +goose StatementEnd
