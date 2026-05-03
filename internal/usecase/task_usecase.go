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
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*TaskDetailResponse, error)
	GetTaskByID(ctx context.Context, id int) (*TaskDetailResponse, error)
	GetTasksByCourseID(ctx context.Context, courseID int) ([]*TaskDetailResponse, error)
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

type TaskDetailResponse struct {
	TaskID      int
	CourseID    int
	Title       string
	Description string
	Deadline    time.Time
	MaxScore    int
	Status      string
	CreatedAt   time.Time
	Criteria    []TaskCriteriaDetail
}

type TaskCriteriaDetail struct {
	ID                   int
	CriterionName        string
	CriterionDescription string
	IsMandatory          bool
	Weight               int
}

func (uc *taskUseCase) CreateTask(ctx context.Context, req *CreateTaskRequest) (*TaskDetailResponse, error) {
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

	criteria := make([]TaskCriteriaDetail, 0, len(req.Criteria))
	for _, criteriaReq := range req.Criteria {
		c := &domain.TaskCriteria{
			TaskID:               taskID,
			CriterionName:        criteriaReq.CriterionName,
			CriterionDescription: criteriaReq.CriterionDescription,
			IsMandatory:          criteriaReq.IsMandatory,
			Weight:               criteriaReq.Weight,
		}

		criteriaID, err := uc.taskRepo.CreateCriteria(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to create task criteria: %w", err)
		}

		criteria = append(criteria, TaskCriteriaDetail{
			ID:                   criteriaID,
			CriterionName:        c.CriterionName,
			CriterionDescription: c.CriterionDescription,
			IsMandatory:          c.IsMandatory,
			Weight:               c.Weight,
		})
	}

	return &TaskDetailResponse{
		TaskID:      taskID,
		CourseID:    req.CourseID,
		Title:       req.Title,
		Description: req.Description,
		Deadline:    req.Deadline,
		MaxScore:    req.MaxScore,
		Status:      string(domain.TaskStatusActive),
		CreatedAt:   task.CreatedAt,
		Criteria:    criteria,
	}, nil
}

func (uc *taskUseCase) GetTaskByID(ctx context.Context, id int) (*TaskDetailResponse, error) {
	task, err := uc.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}

	criteria, err := uc.taskRepo.GetCriteriaByTaskID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task criteria: %w", err)
	}

	return buildTaskDetail(task, criteria), nil
}

func (uc *taskUseCase) GetTasksByCourseID(ctx context.Context, courseID int) ([]*TaskDetailResponse, error) {
	course, err := uc.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}

	tasks, err := uc.taskRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	result := make([]*TaskDetailResponse, 0, len(tasks))
	for _, t := range tasks {
		criteria, err := uc.taskRepo.GetCriteriaByTaskID(ctx, t.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task criteria: %w", err)
		}
		result = append(result, buildTaskDetail(t, criteria))
	}
	return result, nil
}

func buildTaskDetail(task *domain.Task, criteria []*domain.TaskCriteria) *TaskDetailResponse {
	details := make([]TaskCriteriaDetail, 0, len(criteria))
	for _, c := range criteria {
		details = append(details, TaskCriteriaDetail{
			ID:                   c.ID,
			CriterionName:        c.CriterionName,
			CriterionDescription: c.CriterionDescription,
			IsMandatory:          c.IsMandatory,
			Weight:               c.Weight,
		})
	}

	status := string(domain.TaskStatusActive)
	if task.Deadline.Before(time.Now()) {
		status = string(domain.TaskStatusArchived)
	}

	return &TaskDetailResponse{
		TaskID:      task.ID,
		CourseID:    task.CourseID,
		Title:       task.Title,
		Description: task.Description,
		Deadline:    task.Deadline,
		MaxScore:    task.MaxScore,
		Status:      status,
		CreatedAt:   task.CreatedAt,
		Criteria:    details,
	}
}
