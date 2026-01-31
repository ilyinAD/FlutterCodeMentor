package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/api"
	"github.com/ilyin-ad/flutter-code-mentor/internal/usecase"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type TaskHandler struct {
	taskUseCase usecase.TaskUseCase
	logger      *zap.Logger
}

func NewTaskHandler(taskUseCase usecase.TaskUseCase, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		taskUseCase: taskUseCase,
		logger:      logger,
	}
}

type CreateTaskRequest struct {
	CourseID    int                         `json:"course_id" validate:"required,min=1"`
	Title       string                      `json:"title" validate:"required,min=5,max=100"`
	Description string                      `json:"description" validate:"required,min=10"`
	Deadline    time.Time                   `json:"deadline" validate:"required"`
	MaxScore    int                         `json:"max_score" validate:"required,min=1,max=100"`
	Criteria    []CreateTaskCriteriaRequest `json:"criteria,omitempty"`
}

type CreateTaskCriteriaRequest struct {
	CriterionName        string `json:"criterion_name" validate:"required,min=3,max=100"`
	CriterionDescription string `json:"criterion_description" validate:"required,min=10"`
	IsMandatory          bool   `json:"is_mandatory"`
	Weight               int    `json:"weight" validate:"required,min=1,max=100"`
}

func (h *TaskHandler) PostTask(ctx echo.Context) error {
	h.logger.Info("Received task creation request",
		zap.String("method", ctx.Request().Method),
		zap.String("path", ctx.Request().URL.Path),
	)

	var req CreateTaskRequest
	if err := ctx.Bind(&req); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	h.logger.Info("Creating task",
		zap.Int("course_id", req.CourseID),
		zap.String("title", req.Title),
		zap.Int("max_score", req.MaxScore),
	)

	criteria := make([]usecase.TaskCriteriaRequest, len(req.Criteria))
	for i, c := range req.Criteria {
		criteria[i] = usecase.TaskCriteriaRequest{
			CriterionName:        c.CriterionName,
			CriterionDescription: c.CriterionDescription,
			IsMandatory:          c.IsMandatory,
			Weight:               c.Weight,
		}
	}

	usecaseReq := &usecase.CreateTaskRequest{
		CourseID:    req.CourseID,
		Title:       req.Title,
		Description: req.Description,
		Deadline:    req.Deadline,
		MaxScore:    req.MaxScore,
		Criteria:    criteria,
	}

	resp, err := h.taskUseCase.CreateTask(ctx.Request().Context(), usecaseReq)
	if err != nil {
		return h.handleError(ctx, err)
	}

	h.logger.Info("Task created successfully",
		zap.Int("task_id", resp.TaskID),
	)

	status := api.Active
	response := api.TaskResponse{
		TaskId:    &resp.TaskID,
		CourseId:  &resp.CourseID,
		Title:     &resp.Title,
		Status:    &status,
		CreatedAt: &resp.CreatedAt,
	}

	return ctx.JSON(http.StatusCreated, response)
}

func (h *TaskHandler) handleError(ctx echo.Context, err error) error {
	var validationErr *usecase.ValidationError
	if errors.As(err, &validationErr) {
		details := make([]struct {
			Field   *string `json:"field,omitempty"`
			Message *string `json:"message,omitempty"`
		}, len(validationErr.Details))

		for i, detail := range validationErr.Details {
			details[i].Field = stringPtr(detail.Field)
			details[i].Message = stringPtr(detail.Message)
		}

		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error:   stringPtr(validationErr.Message),
			Details: &details,
		})
	}

	if errors.Is(err, usecase.ErrCourseNotFound) {
		return ctx.JSON(http.StatusNotFound, api.ApiError{
			Error: stringPtr("Course not found"),
		})
	}

	if errors.Is(err, usecase.ErrUnauthorized) {
		return ctx.JSON(http.StatusForbidden, api.ApiError{
			Error: stringPtr("Only teachers can create tasks"),
		})
	}

	return ctx.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Internal server error",
	})
}
