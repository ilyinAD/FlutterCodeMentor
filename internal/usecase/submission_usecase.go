package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/ilyin-ad/flutter-code-mentor/internal/repository"
)

var (
	ErrInvalidSubmissionType = errors.New("invalid submission type")
	ErrMissingCode           = errors.New("code is required when submission_type is 'code'")
	ErrMissingGithubURL      = errors.New("github_url is required when submission_type is 'github_link'")
	ErrInvalidGithubURL      = errors.New("invalid github URL format")
	ErrTaskNotFound          = errors.New("task not found")
	ErrUserNotFound          = errors.New("user not found")
)

type SubmissionUseCase interface {
	CreateSubmission(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error)
}

type submissionUseCase struct {
	submissionRepo repository.SubmissionRepository
	taskRepo       repository.TaskRepository
	userRepo       repository.UserRepository
}

func NewSubmissionUseCase(
	submissionRepo repository.SubmissionRepository,
	taskRepo repository.TaskRepository,
	userRepo repository.UserRepository,
) SubmissionUseCase {
	return &submissionUseCase{
		submissionRepo: submissionRepo,
		taskRepo:       taskRepo,
		userRepo:       userRepo,
	}
}

type CreateSubmissionRequest struct {
	TaskID         int
	UserID         int
	SubmissionType string
	Code           *string
	GithubURL      *string
}

type CreateSubmissionResponse struct {
	SubmissionID int
	CreatedAt    time.Time
}

type ValidationErrorDetail struct {
	Field   string
	Message string
}

type ValidationError struct {
	Message string
	Details []ValidationErrorDetail
}

func (e *ValidationError) Error() string {
	return e.Message
}

func (uc *submissionUseCase) CreateSubmission(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error) {
	if err := uc.validateSubmissionRequest(req); err != nil {
		return nil, err
	}

	task, err := uc.taskRepo.GetByID(ctx, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}

	user, err := uc.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserNotFound, err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	submission := &domain.Submission{
		StudentID:      req.UserID,
		TaskID:         req.TaskID,
		Code:           req.Code,
		GithubURL:      req.GithubURL,
		Status:         domain.StatusPending,
		SubmissionType: domain.SubmissionType(req.SubmissionType),
	}

	submissionID, err := uc.submissionRepo.Create(ctx, submission)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	return &CreateSubmissionResponse{
		SubmissionID: submissionID,
		CreatedAt:    submission.SubmittedAt,
	}, nil
}

func (uc *submissionUseCase) validateSubmissionRequest(req *CreateSubmissionRequest) error {
	var details []ValidationErrorDetail

	if req.SubmissionType != string(domain.SubmissionTypeCode) && req.SubmissionType != string(domain.SubmissionTypeGithubLink) {
		details = append(details, ValidationErrorDetail{
			Field:   "submission_type",
			Message: "Must be either 'code' or 'github_link'",
		})
	}

	if req.SubmissionType == string(domain.SubmissionTypeCode) {
		if req.Code == nil || *req.Code == "" {
			details = append(details, ValidationErrorDetail{
				Field:   "code",
				Message: "Required when submission_type is 'code'",
			})
		}
		if req.GithubURL != nil && *req.GithubURL != "" {
			details = append(details, ValidationErrorDetail{
				Field:   "github_url",
				Message: "Should not be provided when submission_type is 'code'",
			})
		}
	}

	if req.SubmissionType == string(domain.SubmissionTypeGithubLink) {
		if req.GithubURL == nil || *req.GithubURL == "" {
			details = append(details, ValidationErrorDetail{
				Field:   "github_url",
				Message: "Required when submission_type is 'github_link'",
			})
		} else {
			githubURLPattern := `^https://github\.com/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)/?$`
			matched, _ := regexp.MatchString(githubURLPattern, *req.GithubURL)
			if !matched {
				details = append(details, ValidationErrorDetail{
					Field:   "github_url",
					Message: "Invalid GitHub URL format. Expected: https://github.com/username/repository",
				})
			}
		}
		if req.Code != nil && *req.Code != "" {
			details = append(details, ValidationErrorDetail{
				Field:   "code",
				Message: "Should not be provided when submission_type is 'github_link'",
			})
		}
	}

	if req.TaskID < 1 {
		details = append(details, ValidationErrorDetail{
			Field:   "task_id",
			Message: "Must be greater than 0",
		})
	}

	if req.UserID < 1 {
		details = append(details, ValidationErrorDetail{
			Field:   "user_id",
			Message: "Must be greater than 0",
		})
	}

	if len(details) > 0 {
		return &ValidationError{
			Message: "Validation failed",
			Details: details,
		}
	}

	return nil
}
