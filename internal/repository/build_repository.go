package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BuildRepository interface {
	CreateBuildResult(ctx context.Context, result *domain.BuildResult) (int, error)
	GetBuildResultBySubmissionID(ctx context.Context, submissionID int) (*domain.BuildResult, error)
}

type buildRepository struct {
	pool *pgxpool.Pool
}

func NewBuildRepository(pool *pgxpool.Pool) BuildRepository {
	return &buildRepository{pool: pool}
}

func (r *buildRepository) CreateBuildResult(ctx context.Context, result *domain.BuildResult) (int, error) {
	query := `
		INSERT INTO build_results (
			submission_id, compile_success, analyze_output,
			test_output, tests_passed, format_output,
			format_correct, execution_time_ms
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		result.SubmissionID,
		result.CompileSuccess,
		result.AnalyzeOutput,
		result.TestOutput,
		result.TestsPassed,
		result.FormatOutput,
		result.FormatCorrect,
		result.ExecutionTimeMs,
	).Scan(&id, &result.CreatedAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create build result: %w", err)
	}

	return id, nil
}

func (r *buildRepository) GetBuildResultBySubmissionID(ctx context.Context, submissionID int) (*domain.BuildResult, error) {
	query := `
		SELECT id, submission_id, compile_success, analyze_output,
			   test_output, tests_passed, format_output, format_correct,
			   execution_time_ms, created_at
		FROM build_results
		WHERE submission_id = $1
	`

	result := &domain.BuildResult{}
	err := r.pool.QueryRow(ctx, query, submissionID).Scan(
		&result.ID,
		&result.SubmissionID,
		&result.CompileSuccess,
		&result.AnalyzeOutput,
		&result.TestOutput,
		&result.TestsPassed,
		&result.FormatOutput,
		&result.FormatCorrect,
		&result.ExecutionTimeMs,
		&result.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get build result: %w", err)
	}

	return result, nil
}
