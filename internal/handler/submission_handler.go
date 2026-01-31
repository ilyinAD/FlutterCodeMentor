package handler

import (
	"errors"
	"net/http"

	"github.com/ilyin-ad/flutter-code-mentor/api"
	"github.com/ilyin-ad/flutter-code-mentor/internal/usecase"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type SubmissionHandler struct {
	submissionUseCase usecase.SubmissionUseCase
	logger            *zap.Logger
}

func NewSubmissionHandler(submissionUseCase usecase.SubmissionUseCase, logger *zap.Logger) *SubmissionHandler {
	return &SubmissionHandler{
		submissionUseCase: submissionUseCase,
		logger:            logger,
	}
}

type CreateSubmissionRequest struct {
	TaskID         int     `json:"task_id" validate:"required,min=1"`
	UserID         int     `json:"user_id" validate:"required,min=1"`
	SubmissionType string  `json:"submission_type" validate:"required,oneof=code github_link"`
	Code           *string `json:"code,omitempty"`
	GithubURL      *string `json:"github_url,omitempty"`
}

func (h *SubmissionHandler) PostSubmission(ctx echo.Context) error {
	h.logger.Info("Received submission creation request",
		zap.String("method", ctx.Request().Method),
		zap.String("path", ctx.Request().URL.Path),
	)

	var req CreateSubmissionRequest
	if err := ctx.Bind(&req); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	h.logger.Info("Creating submission",
		zap.Int("task_id", req.TaskID),
		zap.Int("user_id", req.UserID),
		zap.String("submission_type", req.SubmissionType),
	)

	usecaseReq := &usecase.CreateSubmissionRequest{
		TaskID:         req.TaskID,
		UserID:         req.UserID,
		SubmissionType: req.SubmissionType,
		Code:           req.Code,
		GithubURL:      req.GithubURL,
	}

	resp, err := h.submissionUseCase.CreateSubmission(ctx.Request().Context(), usecaseReq)
	if err != nil {
		return h.handleError(ctx, err)
	}

	h.logger.Info("Submission created successfully",
		zap.Int("submission_id", resp.SubmissionID),
	)

	status := api.Pending
	response := api.SubmissionResponse{
		SubmissionId: &resp.SubmissionID,
		Status:       &status,
		CreatedAt:    &resp.CreatedAt,
	}

	return ctx.JSON(http.StatusCreated, response)
}

func (h *SubmissionHandler) handleError(ctx echo.Context, err error) error {
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

		return ctx.JSON(http.StatusUnprocessableEntity, api.ValidationError{
			Error:   stringPtr(validationErr.Message),
			Details: &details,
		})
	}

	if errors.Is(err, usecase.ErrTaskNotFound) {
		return ctx.JSON(http.StatusNotFound, api.NotFound{
			Error: stringPtr("Task not found"),
		})
	}

	if errors.Is(err, usecase.ErrUserNotFound) {
		return ctx.JSON(http.StatusNotFound, api.NotFound{
			Error: stringPtr("User not found"),
		})
	}

	return ctx.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Internal server error",
	})
}

func stringPtr(s string) *string {
	return &s
}
