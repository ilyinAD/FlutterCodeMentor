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
	StatusProcessing      SubmissionStatus = "processing"
	StatusAIReviewed      SubmissionStatus = "ai_reviewed"
	StatusTeacherReviewed SubmissionStatus = "teacher_reviewed"
	StatusResubmitted     SubmissionStatus = "resubmitted"
	StatusAccepted        SubmissionStatus = "accepted"
)

type Submission struct {
	ID             int
	StudentID      int
	TaskID         int
	Code           *string
	GithubURL      *string
	SubmittedAt    time.Time
	Score          *float64
	Status         SubmissionStatus
	SubmissionType SubmissionType
}

type Task struct {
	ID          int
	CourseID    int
	Title       string
	Description string
	Deadline    time.Time
	MaxScore    int
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

type User struct {
	ID           int
	Email        string
	PasswordHash string
	Role         string
	FirstName    string
	LastName     string
	CreatedAt    time.Time
	LastLogin    *time.Time
}

type TaskStatus string

const (
	TaskStatusActive   TaskStatus = "active"
	TaskStatusArchived TaskStatus = "archived"
)

type Course struct {
	ID          int
	TeacherID   int
	Title       string
	Description *string
	StartDate   time.Time
	EndDate     *time.Time
	IsActive    bool
	CreatedAt   time.Time
}

type CodeReview struct {
	ID              int
	SubmissionID    int
	AIModel         string
	OverallStatus   string
	AIConfidence    *float64
	ExecutionTimeMs *int
	CreatedAt       time.Time
}

type ReviewFeedback struct {
	ID              int
	ReviewID        int
	FeedbackType    string
	FilePath        *string
	LineStart       int
	LineEnd         *int
	CodeSnippet     string
	SuggestedFix    *string
	Description     string
	Severity        int
	IsResolved      bool
	TeacherComment  *string
	TeacherApproved *bool
	CreatedAt       time.Time
}

type TaskCriteria struct {
	ID                   int
	TaskID               int
	CriterionName        string
	CriterionDescription string
	IsMandatory          bool
	Weight               int
	CreatedAt            time.Time
}

type BuildResult struct {
	ID              int
	SubmissionID    int
	CompileSuccess  bool
	AnalyzeOutput   string
	TestOutput      string
	TestsPassed     bool
	FormatOutput    string
	FormatCorrect   bool
	ExecutionTimeMs int
	CreatedAt       time.Time
}
