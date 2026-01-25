package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/api"
	"github.com/ilyin-ad/flutter-code-mentor/internal/usecase"
	"github.com/labstack/echo/v4"
)

type TaskHandler struct {
	taskUseCase usecase.TaskUseCase
}

func NewTaskHandler(taskUseCase usecase.TaskUseCase) *TaskHandler {
	return &TaskHandler{
		taskUseCase: taskUseCase,
	}
}

type CreateTaskRequest struct {
	CourseID    int       `json:"course_id" validate:"required,min=1"`
	Title       string    `json:"title" validate:"required,min=5,max=100"`
	Description string    `json:"description" validate:"required,min=10"`
	Deadline    time.Time `json:"deadline" validate:"required"`
	MaxScore    int       `json:"max_score" validate:"required,min=1,max=100"`
}

func (h *TaskHandler) PostTask(ctx echo.Context) error {
	var req CreateTaskRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	usecaseReq := &usecase.CreateTaskRequest{
		CourseID:    req.CourseID,
		Title:       req.Title,
		Description: req.Description,
		Deadline:    req.Deadline,
		MaxScore:    req.MaxScore,
	}

	resp, err := h.taskUseCase.CreateTask(ctx.Request().Context(), usecaseReq)
	if err != nil {
		return h.handleError(ctx, err)
	}

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
