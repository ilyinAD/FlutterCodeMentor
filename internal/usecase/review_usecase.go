package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/ilyin-ad/flutter-code-mentor/internal/repository"
	"github.com/ilyin-ad/flutter-code-mentor/internal/service"
	"go.uber.org/zap"
)

var githubURLPattern = regexp.MustCompile(`^https?://github\.com/[\w-]+/[\w.-]+(?:\.git)?$`)

type ReviewUseCase interface {
	ProcessPendingSubmissions(ctx context.Context) error
	GetReviewBySubmissionID(ctx context.Context, submissionID int) (*ReviewDetail, error)
	SubmitTeacherReview(ctx context.Context, submissionID int, req *TeacherReviewInput) (*ReviewDetail, error)
	GradeSubmission(ctx context.Context, submissionID int, score float64, comment *string) (*SubmissionDetail, error)
}

type ReviewDetail struct {
	ID              int
	SubmissionID    int
	AIModel         string
	OverallStatus   string
	AIConfidence    *float64
	ExecutionTimeMs *int
	CreatedAt       time.Time
	Feedbacks       []FeedbackDetail
}

type FeedbackDetail struct {
	ID              int
	ReviewID        int
	FeedbackType    string
	FilePath        *string
	LineStart       int
	LineEnd         *int
	CodeSnippet     string
	SuggestedFix    *string
	Description     string
	Severity        int
	IsResolved      bool
	TeacherComment  *string
	TeacherApproved *bool
}

type TeacherReviewInput struct {
	Actions          []FeedbackActionInput
	TeacherFeedbacks []TeacherFeedbackInput
}

type FeedbackActionInput struct {
	FeedbackID      int
	TeacherApproved *bool
	TeacherComment  *string
}

type TeacherFeedbackInput struct {
	FeedbackType *string
	FilePath     *string
	LineStart    *int
	LineEnd      *int
	CodeSnippet  *string
	SuggestedFix *string
	Description  string
	Severity     int
}

type reviewUseCase struct {
	submissionRepo repository.SubmissionRepository
	reviewRepo     repository.ReviewRepository
	buildRepo      repository.BuildRepository
	taskRepo       repository.TaskRepository
	userRepo       repository.UserRepository
	aiService      service.AIService
	githubService  service.GitHubService
	buildService   service.BuildService
	buildEnabled   bool
	logger         *zap.Logger
}

func NewReviewUseCase(
	submissionRepo repository.SubmissionRepository,
	reviewRepo repository.ReviewRepository,
	buildRepo repository.BuildRepository,
	taskRepo repository.TaskRepository,
	userRepo repository.UserRepository,
	aiService service.AIService,
	githubService service.GitHubService,
	buildService service.BuildService,
	cfg *config.Config,
	logger *zap.Logger,
) ReviewUseCase {
	return &reviewUseCase{
		submissionRepo: submissionRepo,
		reviewRepo:     reviewRepo,
		buildRepo:      buildRepo,
		taskRepo:       taskRepo,
		userRepo:       userRepo,
		aiService:      aiService,
		githubService:  githubService,
		buildService:   buildService,
		buildEnabled:   cfg.Build.Enabled,
		logger:         logger,
	}
}

