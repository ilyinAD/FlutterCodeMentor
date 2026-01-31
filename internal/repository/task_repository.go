package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) (int, error)
	GetByID(ctx context.Context, id int) (*domain.Task, error)
	GetByCourseID(ctx context.Context, courseID int) ([]*domain.Task, error)
	CreateCriteria(ctx context.Context, criteria *domain.TaskCriteria) (int, error)
	GetCriteriaByTaskID(ctx context.Context, taskID int) ([]*domain.TaskCriteria, error)
	DeleteCriteriaByTaskID(ctx context.Context, taskID int) error
}

type taskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) TaskRepository {
	return &taskRepository{pool: pool}
}

func (r *taskRepository) Create(ctx context.Context, task *domain.Task) (int, error) {
	query := `
		INSERT INTO tasks (course_id, title, description, deadline, max_score)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		task.CourseID,
		task.Title,
		task.Description,
		task.Deadline,
		task.MaxScore,
	).Scan(&id, &task.CreatedAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create task: %w", err)
	}

	return id, nil
}

func (r *taskRepository) GetByID(ctx context.Context, id int) (*domain.Task, error) {
	query := `
		SELECT id, course_id, title, description, deadline, max_score, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	task := &domain.Task{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.CourseID,
		&task.Title,
		&task.Description,
		&task.Deadline,
		&task.MaxScore,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

func (r *taskRepository) GetByCourseID(ctx context.Context, courseID int) ([]*domain.Task, error) {
	query := `
		SELECT id, course_id, title, description, deadline, max_score, created_at, updated_at
		FROM tasks
		WHERE course_id = $1
		ORDER BY deadline ASC
	`

	rows, err := r.pool.Query(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task := &domain.Task{}
		err := rows.Scan(
			&task.ID,
			&task.CourseID,
			&task.Title,
			&task.Description,
			&task.Deadline,
			&task.MaxScore,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

func (r *taskRepository) CreateCriteria(ctx context.Context, criteria *domain.TaskCriteria) (int, error) {
	query := `
		INSERT INTO task_criteria (task_id, criterion_name, criterion_description, is_mandatory, weight)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		criteria.TaskID,
		criteria.CriterionName,
		criteria.CriterionDescription,
		criteria.IsMandatory,
		criteria.Weight,
	).Scan(&id, &criteria.CreatedAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create task criteria: %w", err)
	}

	return id, nil
}

func (r *taskRepository) GetCriteriaByTaskID(ctx context.Context, taskID int) ([]*domain.TaskCriteria, error) {
	query := `
		SELECT id, task_id, criterion_name, criterion_description, is_mandatory, weight, created_at
		FROM task_criteria
		WHERE task_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task criteria: %w", err)
	}
	defer rows.Close()

	var criteria []*domain.TaskCriteria
	for rows.Next() {
		c := &domain.TaskCriteria{}
		err := rows.Scan(
			&c.ID,
			&c.TaskID,
			&c.CriterionName,
			&c.CriterionDescription,
			&c.IsMandatory,
			&c.Weight,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task criteria: %w", err)
		}

		criteria = append(criteria, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task criteria: %w", err)
	}

	return criteria, nil
}

func (r *taskRepository) DeleteCriteriaByTaskID(ctx context.Context, taskID int) error {
	query := `DELETE FROM task_criteria WHERE task_id = $1`

	_, err := r.pool.Exec(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task criteria: %w", err)
	}

	return nil
}
