package domain

import "time"

type SubmissionType string

const (
	SubmissionTypeCode       SubmissionType = "code"
	SubmissionTypeGithubLink SubmissionType = "github_link"
)

type SubmissionStatus string

const (
	StatusPending         SubmissionStatus = "pending"
	StatusAIReviewed      SubmissionStatus = "ai_reviewed"
	StatusTeacherReviewed SubmissionStatus = "teacher_reviewed"
	StatusResubmitted     SubmissionStatus = "resubmitted"
	StatusAccepted        SubmissionStatus = "accepted"
)

type Submission struct {
	ID             int              `db:"id"`
	StudentID      int              `db:"student_id"`
	TaskID         int              `db:"task_id"`
	Code           *string          `db:"code"`
	GithubURL      *string          `db:"github_url"`
	SubmittedAt    time.Time        `db:"submitted_at"`
	Score          *float64         `db:"score"`
	Status         SubmissionStatus `db:"status"`
	SubmissionType SubmissionType   `db:"submission_type"`
}

type Task struct {
	ID          int        `db:"id"`
	CourseID    int        `db:"course_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	Deadline    time.Time  `db:"deadline"`
	MaxScore    int        `db:"max_score"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`
}

type User struct {
	ID           int        `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Role         string     `db:"role"`
	FirstName    string     `db:"first_name"`
	LastName     string     `db:"last_name"`
	CreatedAt    time.Time  `db:"created_at"`
	LastLogin    *time.Time `db:"last_login"`
}

type TaskStatus string

const (
	TaskStatusActive   TaskStatus = "active"
	TaskStatusArchived TaskStatus = "archived"
)

type Course struct {
	ID          int        `db:"id"`
	TeacherID   int        `db:"teacher_id"`
	Title       string     `db:"title"`
	Description *string    `db:"description"`
	StartDate   time.Time  `db:"start_date"`
	EndDate     *time.Time `db:"end_date"`
	IsActive    bool       `db:"is_active"`
	CreatedAt   time.Time  `db:"created_at"`
}
