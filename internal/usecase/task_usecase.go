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
	ErrCourseNotFound  = errors.New("course not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidDeadline = errors.New("deadline must be in the future")
)

type TaskUseCase interface {
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error)
}

type taskUseCase struct {
	taskRepo   repository.TaskRepository
	courseRepo repository.CourseRepository
	userRepo   repository.UserRepository
}

func NewTaskUseCase(
	taskRepo repository.TaskRepository,
	courseRepo repository.CourseRepository,
	userRepo repository.UserRepository,
) TaskUseCase {
	return &taskUseCase{
		taskRepo:   taskRepo,
		courseRepo: courseRepo,
		userRepo:   userRepo,
	}
}

type CreateTaskRequest struct {
	CourseID    int
	Title       string
	Description string
	Deadline    time.Time
	MaxScore    int
	Criteria    []TaskCriteriaRequest
}

type TaskCriteriaRequest struct {
	CriterionName        string
	CriterionDescription string
	IsMandatory          bool
	Weight               int
}

type CreateTaskResponse struct {
	TaskID    int
	CourseID  int
	Title     string
	Status    string
	CreatedAt time.Time
}

func (uc *taskUseCase) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	if err := uc.validateTaskRequest(req); err != nil {
		return nil, err
	}

	course, err := uc.courseRepo.GetByID(ctx, req.CourseID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCourseNotFound, err)
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}

	task := &domain.Task{
		CourseID:    req.CourseID,
		Title:       req.Title,
		Description: req.Description,
		Deadline:    req.Deadline,
		MaxScore:    req.MaxScore,
	}

	taskID, err := uc.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	for _, criteriaReq := range req.Criteria {
		criteria := &domain.TaskCriteria{
			TaskID:               taskID,
			CriterionName:        criteriaReq.CriterionName,
			CriterionDescription: criteriaReq.CriterionDescription,
			IsMandatory:          criteriaReq.IsMandatory,
			Weight:               criteriaReq.Weight,
		}

		_, err := uc.taskRepo.CreateCriteria(ctx, criteria)
		if err != nil {
			return nil, fmt.Errorf("failed to create task criteria: %w", err)
		}
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		CourseID:  req.CourseID,
		Title:     req.Title,
		Status:    string(domain.TaskStatusActive),
		CreatedAt: task.CreatedAt,
	}, nil
}

func (uc *taskUseCase) validateTaskRequest(req *CreateTaskRequest) error {
	var details []ValidationErrorDetail

	if req.CourseID < 1 {
		details = append(details, ValidationErrorDetail{
			Field:   "course_id",
			Message: "Must be greater than 0",
		})
	}

	if len(req.Title) < 5 || len(req.Title) > 100 {
		details = append(details, ValidationErrorDetail{
			Field:   "title",
			Message: "Must be between 5 and 100 characters",
		})
	}

	if len(req.Description) < 10 {
		details = append(details, ValidationErrorDetail{
			Field:   "description",
			Message: "Must be at least 10 characters",
		})
	}

	if req.Deadline.Before(time.Now()) {
		details = append(details, ValidationErrorDetail{
			Field:   "deadline",
			Message: "Must be in the future",
		})
	}

	if req.MaxScore < 1 || req.MaxScore > 100 {
		details = append(details, ValidationErrorDetail{
			Field:   "max_score",
			Message: "Must be between 1 and 100",
		})
	}

	if len(details) > 0 {
		return &ValidationError{
			Message: "Validation failed",
			Details: details,
		}
	}

	for i, criteria := range req.Criteria {
		if len(criteria.CriterionName) < 3 || len(criteria.CriterionName) > 100 {
			details = append(details, ValidationErrorDetail{
				Field:   fmt.Sprintf("criteria[%d].criterion_name", i),
				Message: "Must be between 3 and 100 characters",
			})
		}

		if len(criteria.CriterionDescription) < 10 {
			details = append(details, ValidationErrorDetail{
				Field:   fmt.Sprintf("criteria[%d].criterion_description", i),
				Message: "Must be at least 10 characters",
			})
		}

		if criteria.Weight < 1 || criteria.Weight > 100 {
			details = append(details, ValidationErrorDetail{
				Field:   fmt.Sprintf("criteria[%d].weight", i),
				Message: "Must be between 1 and 100",
			})
		}
	}

	if len(details) > 0 {
		return &ValidationError{
			Message: "Validation failed",
			Details: details,
		}
	}

	return nil
}
