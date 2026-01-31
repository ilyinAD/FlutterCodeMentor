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

type CodeReview struct {
	ID              int       `db:"id"`
	SubmissionID    int       `db:"submission_id"`
	AIModel         string    `db:"ai_model"`
	OverallStatus   string    `db:"overall_status"`
	AIConfidence    *float64  `db:"ai_confidence"`
	ExecutionTimeMs *int      `db:"execution_time_ms"`
	CreatedAt       time.Time `db:"created_at"`
}

type ReviewFeedback struct {
	ID              int       `db:"id"`
	ReviewID        int       `db:"review_id"`
	FeedbackType    string    `db:"feedback_type"`
	FilePath        *string   `db:"file_path"`
	LineStart       int       `db:"line_start"`
	LineEnd         *int      `db:"line_end"`
	CodeSnippet     string    `db:"code_snippet"`
	SuggestedFix    *string   `db:"suggested_fix"`
	Description     string    `db:"description"`
	Severity        int       `db:"severity"`
	IsResolved      bool      `db:"is_resolved"`
	TeacherComment  *string   `db:"teacher_comment"`
	TeacherApproved *bool     `db:"teacher_approved"`
	CreatedAt       time.Time `db:"created_at"`
}

type TaskCriteria struct {
	ID                   int       `db:"id"`
	TaskID               int       `db:"task_id"`
	CriterionName        string    `db:"criterion_name"`
	CriterionDescription string    `db:"criterion_description"`
	IsMandatory          bool      `db:"is_mandatory"`
	Weight               int       `db:"weight"`
	CreatedAt            time.Time `db:"created_at"`
}
