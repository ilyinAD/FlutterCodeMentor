package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/ilyin-ad/flutter-code-mentor/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password must be at least 12 characters")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserUseCase interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error)
	Login(ctx context.Context, email, password string) (*CreateUserResponse, error)
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
	FirstName string
	LastName  string
	CreatedAt time.Time
}

func (uc *userUseCase) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
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
		FirstName: req.FirstName,
		LastName:  req.LastName,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (uc *userUseCase) Login(ctx context.Context, email, password string) (*CreateUserResponse, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &CreateUserResponse{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
	}, nil
}
