package usecase

import (
	"context"
	"errors"
	"fmt"
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
	ErrSubmissionNotFound    = errors.New("submission not found")
	ErrReviewNotFound        = errors.New("review not found")
)

type SubmissionUseCase interface {
	CreateSubmission(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error)
	GetSubmissionByID(ctx context.Context, id int) (*SubmissionDetail, error)
	GetSubmissionsByTaskID(ctx context.Context, taskID int, studentID *int) ([]*SubmissionDetail, error)
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

type SubmissionDetail struct {
	SubmissionID   int
	TaskID         int
	UserID         int
	StudentName    string
	SubmissionType string
	Code           *string
	GithubURL      *string
	Status         string
	Score          *float64
	CreatedAt      time.Time
}

func (uc *submissionUseCase) CreateSubmission(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error) {
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

func (uc *submissionUseCase) GetSubmissionByID(ctx context.Context, id int) (*SubmissionDetail, error) {
	submission, err := uc.submissionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}
	if submission == nil {
		return nil, ErrSubmissionNotFound
	}

	user, err := uc.userRepo.GetByID(ctx, submission.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	return submissionToDetail(submission, user), nil
}

func (uc *submissionUseCase) GetSubmissionsByTaskID(ctx context.Context, taskID int, studentID *int) ([]*SubmissionDetail, error) {
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}

	var submissions []*domain.Submission
	if studentID != nil {
		submissions, err = uc.submissionRepo.GetByTaskAndStudent(ctx, taskID, *studentID)
	} else {
		submissions, err = uc.submissionRepo.GetByTaskID(ctx, taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}

	result := make([]*SubmissionDetail, 0, len(submissions))
	for _, s := range submissions {
		user, err := uc.userRepo.GetByID(ctx, s.StudentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get student: %w", err)
		}
		result = append(result, submissionToDetail(s, user))
	}
	return result, nil
}

func submissionToDetail(s *domain.Submission, user *domain.User) *SubmissionDetail {
	studentName := ""
	if user != nil {
		studentName = user.FirstName + " " + user.LastName
	}
	return &SubmissionDetail{
		SubmissionID:   s.ID,
		TaskID:         s.TaskID,
		UserID:         s.StudentID,
		StudentName:    studentName,
		SubmissionType: string(s.SubmissionType),
		Code:           s.Code,
		GithubURL:      s.GithubURL,
		Status:         string(s.Status),
		Score:          s.Score,
		CreatedAt:      s.SubmittedAt,
	}
}
