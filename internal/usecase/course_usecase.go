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
	GetCourses(ctx context.Context, teacherID *int) ([]*CreateCourseResponse, error)
	GetCourseByID(ctx context.Context, id int) (*CreateCourseResponse, error)
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

func courseToResponse(c *domain.Course) *CreateCourseResponse {
	return &CreateCourseResponse{
		CourseID:    c.ID,
		TeacherID:   c.TeacherID,
		Title:       c.Title,
		Description: c.Description,
		StartDate:   c.StartDate,
		EndDate:     c.EndDate,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
	}
}

func (uc *courseUseCase) GetCourses(ctx context.Context, teacherID *int) ([]*CreateCourseResponse, error) {
	var (
		courses []*domain.Course
		err     error
	)
	if teacherID != nil {
		courses, err = uc.courseRepo.GetByTeacherID(ctx, *teacherID)
	} else {
		courses, err = uc.courseRepo.GetAll(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list courses: %w", err)
	}

	responses := make([]*CreateCourseResponse, 0, len(courses))
	for _, c := range courses {
		responses = append(responses, courseToResponse(c))
	}
	return responses, nil
}

func (uc *courseUseCase) GetCourseByID(ctx context.Context, id int) (*CreateCourseResponse, error) {
	course, err := uc.courseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}
	return courseToResponse(course), nil
}
