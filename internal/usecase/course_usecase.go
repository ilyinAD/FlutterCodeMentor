package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/ilyin-ad/flutter-code-mentor/internal/repository"
)

type CourseUseCase interface {
	CreateCourse(ctx context.Context, req *CreateCourseRequest) (*CreateCourseResponse, error)
}

type courseUseCase struct {
	courseRepo repository.CourseRepository
	userRepo   repository.UserRepository
}

func NewCourseUseCase(
	courseRepo repository.CourseRepository,
	userRepo repository.UserRepository,
) CourseUseCase {
	return &courseUseCase{
		courseRepo: courseRepo,
		userRepo:   userRepo,
	}
}

type CreateCourseRequest struct {
	TeacherID   int
	Title       string
	Description *string
	StartDate   time.Time
	EndDate     *time.Time
	IsActive    bool
}

type CreateCourseResponse struct {
	CourseID    int
	TeacherID   int
	Title       string
	Description *string
	StartDate   time.Time
	EndDate     *time.Time
	IsActive    bool
	CreatedAt   time.Time
}

func (uc *courseUseCase) CreateCourse(ctx context.Context, req *CreateCourseRequest) (*CreateCourseResponse, error) {
	if err := uc.validateCourseRequest(req); err != nil {
		return nil, err
	}

	teacher, err := uc.userRepo.GetByID(ctx, req.TeacherID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserNotFound, err)
	}
	if teacher == nil {
		return nil, ErrUserNotFound
	}

	if teacher.Role != "teacher" {
		return nil, ErrUnauthorized
	}

	course := &domain.Course{
		TeacherID:   req.TeacherID,
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		IsActive:    req.IsActive,
	}

	courseID, err := uc.courseRepo.Create(ctx, course)
	if err != nil {
		return nil, fmt.Errorf("failed to create course: %w", err)
	}

	return &CreateCourseResponse{
		CourseID:    courseID,
		TeacherID:   req.TeacherID,
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		IsActive:    req.IsActive,
		CreatedAt:   course.CreatedAt,
	}, nil
}

func (uc *courseUseCase) validateCourseRequest(req *CreateCourseRequest) error {
	var details []ValidationErrorDetail

	if len(req.Title) < 3 || len(req.Title) > 100 {
		details = append(details, ValidationErrorDetail{
			Field:   "title",
			Message: "Must be between 3 and 100 characters",
		})
	}

	if req.EndDate != nil && req.EndDate.Before(req.StartDate) {
		details = append(details, ValidationErrorDetail{
			Field:   "end_date",
			Message: "Must be after start_date",
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
