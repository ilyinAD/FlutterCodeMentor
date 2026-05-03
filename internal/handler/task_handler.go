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
	taskUseCase       usecase.TaskUseCase
	submissionUseCase usecase.SubmissionUseCase
	logger            *zap.Logger
}

func NewTaskHandler(taskUseCase usecase.TaskUseCase, submissionUseCase usecase.SubmissionUseCase, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		taskUseCase:       taskUseCase,
		submissionUseCase: submissionUseCase,
		logger:            logger,
	}
}

type CreateTaskRequest struct {
	CourseID    int                         `json:"course_id"`
	Title       string                      `json:"title"`
	Description string                      `json:"description"`
	Deadline    time.Time                   `json:"deadline"`
	MaxScore    int                         `json:"max_score"`
	Criteria    []CreateTaskCriteriaRequest `json:"criteria,omitempty"`
}

type CreateTaskCriteriaRequest struct {
	CriterionName        string `json:"criterion_name"`
	CriterionDescription string `json:"criterion_description"`
	IsMandatory          bool   `json:"is_mandatory"`
	Weight               int    `json:"weight"`
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

	return ctx.JSON(http.StatusCreated, taskDetailToAPI(resp))
}

func (h *TaskHandler) GetTasksTaskId(ctx echo.Context, taskID int) error {
	task, err := h.taskUseCase.GetTaskByID(ctx.Request().Context(), taskID)
	if err != nil {
		return h.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, taskDetailToAPI(task))
}

func (h *TaskHandler) GetTasksTaskIdSubmissions(ctx echo.Context, taskID int, params api.GetTasksTaskIdSubmissionsParams) error {
	submissions, err := h.submissionUseCase.GetSubmissionsByTaskID(ctx.Request().Context(), taskID, params.StudentId)
	if err != nil {
		return h.handleError(ctx, err)
	}
	response := make([]api.SubmissionDetailResponse, 0, len(submissions))
	for _, s := range submissions {
		response = append(response, submissionDetailToAPI(s))
	}
	return ctx.JSON(http.StatusOK, response)
}

func taskDetailToAPI(t *usecase.TaskDetailResponse) api.TaskDetailResponse {
	criteria := make([]api.TaskCriteriaResponse, 0, len(t.Criteria))
	for _, c := range t.Criteria {
		c := c
		criteria = append(criteria, api.TaskCriteriaResponse{
			Id:                   &c.ID,
			CriterionName:        &c.CriterionName,
			CriterionDescription: &c.CriterionDescription,
			IsMandatory:          &c.IsMandatory,
			Weight:               &c.Weight,
		})
	}
	status := api.TaskDetailResponseStatus(t.Status)
	return api.TaskDetailResponse{
		TaskId:      &t.TaskID,
		CourseId:    &t.CourseID,
		Title:       &t.Title,
		Description: &t.Description,
		Deadline:    &t.Deadline,
		MaxScore:    &t.MaxScore,
		Status:      &status,
		CreatedAt:   &t.CreatedAt,
		Criteria:    &criteria,
	}
}

func (h *TaskHandler) handleError(ctx echo.Context, err error) error {
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

	if errors.Is(err, usecase.ErrTaskNotFound) {
		return ctx.JSON(http.StatusNotFound, api.NotFound{
			Error: stringPtr("Task not found"),
		})
	}

	h.logger.Error("unhandled task error",
		zap.String("path", ctx.Request().URL.Path),
		zap.String("method", ctx.Request().Method),
		zap.Error(err),
	)
	return ctx.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Internal server error",
	})
}