func (uc *reviewUseCase) ProcessPendingSubmissions(ctx context.Context) error {
	submissions, err := uc.submissionRepo.ClaimPendingSubmissions(ctx, 10)
	if err != nil {
		return fmt.Errorf("failed to claim pending submissions: %w", err)
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

			subCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			if err := uc.processSubmission(subCtx, sub); err != nil {
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

	err := uc.doProcessSubmission(ctx, submission)
	if err != nil {
		uc.logger.Error("Submission processing failed, reverting to pending",
			zap.Int("submission_id", submission.ID),
			zap.Error(err),
		)
		revertCtx, revertCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer revertCancel()
		if revertErr := uc.submissionRepo.UpdateStatus(revertCtx, submission.ID, domain.StatusPending); revertErr != nil {
			uc.logger.Error("Failed to revert submission status",
				zap.Int("submission_id", submission.ID),
				zap.Error(revertErr),
			)
		}
	}
	return err
}

func (uc *reviewUseCase) doProcessSubmission(ctx context.Context, submission *domain.Submission) error {
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

	var buildOutput *service.BuildOutput
	if uc.buildEnabled {
		bo, err := uc.buildService.BuildCodeSnippet(ctx, *submission.Code)
		if err != nil {
			uc.logger.Warn("Build step failed, continuing without build results",
				zap.Int("submission_id", submission.ID),
				zap.Error(err),
			)
		} else {
			buildOutput = bo
			uc.saveBuildResult(ctx, submission.ID, buildOutput)
		}
	}

	return uc.aiService.ReviewCode(ctx, submission.Code, task, criteria, buildOutput)
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

	var buildOutput *service.BuildOutput
	if uc.buildEnabled {
		bo, err := uc.buildService.BuildAndTest(ctx, repoPath)
		if err != nil {
			uc.logger.Warn("Build step failed, continuing without build results",
				zap.Int("submission_id", submission.ID),
				zap.Error(err),
			)
		} else {
			buildOutput = bo
			uc.saveBuildResult(ctx, submission.ID, buildOutput)
		}
	}

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

	return uc.aiService.ReviewGitHubProject(ctx, files, task, criteria, buildOutput)
}

func (uc *reviewUseCase) saveBuildResult(ctx context.Context, submissionID int, bo *service.BuildOutput) {
	buildResult := &domain.BuildResult{
		SubmissionID:    submissionID,
		CompileSuccess:  bo.BuildSuccess,
		AnalyzeOutput:   bo.AnalyzeOutput,
		BuildSuccess:    bo.BuildSuccess,
		BuildOutput:     bo.BuildOutput,
		TestOutput:      bo.TestOutput,
		TestsPassed:     bo.TestsPassed,
		ExecutionTimeMs: bo.ExecutionTimeMs,
	}

	id, err := uc.buildRepo.CreateBuildResult(ctx, buildResult)
	if err != nil {
		uc.logger.Error("Failed to save build result",
			zap.Int("submission_id", submissionID),
			zap.Error(err),
		)
		return
	}

	uc.logger.Info("Saved build result",
		zap.Int("submission_id", submissionID),
		zap.Int("build_result_id", id),
		zap.Bool("build_success", bo.BuildSuccess),
		zap.Bool("tests_passed", bo.TestsPassed),
	)
}

func (uc *reviewUseCase) saveReviewResult(ctx context.Context, submissionID int, result *service.CodeReviewResult) error {
	review := &domain.CodeReview{
		SubmissionID:    submissionID,
		AIModel:         uc.aiService.ModelName(),
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

func (uc *reviewUseCase) GetReviewBySubmissionID(ctx context.Context, submissionID int) (*ReviewDetail, error) {
	submission, err := uc.submissionRepo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}
	if submission == nil {
		return nil, ErrSubmissionNotFound
	}

	review, err := uc.reviewRepo.GetCodeReviewBySubmissionID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	if review == nil {
		return nil, ErrReviewNotFound
	}

	feedbacks, err := uc.reviewRepo.GetReviewFeedbackByReviewID(ctx, review.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedbacks: %w", err)
	}

	return buildReviewDetail(review, feedbacks), nil
}

func (uc *reviewUseCase) SubmitTeacherReview(ctx context.Context, submissionID int, req *TeacherReviewInput) (*ReviewDetail, error) {
	review, err := uc.reviewRepo.GetCodeReviewBySubmissionID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	if review == nil {
		return nil, ErrReviewNotFound
	}

	for _, action := range req.Actions {
		feedback, err := uc.reviewRepo.GetFeedbackByID(ctx, action.FeedbackID)
		if err != nil {
			return nil, fmt.Errorf("failed to get feedback: %w", err)
		}
		if feedback == nil || feedback.ReviewID != review.ID {
			return nil, fmt.Errorf("feedback %d does not belong to review", action.FeedbackID)
		}
		if err := uc.reviewRepo.UpdateFeedbackTeacherReview(ctx, action.FeedbackID, action.TeacherApproved, action.TeacherComment); err != nil {
			return nil, fmt.Errorf("failed to update feedback: %w", err)
		}
	}

	for _, tf := range req.TeacherFeedbacks {
		feedbackType := "improvement"
		if tf.FeedbackType != nil && *tf.FeedbackType != "" {
			feedbackType = *tf.FeedbackType
		}
		lineStart := 1
		if tf.LineStart != nil {
			lineStart = *tf.LineStart
		}
		codeSnippet := ""
		if tf.CodeSnippet != nil {
			codeSnippet = *tf.CodeSnippet
		}
		approved := true
		feedback := &domain.ReviewFeedback{
			ReviewID:        review.ID,
			FeedbackType:    feedbackType,
			FilePath:        tf.FilePath,
			LineStart:       lineStart,
			LineEnd:         tf.LineEnd,
			CodeSnippet:     codeSnippet,
			SuggestedFix:    tf.SuggestedFix,
			Description:     tf.Description,
			Severity:        tf.Severity,
			IsResolved:      false,
			TeacherApproved: &approved,
		}
		if err := uc.reviewRepo.CreateReviewFeedback(ctx, feedback); err != nil {
			return nil, fmt.Errorf("failed to create teacher feedback: %w", err)
		}
	}

	feedbacks, err := uc.reviewRepo.GetReviewFeedbackByReviewID(ctx, review.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedbacks: %w", err)
	}

	return buildReviewDetail(review, feedbacks), nil
}

func (uc *reviewUseCase) GradeSubmission(ctx context.Context, submissionID int, score float64, comment *string) (*SubmissionDetail, error) {
	submission, err := uc.submissionRepo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}
	if submission == nil {
		return nil, ErrSubmissionNotFound
	}

	if err := uc.submissionRepo.UpdateScore(ctx, submissionID, score); err != nil {
		return nil, fmt.Errorf("failed to update score: %w", err)
	}
	if err := uc.submissionRepo.UpdateStatus(ctx, submissionID, domain.StatusTeacherReviewed); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	submission.Score = &score
	submission.Status = domain.StatusTeacherReviewed

	user, err := uc.userRepo.GetByID(ctx, submission.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	return submissionToDetail(submission, user), nil
}

func buildReviewDetail(review *domain.CodeReview, feedbacks []*domain.ReviewFeedback) *ReviewDetail {
	details := make([]FeedbackDetail, 0, len(feedbacks))
	for _, f := range feedbacks {
		details = append(details, FeedbackDetail{
			ID:              f.ID,
			ReviewID:        f.ReviewID,
			FeedbackType:    f.FeedbackType,
			FilePath:        f.FilePath,
			LineStart:       f.LineStart,
			LineEnd:         f.LineEnd,
			CodeSnippet:     f.CodeSnippet,
			SuggestedFix:    f.SuggestedFix,
			Description:     f.Description,
			Severity:        f.Severity,
			IsResolved:      f.IsResolved,
			TeacherComment:  f.TeacherComment,
			TeacherApproved: f.TeacherApproved,
		})
	}
	return &ReviewDetail{
		ID:              review.ID,
		SubmissionID:    review.SubmissionID,
		AIModel:         review.AIModel,
		OverallStatus:   review.OverallStatus,
		AIConfidence:    review.AIConfidence,
		ExecutionTimeMs: review.ExecutionTimeMs,
		CreatedAt:       review.CreatedAt,
		Feedbacks:       details,
	}
}
