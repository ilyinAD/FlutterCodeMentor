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
	GetByStudentID(ctx context.Context, studentID int) ([]*domain.Course, error)
	GetAll(ctx context.Context) ([]*domain.Course, error)
	CreateEnrollment(ctx context.Context, courseID, studentID int) (*domain.Enrollment, bool, error)
	GetEnrollmentsByCourseID(ctx context.Context, courseID int) ([]*domain.Enrollment, error)
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

func (r *courseRepository) GetByStudentID(ctx context.Context, studentID int) ([]*domain.Course, error) {
	query := `
		SELECT c.id, c.teacher_id, c.title, c.description, c.start_date, c.end_date, c.is_active, c.created_at
		FROM courses c
		JOIN course_enrollments e ON e.course_id = c.id
		WHERE e.student_id = $1
		ORDER BY c.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, studentID)
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

func (r *courseRepository) CreateEnrollment(ctx context.Context, courseID, studentID int) (*domain.Enrollment, bool, error) {
	insertQuery := `
		INSERT INTO course_enrollments (student_id, course_id)
		VALUES ($1, $2)
		ON CONFLICT (student_id, course_id) DO NOTHING
		RETURNING enrolled_at
	`

	enrollment := &domain.Enrollment{
		StudentID: studentID,
		CourseID:  courseID,
	}

	err := r.pool.QueryRow(ctx, insertQuery, studentID, courseID).Scan(&enrollment.EnrolledAt)
	if err == nil {
		return enrollment, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("failed to create enrollment: %w", err)
	}

	selectQuery := `
		SELECT enrolled_at
		FROM course_enrollments
		WHERE student_id = $1 AND course_id = $2
	`
	if err := r.pool.QueryRow(ctx, selectQuery, studentID, courseID).Scan(&enrollment.EnrolledAt); err != nil {
		return nil, false, fmt.Errorf("failed to load existing enrollment: %w", err)
	}
	return enrollment, true, nil
}

func (r *courseRepository) GetEnrollmentsByCourseID(ctx context.Context, courseID int) ([]*domain.Enrollment, error) {
	query := `
		SELECT student_id, course_id, enrolled_at
		FROM course_enrollments
		WHERE course_id = $1
		ORDER BY enrolled_at ASC
	`

	rows, err := r.pool.Query(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query enrollments: %w", err)
	}
	defer rows.Close()

	var enrollments []*domain.Enrollment
	for rows.Next() {
		e := &domain.Enrollment{}
		if err := rows.Scan(&e.StudentID, &e.CourseID, &e.EnrolledAt); err != nil {
			return nil, fmt.Errorf("failed to scan enrollment: %w", err)
		}
		enrollments = append(enrollments, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating enrollments: %w", err)
	}

	return enrollments, nil
}

func (r *courseRepository) GetAll(ctx context.Context) ([]*domain.Course, error) {
	query := `
		SELECT id, teacher_id, title, description, start_date, end_date, is_active, created_at
		FROM courses
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
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
