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

type CourseHandler struct {
	courseUseCase usecase.CourseUseCase
	taskUseCase   usecase.TaskUseCase
	logger        *zap.Logger
}

func NewCourseHandler(courseUseCase usecase.CourseUseCase, taskUseCase usecase.TaskUseCase, logger *zap.Logger) *CourseHandler {
	return &CourseHandler{
		courseUseCase: courseUseCase,
		taskUseCase:   taskUseCase,
		logger:        logger,
	}
}

type CreateCourseRequest struct {
	TeacherID   int        `json:"teacher_id" validate:"required,min=1"`
	Title       string     `json:"title" validate:"required,min=3,max=100"`
	Description *string    `json:"description,omitempty"`
	StartDate   time.Time  `json:"start_date" validate:"required"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

func (h *CourseHandler) PostCourses(ctx echo.Context) error {
	h.logger.Info("Received course creation request",
		zap.String("method", ctx.Request().Method),
		zap.String("path", ctx.Request().URL.Path),
	)

	var req CreateCourseRequest
	if err := ctx.Bind(&req); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	usecaseReq := &usecase.CreateCourseRequest{
		TeacherID:   req.TeacherID,
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		IsActive:    isActive,
	}

	resp, err := h.courseUseCase.CreateCourse(ctx.Request().Context(), usecaseReq)
	if err != nil {
		return h.handleError(ctx, err)
	}

	h.logger.Info("Course created successfully",
		zap.Int("course_id", resp.CourseID),
	)

	return ctx.JSON(http.StatusCreated, courseToAPI(resp))
}

func (h *CourseHandler) GetCourses(ctx echo.Context, params api.GetCoursesParams) error {
	courses, err := h.courseUseCase.GetCourses(ctx.Request().Context(), params.TeacherId)
	if err != nil {
		return h.handleError(ctx, err)
	}

	response := make([]api.CourseResponse, 0, len(courses))
	for _, c := range courses {
		response = append(response, courseToAPI(c))
	}
	return ctx.JSON(http.StatusOK, response)
}

func (h *CourseHandler) GetCoursesCourseId(ctx echo.Context, courseID int) error {
	course, err := h.courseUseCase.GetCourseByID(ctx.Request().Context(), courseID)
	if err != nil {
		return h.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, courseToAPI(course))
}

func (h *CourseHandler) GetCoursesCourseIdTasks(ctx echo.Context, courseID int) error {
	tasks, err := h.taskUseCase.GetTasksByCourseID(ctx.Request().Context(), courseID)
	if err != nil {
		return h.handleError(ctx, err)
	}
	response := make([]api.TaskDetailResponse, 0, len(tasks))
	for _, t := range tasks {
		response = append(response, taskDetailToAPI(t))
	}
	return ctx.JSON(http.StatusOK, response)
}

func courseToAPI(resp *usecase.CreateCourseResponse) api.CourseResponse {
	return api.CourseResponse{
		CourseId:    &resp.CourseID,
		TeacherId:   &resp.TeacherID,
		Title:       &resp.Title,
		Description: resp.Description,
		StartDate:   &resp.StartDate,
		EndDate:     resp.EndDate,
		IsActive:    &resp.IsActive,
		CreatedAt:   &resp.CreatedAt,
	}
}

func (h *CourseHandler) handleError(ctx echo.Context, err error) error {
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

	if errors.Is(err, usecase.ErrUserNotFound) {
		return ctx.JSON(http.StatusNotFound, api.ApiError{
			Error: stringPtr("Teacher not found"),
		})
	}

	if errors.Is(err, usecase.ErrUnauthorized) {
		return ctx.JSON(http.StatusForbidden, api.ApiError{
			Error: stringPtr("Only teachers can create courses"),
		})
	}

	if errors.Is(err, usecase.ErrCourseNotFound) {
		return ctx.JSON(http.StatusNotFound, api.NotFound{
			Error: stringPtr("Course not found"),
		})
	}

	h.logger.Error("unhandled course error",
		zap.String("path", ctx.Request().URL.Path),
		zap.String("method", ctx.Request().Method),
		zap.Error(err),
	)
	return ctx.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Internal server error",
	})
}
