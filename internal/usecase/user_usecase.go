package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/ilyin-ad/flutter-code-mentor/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password must be at least 12 characters")
)

type UserUseCase interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error)
}

type userUseCase struct {
	userRepo repository.UserRepository
}

func NewUserUseCase(userRepo repository.UserRepository) UserUseCase {
	return &userUseCase{
		userRepo: userRepo,
	}
}

type CreateUserRequest struct {
	Email     string
	Password  string
	Role      string
	FirstName string
	LastName  string
}

type CreateUserResponse struct {
	UserID    int
	Email     string
	Role      string
	CreatedAt time.Time
}

func (uc *userUseCase) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	if err := uc.validateUserRequest(req); err != nil {
		return nil, err
	}

	existingUser, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         req.Role,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
	}

	userID, err := uc.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &CreateUserResponse{
		UserID:    userID,
		Email:     req.Email,
		Role:      req.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (uc *userUseCase) validateUserRequest(req *CreateUserRequest) error {
	var details []ValidationErrorDetail

	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(emailRegex, req.Email)
	if !matched {
		details = append(details, ValidationErrorDetail{
			Field:   "email",
			Message: "Invalid email format",
		})
	}

	if len(req.Password) < 12 {
		details = append(details, ValidationErrorDetail{
			Field:   "password",
			Message: "Must be at least 12 characters",
		})
	}

	validRoles := map[string]bool{
		"student": true,
		"teacher": true,
	}
	if !validRoles[req.Role] {
		details = append(details, ValidationErrorDetail{
			Field:   "role",
			Message: "Must be either 'student' or 'teacher'",
		})
	}

	if len(req.FirstName) < 2 || len(req.FirstName) > 50 {
		details = append(details, ValidationErrorDetail{
			Field:   "first_name",
			Message: "Must be between 2 and 50 characters",
		})
	}

	if len(req.LastName) < 2 || len(req.LastName) > 50 {
		details = append(details, ValidationErrorDetail{
			Field:   "last_name",
			Message: "Must be between 2 and 50 characters",
		})
	}

	if len(details) > 0 {
		return &ValidationError{
			Message: "Validation failed",
			Details: details,
		}
	}

	return nil
}
