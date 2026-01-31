package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewRepository interface {
	CreateCodeReview(ctx context.Context, review *domain.CodeReview) (int, error)
	CreateReviewFeedback(ctx context.Context, feedback *domain.ReviewFeedback) error
	GetCodeReviewBySubmissionID(ctx context.Context, submissionID int) (*domain.CodeReview, error)
	GetReviewFeedbackByReviewID(ctx context.Context, reviewID int) ([]*domain.ReviewFeedback, error)
}

type reviewRepository struct {
	pool *pgxpool.Pool
}

func NewReviewRepository(pool *pgxpool.Pool) ReviewRepository {
	return &reviewRepository{pool: pool}
}

func (r *reviewRepository) CreateCodeReview(ctx context.Context, review *domain.CodeReview) (int, error) {
	query := `
		INSERT INTO code_reviews (
			submission_id, ai_model, overall_status,
			ai_confidence, execution_time_ms
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		review.SubmissionID,
		review.AIModel,
		review.OverallStatus,
		review.AIConfidence,
		review.ExecutionTimeMs,
	).Scan(&id, &review.CreatedAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create code review: %w", err)
	}

	return id, nil
}

func (r *reviewRepository) CreateReviewFeedback(ctx context.Context, feedback *domain.ReviewFeedback) error {
	query := `
		INSERT INTO review_feedback (
			review_id, feedback_type, file_path, line_start, line_end,
			code_snippet, suggested_fix, description, severity,
			is_resolved, teacher_comment, teacher_approved
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		feedback.ReviewID,
		feedback.FeedbackType,
		feedback.FilePath,
		feedback.LineStart,
		feedback.LineEnd,
		feedback.CodeSnippet,
		feedback.SuggestedFix,
		feedback.Description,
		feedback.Severity,
		feedback.IsResolved,
		feedback.TeacherComment,
		feedback.TeacherApproved,
	).Scan(&feedback.ID, &feedback.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create review feedback: %w", err)
	}

	return nil
}

func (r *reviewRepository) GetCodeReviewBySubmissionID(ctx context.Context, submissionID int) (*domain.CodeReview, error) {
	query := `
		SELECT id, submission_id, ai_model, overall_status,
			   ai_confidence, execution_time_ms, created_at
		FROM code_reviews
		WHERE submission_id = $1
	`

	review := &domain.CodeReview{}
	err := r.pool.QueryRow(ctx, query, submissionID).Scan(
		&review.ID,
		&review.SubmissionID,
		&review.AIModel,
		&review.OverallStatus,
		&review.AIConfidence,
		&review.ExecutionTimeMs,
		&review.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get code review: %w", err)
	}

	return review, nil
}

func (r *reviewRepository) GetReviewFeedbackByReviewID(ctx context.Context, reviewID int) ([]*domain.ReviewFeedback, error) {
	query := `
		SELECT id, review_id, feedback_type, file_path, line_start, line_end,
			   code_snippet, suggested_fix, description, severity,
			   is_resolved, teacher_comment, teacher_approved, created_at
		FROM review_feedback
		WHERE review_id = $1
		ORDER BY severity DESC, line_start ASC
	`

	rows, err := r.pool.Query(ctx, query, reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to query review feedback: %w", err)
	}
	defer rows.Close()

	var feedbacks []*domain.ReviewFeedback
	for rows.Next() {
		feedback := &domain.ReviewFeedback{}
		err := rows.Scan(
			&feedback.ID,
			&feedback.ReviewID,
			&feedback.FeedbackType,
			&feedback.FilePath,
			&feedback.LineStart,
			&feedback.LineEnd,
			&feedback.CodeSnippet,
			&feedback.SuggestedFix,
			&feedback.Description,
			&feedback.Severity,
			&feedback.IsResolved,
			&feedback.TeacherComment,
			&feedback.TeacherApproved,
			&feedback.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review feedback: %w", err)
		}

		feedbacks = append(feedbacks, feedback)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating review feedback: %w", err)
	}

	return feedbacks, nil
}
