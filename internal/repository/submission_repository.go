package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type SubmissionRepository interface {
	Create(ctx context.Context, submission *domain.Submission) (int, error)
	GetByID(ctx context.Context, id int) (*domain.Submission, error)
	GetByTaskAndStudent(ctx context.Context, taskID, studentID int) ([]*domain.Submission, error)
	GetPendingSubmissions(ctx context.Context) ([]*domain.Submission, error)
	UpdateStatus(ctx context.Context, id int, status domain.SubmissionStatus) error
}

type submissionRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewSubmissionRepository(pool *pgxpool.Pool, logger *zap.Logger) SubmissionRepository {
	return &submissionRepository{pool: pool, logger: logger}
}

func (r *submissionRepository) Create(ctx context.Context, submission *domain.Submission) (int, error) {
	query := `
		INSERT INTO submissions (student_id, task_id, code, github_url, status, submission_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, submitted_at
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		submission.StudentID,
		submission.TaskID,
		submission.Code,
		submission.GithubURL,
		submission.Status,
		submission.SubmissionType,
	).Scan(&id, &submission.SubmittedAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create submission: %w", err)
	}

	return id, nil
}

func (r *submissionRepository) GetByID(ctx context.Context, id int) (*domain.Submission, error) {
	query := `
		SELECT id, student_id, task_id, code, github_url, submitted_at, score, status, submission_type
		FROM submissions
		WHERE id = $1
	`

	submission := &domain.Submission{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&submission.ID,
		&submission.StudentID,
		&submission.TaskID,
		&submission.Code,
		&submission.GithubURL,
		&submission.SubmittedAt,
		&submission.Score,
		&submission.Status,
		&submission.SubmissionType,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	return submission, nil
}

func (r *submissionRepository) GetByTaskAndStudent(ctx context.Context, taskID, studentID int) ([]*domain.Submission, error) {
	query := `
		SELECT id, student_id, task_id, code, github_url, submitted_at, score, status, submission_type
		FROM submissions
		WHERE task_id = $1 AND student_id = $2
		ORDER BY submitted_at DESC
	`

	rows, err := r.pool.Query(ctx, query, taskID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*domain.Submission
	for rows.Next() {
		submission := &domain.Submission{}
		err := rows.Scan(
			&submission.ID,
			&submission.StudentID,
			&submission.TaskID,
			&submission.Code,
			&submission.GithubURL,
			&submission.SubmittedAt,
			&submission.Score,
			&submission.Status,
			&submission.SubmissionType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		submissions = append(submissions, submission)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submissions: %w", err)
	}

	return submissions, nil
}

func (r *submissionRepository) GetPendingSubmissions(ctx context.Context) ([]*domain.Submission, error) {
	query := `
		SELECT id, student_id, task_id, code, github_url, submitted_at, score, status, submission_type
		FROM submissions
		WHERE status = $1
		ORDER BY submitted_at ASC
		LIMIT 10
	`

	rows, err := r.pool.Query(ctx, query, domain.StatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*domain.Submission
	for rows.Next() {
		submission := &domain.Submission{}
		err := rows.Scan(
			&submission.ID,
			&submission.StudentID,
			&submission.TaskID,
			&submission.Code,
			&submission.GithubURL,
			&submission.SubmittedAt,
			&submission.Score,
			&submission.Status,
			&submission.SubmissionType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		submissions = append(submissions, submission)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submissions: %w", err)
	}

	return submissions, nil
}

func (r *submissionRepository) UpdateStatus(ctx context.Context, id int, status domain.SubmissionStatus) error {
	query := `
		UPDATE submissions
		SET status = $1
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}

	return nil
}
