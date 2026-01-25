-- +goose Up
-- +goose StatementBegin
begin;

--- Пользователи
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE CHECK (email ~* '^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+[.][A-Za-z]+$'),
  password_hash VARCHAR(128) NOT NULL,
  role VARCHAR(10) NOT NULL CHECK (role IN ('student', 'teacher', 'admin')),
  first_name VARCHAR(50) NOT NULL,
  last_name VARCHAR(50) NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  last_login TIMESTAMP
);

--- Курсы
CREATE TABLE courses (
  id SERIAL PRIMARY KEY,
  teacher_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title VARCHAR(100) NOT NULL CHECK (LENGTH(title) >= 3),
  description TEXT,
  start_date DATE NOT NULL DEFAULT NOW(),
  end_date DATE CHECK (end_date > start_date),
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW()
);

--- Зачисление учеников на курсы
CREATE TABLE course_enrollments (
  student_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  enrolled_at TIMESTAMP DEFAULT NOW(),
  completion_status VARCHAR(15) DEFAULT 'active' CHECK (
    completion_status IN ('active', 'completed', 'dropped', 'failed')
  ),
  final_score NUMERIC(5,2) CHECK (final_score BETWEEN 0 AND 100),
  PRIMARY KEY (student_id, course_id)
);

--- Задачи в рамках курса
CREATE TABLE tasks (
  id SERIAL PRIMARY KEY,
  course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title VARCHAR(100) NOT NULL,
  description TEXT NOT NULL,
  deadline TIMESTAMP NOT NULL,
  max_score INT NOT NULL CHECK (max_score > 0),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP
);

--- Сданные работы
CREATE TABLE submissions (
  id SERIAL PRIMARY KEY,
  student_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  code TEXT,
  github_url VARCHAR(255),
  submitted_at TIMESTAMP DEFAULT NOW(),
  score NUMERIC(5,2),
  status VARCHAR(15) NOT NULL DEFAULT 'pending' CHECK (
    status IN ('pending', 'ai_reviewed', 'teacher_reviewed', 'resubmitted', 'accepted')
  ),
  submission_type VARCHAR(20) NOT NULL DEFAULT 'code' CHECK (
    submission_type IN ('code', 'github_link')
  ),
  CONSTRAINT valid_submission CHECK (
    (submission_type = 'code' AND code IS NOT NULL) OR
    (submission_type = 'github_link' AND github_url IS NOT NULL)
  )
);

-- Результаты проверки ИИ
CREATE TABLE code_reviews (
  id SERIAL PRIMARY KEY,
  submission_id INT NOT NULL UNIQUE REFERENCES submissions(id) ON DELETE CASCADE,
  ai_model VARCHAR(50) NOT NULL DEFAULT 'gpt-4-turbo',
  overall_status VARCHAR(20) NOT NULL CHECK (
    overall_status IN ('passed', 'failed', 'needs_improvement')
  ),
  ai_confidence NUMERIC(3,2) CHECK (ai_confidence BETWEEN 0 AND 1),
  execution_time_ms INT CHECK (execution_time_ms > 0),
  created_at TIMESTAMP DEFAULT NOW()
);

--- Детализация замечаний ИИ
CREATE TABLE review_feedback (
  id SERIAL PRIMARY KEY,
  review_id INT NOT NULL REFERENCES code_reviews(id) ON DELETE CASCADE,
  feedback_type VARCHAR(20) NOT NULL CHECK (
    feedback_type IN (
      'critical_error', 'logic_error', 'style_issue', 
      'performance', 'security_risk', 'improvement'
    )
  ),
  line_start INT NOT NULL CHECK (line_start > 0),
  line_end INT CHECK (line_end >= line_start),
  code_snippet TEXT NOT NULL,
  suggested_fix TEXT,
  description TEXT NOT NULL,
  severity INT NOT NULL DEFAULT 3 CHECK (severity BETWEEN 1 AND 5),
  is_resolved BOOLEAN DEFAULT false,
  teacher_comment TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_courses_teacher ON courses(teacher_id);
CREATE INDEX idx_tasks_course ON tasks(course_id);
CREATE INDEX idx_submissions_task ON submissions(task_id);
CREATE INDEX idx_submissions_student ON submissions(student_id);
CREATE INDEX idx_feedback_review ON review_feedback(review_id);
CREATE INDEX idx_feedback_severity ON review_feedback(severity);

end;

-- +goose StatementEnd

-- +goose Down