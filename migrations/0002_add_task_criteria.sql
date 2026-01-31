-- +goose Up
-- +goose StatementBegin
begin;

CREATE TABLE task_criteria (
  id SERIAL PRIMARY KEY,
  task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  criterion_name VARCHAR(100) NOT NULL,
  criterion_description TEXT NOT NULL,
  is_mandatory BOOLEAN DEFAULT true,
  weight INT DEFAULT 10 CHECK (weight BETWEEN 1 AND 100),
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_task_criteria_task ON task_criteria(task_id);

end;

-- +goose StatementEnd

-- +goose Down