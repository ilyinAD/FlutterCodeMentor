package handler

import (
	"errors"
	"net/http"

	"github.com/ilyin-ad/flutter-code-mentor/api"
	"github.com/ilyin-ad/flutter-code-mentor/internal/usecase"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type UserHandler struct {
	userUseCase usecase.UserUseCase
	logger      *zap.Logger
}

func NewUserHandler(userUseCase usecase.UserUseCase, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
		logger:      logger,
	}
}

type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=12"`
	Role      string `json:"role" validate:"required,oneof=student teacher"`
	FirstName string `json:"first_name" validate:"required,min=2,max=50"`
	LastName  string `json:"last_name" validate:"required,min=2,max=50"`
}

func (h *UserHandler) PostUser(ctx echo.Context) error {
	h.logger.Info("Received user creation request",
		zap.String("method", ctx.Request().Method),
		zap.String("path", ctx.Request().URL.Path),
	)

	var req CreateUserRequest
	if err := ctx.Bind(&req); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	usecaseReq := &usecase.CreateUserRequest{
		Email:     req.Email,
		Password:  req.Password,
		Role:      req.Role,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	resp, err := h.userUseCase.CreateUser(ctx.Request().Context(), usecaseReq)
	if err != nil {
		return h.handleError(ctx, err)
	}

	h.logger.Info("User created successfully",
		zap.Int("user_id", resp.UserID),
		zap.String("email", resp.Email),
	)

	return ctx.JSON(http.StatusCreated, userToResponse(resp))
}

type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) PostAuthLogin(ctx echo.Context) error {
	var req LoginUserRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ValidationError{
			Error: stringPtr("Invalid request body"),
		})
	}

	resp, err := h.userUseCase.Login(ctx.Request().Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			return ctx.JSON(http.StatusUnauthorized, api.ApiError{
				Error: stringPtr("Invalid credentials"),
			})
		}
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return ctx.JSON(http.StatusOK, userToResponse(resp))
}

func userToResponse(resp *usecase.CreateUserResponse) api.UserResponse {
	return api.UserResponse{
		UserId:    &resp.UserID,
		Email:     &resp.Email,
		Role:      (*api.UserResponseRole)(&resp.Role),
		FirstName: &resp.FirstName,
		LastName:  &resp.LastName,
		CreatedAt: &resp.CreatedAt,
	}
}

func (h *UserHandler) handleError(ctx echo.Context, err error) error {
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

	if errors.Is(err, usecase.ErrEmailAlreadyExists) {
		return ctx.JSON(http.StatusConflict, api.ApiError{
			Error: stringPtr("Email already exists"),
		})
	}

	h.logger.Error("unhandled user error",
		zap.String("path", ctx.Request().URL.Path),
		zap.String("method", ctx.Request().Method),
		zap.Error(err),
	)
	return ctx.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Internal server error",
	})
}
