package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/ilyin-ad/flutter-code-mentor/internal/repository"
	"github.com/ilyin-ad/flutter-code-mentor/internal/service"
	"go.uber.org/zap"
)

var githubURLPattern = regexp.MustCompile(`^https?://github\.com/[\w-]+/[\w.-]+(?:\.git)?$`)

type ReviewUseCase interface {
	ProcessPendingSubmissions(ctx context.Context) error
}

type reviewUseCase struct {
	submissionRepo repository.SubmissionRepository
	reviewRepo     repository.ReviewRepository
	taskRepo       repository.TaskRepository
	aiService      service.AIService
	githubService  service.GitHubService
	logger         *zap.Logger
}

func NewReviewUseCase(
	submissionRepo repository.SubmissionRepository,
	reviewRepo repository.ReviewRepository,
	taskRepo repository.TaskRepository,
	aiService service.AIService,
	githubService service.GitHubService,
	logger *zap.Logger,
) ReviewUseCase {
	return &reviewUseCase{
		submissionRepo: submissionRepo,
		reviewRepo:     reviewRepo,
		taskRepo:       taskRepo,
		aiService:      aiService,
		githubService:  githubService,
		logger:         logger,
	}
}

func (uc *reviewUseCase) ProcessPendingSubmissions(ctx context.Context) error {
	submissions, err := uc.submissionRepo.GetPendingSubmissions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending submissions: %w", err)
	}

	uc.logger.Info("Processing pending submissions", zap.Int("count", len(submissions)))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for _, submission := range submissions {
		wg.Add(1)
		go func(sub *domain.Submission) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := uc.processSubmission(ctx, sub); err != nil {
				uc.logger.Error("Failed to process submission",
					zap.Int("submission_id", sub.ID),
					zap.Error(err),
				)
			}
		}(submission)
	}

	wg.Wait()
	return nil
}

func (uc *reviewUseCase) processSubmission(ctx context.Context, submission *domain.Submission) error {
	uc.logger.Info("Processing submission",
		zap.Int("submission_id", submission.ID),
		zap.String("type", string(submission.SubmissionType)),
	)

	existingReview, err := uc.reviewRepo.GetCodeReviewBySubmissionID(ctx, submission.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing review: %w", err)
	}

	if existingReview != nil {
		uc.logger.Info("Submission already reviewed", zap.Int("submission_id", submission.ID))
		return nil
	}

	task, err := uc.taskRepo.GetByID(ctx, submission.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found for submission")
	}

	criteria, err := uc.taskRepo.GetCriteriaByTaskID(ctx, submission.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task criteria: %w", err)
	}

	var result *service.CodeReviewResult

	switch submission.SubmissionType {
	case domain.SubmissionTypeCode:
		result, err = uc.processCodeSubmission(ctx, submission, task, criteria)
	case domain.SubmissionTypeGithubLink:
		result, err = uc.processGitHubSubmission(ctx, submission, task, criteria)
	default:
		return fmt.Errorf("unknown submission type: %s", submission.SubmissionType)
	}

	if err != nil {
		return err
	}

	return uc.saveReviewResult(ctx, submission.ID, result)
}

func (uc *reviewUseCase) processCodeSubmission(ctx context.Context, submission *domain.Submission, task *domain.Task, criteria []*domain.TaskCriteria) (*service.CodeReviewResult, error) {
	if submission.Code == nil || *submission.Code == "" {
		return nil, fmt.Errorf("submission has no code to review")
	}

	uc.logger.Info("Reviewing code submission", zap.Int("submission_id", submission.ID))
	return uc.aiService.ReviewCode(ctx, submission.Code, task, criteria)
}

func (uc *reviewUseCase) processGitHubSubmission(ctx context.Context, submission *domain.Submission, task *domain.Task, criteria []*domain.TaskCriteria) (*service.CodeReviewResult, error) {
	if submission.GithubURL == nil || *submission.GithubURL == "" {
		return nil, fmt.Errorf("submission has no GitHub URL to review")
	}

	if !githubURLPattern.MatchString(*submission.GithubURL) {
		return nil, fmt.Errorf("invalid GitHub URL format: %s", *submission.GithubURL)
	}

	uc.logger.Info("Reviewing GitHub submission",
		zap.Int("submission_id", submission.ID),
		zap.String("github_url", *submission.GithubURL),
	)

	repoPath, err := uc.githubService.CloneRepository(ctx, *submission.GithubURL)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer uc.githubService.Cleanup(repoPath)

	dartFiles, err := uc.githubService.GetDartFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get Dart files: %w", err)
	}

	if len(dartFiles) == 0 {
		return nil, fmt.Errorf("no Dart files found in repository")
	}

	uc.logger.Info("Found Dart files in repository",
		zap.Int("submission_id", submission.ID),
		zap.Int("files_count", len(dartFiles)),
	)

	files := make(map[string]string)
	for _, relPath := range dartFiles {
		fullPath := filepath.Join(repoPath, relPath)
		content, err := uc.githubService.ReadFile(fullPath)
		if err != nil {
			uc.logger.Warn("Failed to read file",
				zap.String("file", relPath),
				zap.Error(err),
			)
			continue
		}
		files[relPath] = content
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("failed to read any Dart files from repository")
	}

	return uc.aiService.ReviewGitHubProject(ctx, files, task, criteria)
}

func (uc *reviewUseCase) saveReviewResult(ctx context.Context, submissionID int, result *service.CodeReviewResult) error {
	review := &domain.CodeReview{
		SubmissionID:    submissionID,
		AIModel:         "deepseek",
		OverallStatus:   result.OverallStatus,
		AIConfidence:    &result.AIConfidence,
		ExecutionTimeMs: &result.ExecutionTimeMs,
	}

	reviewID, err := uc.reviewRepo.CreateCodeReview(ctx, review)
	if err != nil {
		return fmt.Errorf("failed to create code review: %w", err)
	}

	uc.logger.Info("Created code review",
		zap.Int("submission_id", submissionID),
		zap.Int("review_id", reviewID),
		zap.String("status", result.OverallStatus),
	)

	for _, fb := range result.Feedbacks {
		var filePath *string
		if fb.FilePath != "" {
			filePath = &fb.FilePath
		}

		feedback := &domain.ReviewFeedback{
			ReviewID:     reviewID,
			FeedbackType: fb.FeedbackType,
			FilePath:     filePath,
			LineStart:    fb.LineStart,
			LineEnd:      &fb.LineEnd,
			CodeSnippet:  fb.CodeSnippet,
			SuggestedFix: &fb.SuggestedFix,
			Description:  fb.Description,
			Severity:     fb.Severity,
			IsResolved:   false,
		}

		if err := uc.reviewRepo.CreateReviewFeedback(ctx, feedback); err != nil {
			uc.logger.Error("Failed to create review feedback",
				zap.Int("review_id", reviewID),
				zap.Error(err),
			)
			continue
		}
	}

	if err := uc.submissionRepo.UpdateStatus(ctx, submissionID, domain.StatusAIReviewed); err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}

	uc.logger.Info("Successfully processed submission",
		zap.Int("submission_id", submissionID),
		zap.Int("feedbacks_count", len(result.Feedbacks)),
	)

	return nil
}
