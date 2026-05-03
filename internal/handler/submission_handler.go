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
	reviewUseCase     usecase.ReviewUseCase
	logger            *zap.Logger
}

func NewSubmissionHandler(submissionUseCase usecase.SubmissionUseCase, reviewUseCase usecase.ReviewUseCase, logger *zap.Logger) *SubmissionHandler {
	return &SubmissionHandler{
		submissionUseCase: submissionUseCase,
		reviewUseCase:     reviewUseCase,
		logger:            logger,
	}
}

type CreateSubmissionRequest struct {
	TaskID         int     `json:"task_id"`
	UserID         int     `json:"user_id"`
	SubmissionType string  `json:"submission_type"`
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

func (h *SubmissionHandler) GetSubmissionsSubmissionId(ctx echo.Context, submissionID int) error {
	detail, err := h.submissionUseCase.GetSubmissionByID(ctx.Request().Context(), submissionID)
	if err != nil {
		return h.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, submissionDetailToAPI(detail))
}

func (h *SubmissionHandler) GetSubmissionsSubmissionIdReviews(ctx echo.Context, submissionID int) error {
	review, err := h.reviewUseCase.GetReviewBySubmissionID(ctx.Request().Context(), submissionID)
	if err != nil {
		return h.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, reviewDetailToAPI(review))
}

func (h *SubmissionHandler) PostSubmissionsSubmissionIdTeacherReview(ctx echo.Context, submissionID int) error {
	var req api.TeacherReviewRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	input := &usecase.TeacherReviewInput{}
	if req.Actions != nil {
		for _, a := range *req.Actions {
			input.Actions = append(input.Actions, usecase.FeedbackActionInput{
				FeedbackID:      a.FeedbackId,
				TeacherApproved: a.TeacherApproved,
				TeacherComment:  a.TeacherComment,
			})
		}
	}
	if req.TeacherFeedbacks != nil {
		for _, tf := range *req.TeacherFeedbacks {
			input.TeacherFeedbacks = append(input.TeacherFeedbacks, usecase.TeacherFeedbackInput{
				FeedbackType: tf.FeedbackType,
				FilePath:     tf.FilePath,
				LineStart:    tf.LineStart,
				LineEnd:      tf.LineEnd,
				CodeSnippet:  tf.CodeSnippet,
				SuggestedFix: tf.SuggestedFix,
				Description:  tf.Description,
				Severity:     tf.Severity,
			})
		}
	}

	review, err := h.reviewUseCase.SubmitTeacherReview(ctx.Request().Context(), submissionID, input)
	if err != nil {
		return h.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, reviewDetailToAPI(review))
}

func (h *SubmissionHandler) PutSubmissionsSubmissionIdGrade(ctx echo.Context, submissionID int) error {
	var req api.GradeRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	detail, err := h.reviewUseCase.GradeSubmission(ctx.Request().Context(), submissionID, float64(req.Score), req.Comment)
	if err != nil {
		return h.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, submissionDetailToAPI(detail))
}

func submissionDetailToAPI(d *usecase.SubmissionDetail) api.SubmissionDetailResponse {
	status := api.SubmissionDetailResponseStatus(d.Status)
	subType := api.SubmissionDetailResponseSubmissionType(d.SubmissionType)
	resp := api.SubmissionDetailResponse{
		SubmissionId:   &d.SubmissionID,
		TaskId:         &d.TaskID,
		UserId:         &d.UserID,
		StudentName:    &d.StudentName,
		SubmissionType: &subType,
		Code:           d.Code,
		GithubUrl:      d.GithubURL,
		Status:         &status,
		CreatedAt:      &d.CreatedAt,
	}
	if d.Score != nil {
		score := float32(*d.Score)
		resp.Score = &score
	}
	return resp
}

func reviewDetailToAPI(r *usecase.ReviewDetail) api.CodeReviewResponse {
	overall := api.CodeReviewResponseOverallStatus(r.OverallStatus)
	feedbacks := make([]api.ReviewFeedbackResponse, 0, len(r.Feedbacks))
	for _, f := range r.Feedbacks {
		f := f
		feedbacks = append(feedbacks, api.ReviewFeedbackResponse{
			Id:              &f.ID,
			ReviewId:        &f.ReviewID,
			FeedbackType:    &f.FeedbackType,
			FilePath:        f.FilePath,
			LineStart:       &f.LineStart,
			LineEnd:         f.LineEnd,
			CodeSnippet:     &f.CodeSnippet,
			SuggestedFix:    f.SuggestedFix,
			Description:     &f.Description,
			Severity:        &f.Severity,
			IsResolved:      &f.IsResolved,
			TeacherComment:  f.TeacherComment,
			TeacherApproved: f.TeacherApproved,
		})
	}
	resp := api.CodeReviewResponse{
		Id:              &r.ID,
		SubmissionId:    &r.SubmissionID,
		AiModel:         &r.AIModel,
		OverallStatus:   &overall,
		ExecutionTimeMs: r.ExecutionTimeMs,
		CreatedAt:       &r.CreatedAt,
		Feedbacks:       &feedbacks,
	}
	if r.AIConfidence != nil {
		conf := float32(*r.AIConfidence)
		resp.AiConfidence = &conf
	}
	return resp
}

func (h *SubmissionHandler) handleError(ctx echo.Context, err error) error {
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

	if errors.Is(err, usecase.ErrSubmissionNotFound) {
		return ctx.JSON(http.StatusNotFound, api.NotFound{
			Error: stringPtr("Submission not found"),
		})
	}

	if errors.Is(err, usecase.ErrReviewNotFound) {
		return ctx.JSON(http.StatusNotFound, api.NotFound{
			Error: stringPtr("Review not found"),
		})
	}

	h.logger.Error("unhandled submission error",
		zap.String("path", ctx.Request().URL.Path),
		zap.String("method", ctx.Request().Method),
		zap.Error(err),
	)
	return ctx.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Internal server error",
	})
}

func stringPtr(s string) *string {
	return &s
}
