package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseRepository interface {
	Create(ctx context.Context, course *domain.Course) (int, error)
	GetByID(ctx context.Context, id int) (*domain.Course, error)
	GetByTeacherID(ctx context.Context, teacherID int) ([]*domain.Course, error)
}

type courseRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) CourseRepository {
	return &courseRepository{pool: pool}
}

func (r *courseRepository) Create(ctx context.Context, course *domain.Course) (int, error) {
	query := `
		INSERT INTO courses (teacher_id, title, description, start_date, end_date, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		course.TeacherID,
		course.Title,
		course.Description,
		course.StartDate,
		course.EndDate,
		course.IsActive,
	).Scan(&id, &course.CreatedAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create course: %w", err)
	}

	return id, nil
}

func (r *courseRepository) GetByID(ctx context.Context, id int) (*domain.Course, error) {
	query := `
		SELECT id, teacher_id, title, description, start_date, end_date, is_active, created_at
		FROM courses
		WHERE id = $1
	`

	course := &domain.Course{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&course.ID,
		&course.TeacherID,
		&course.Title,
		&course.Description,
		&course.StartDate,
		&course.EndDate,
		&course.IsActive,
		&course.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	return course, nil
}

func (r *courseRepository) GetByTeacherID(ctx context.Context, teacherID int) ([]*domain.Course, error) {
	query := `
		SELECT id, teacher_id, title, description, start_date, end_date, is_active, created_at
		FROM courses
		WHERE teacher_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to query courses: %w", err)
	}
	defer rows.Close()

	var courses []*domain.Course
	for rows.Next() {
		course := &domain.Course{}
		err := rows.Scan(
			&course.ID,
			&course.TeacherID,
			&course.Title,
			&course.Description,
			&course.StartDate,
			&course.EndDate,
			&course.IsActive,
			&course.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan course: %w", err)
		}

		courses = append(courses, course)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating courses: %w", err)
	}

	return courses, nil
}
