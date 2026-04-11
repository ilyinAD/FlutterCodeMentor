package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"go.uber.org/zap"
)

type AIService interface {
	ReviewCode(ctx context.Context, code *string, task *domain.Task, criteria []*domain.TaskCriteria, buildOutput *BuildOutput) (*CodeReviewResult, error)
	ReviewGitHubProject(ctx context.Context, files map[string]string, task *domain.Task, criteria []*domain.TaskCriteria, buildOutput *BuildOutput) (*CodeReviewResult, error)
	ModelName() string
}

type aiService struct {
	client LLMClient
	logger *zap.Logger
}

func NewAIService(client LLMClient, logger *zap.Logger) AIService {
	return &aiService{
		client: client,
		logger: logger,
	}
}

type CodeReviewResult struct {
	OverallStatus   string
	AIConfidence    float64
	ExecutionTimeMs int
	Feedbacks       []FeedbackItem
}

type FeedbackItem struct {
	FeedbackType string
	FilePath     string
	LineStart    int
	LineEnd      int
	CodeSnippet  string
	SuggestedFix string
	Description  string
	Severity     int
}

type aiReviewResponse struct {
	OverallStatus string         `json:"overall_status"`
	Confidence    float64        `json:"confidence"`
	Feedbacks     []feedbackJSON `json:"feedbacks"`
}

type feedbackJSON struct {
	Type         string `json:"type"`
	FilePath     string `json:"file_path"`
	LineStart    int    `json:"line_start"`
	LineEnd      int    `json:"line_end"`
	CodeSnippet  string `json:"code_snippet"`
	SuggestedFix string `json:"suggested_fix"`
	Description  string `json:"description"`
	Severity     int    `json:"severity"`
}

func (s *aiService) ModelName() string {
	return s.client.ModelName()
}

func (s *aiService) ReviewCode(ctx context.Context, code *string, task *domain.Task, criteria []*domain.TaskCriteria, buildOutput *BuildOutput) (*CodeReviewResult, error) {
	startTime := time.Now()

	s.logger.Info("Starting AI code review",
		zap.Int("code_length", len(*code)),
		zap.Int("criteria_count", len(criteria)),
	)

	prompt := s.buildPrompt(code, task, criteria, buildOutput)
	systemPrompt := "You are an expert Flutter/Dart code reviewer. Analyze code and provide structured feedback in JSON format."

	content, err := s.client.Chat(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	return s.parseReviewResponse(content, startTime)
}

func (s *aiService) ReviewGitHubProject(ctx context.Context, files map[string]string, task *domain.Task, criteria []*domain.TaskCriteria, buildOutput *BuildOutput) (*CodeReviewResult, error) {
	startTime := time.Now()

	s.logger.Info("Starting AI GitHub project review",
		zap.Int("files_count", len(files)),
		zap.Int("criteria_count", len(criteria)),
	)

	prompt := s.buildGitHubProjectPrompt(files, task, criteria, buildOutput)
	systemPrompt := "You are an expert Flutter/Dart code reviewer. Analyze Flutter/Dart projects and provide structured feedback in JSON format."

	content, err := s.client.Chat(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	return s.parseReviewResponse(content, startTime)
}

func (s *aiService) parseReviewResponse(content string, startTime time.Time) (*CodeReviewResult, error) {
	var aiReview aiReviewResponse
	if err := json.Unmarshal([]byte(content), &aiReview); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	executionTime := int(time.Since(startTime).Milliseconds())

	result := &CodeReviewResult{
		OverallStatus:   aiReview.OverallStatus,
		AIConfidence:    aiReview.Confidence,
		ExecutionTimeMs: executionTime,
		Feedbacks:       make([]FeedbackItem, 0, len(aiReview.Feedbacks)),
	}

	for _, fb := range aiReview.Feedbacks {
		result.Feedbacks = append(result.Feedbacks, FeedbackItem{
			FeedbackType: fb.Type,
			FilePath:     fb.FilePath,
			LineStart:    fb.LineStart,
			LineEnd:      fb.LineEnd,
			CodeSnippet:  fb.CodeSnippet,
			SuggestedFix: fb.SuggestedFix,
			Description:  fb.Description,
			Severity:     fb.Severity,
		})
	}

	s.logger.Info("AI review completed successfully",
		zap.String("overall_status", result.OverallStatus),
		zap.Float64("confidence", result.AIConfidence),
		zap.Int("execution_time_ms", executionTime),
		zap.Int("feedbacks_count", len(result.Feedbacks)),
	)

	return result, nil
}

func (s *aiService) buildBuildResultsSection(buildOutput *BuildOutput) string {
	if buildOutput == nil {
		return ""
	}

	buildStatus := "FAILED"
	if buildOutput.BuildSuccess {
		buildStatus = "PASSED"
	}
	analyzeStatus := "CLEAN"
	if !buildOutput.AnalyzeClean {
		analyzeStatus = "HAS ISSUES"
	}
	testStatus := "FAILED"
	if buildOutput.TestsPassed {
		testStatus = "PASSED"
	}

	return fmt.Sprintf(`

== Automated Build & Test Results (ground truth) ==
Build (compilation): %s
Build output:
%s

Static Analysis (dart analyze): %s
Analysis output:
%s

Tests: %s
Test output:
%s

IMPORTANT: These are REAL build/test results from running the code. Use them as ground truth.
- If build fails, the project does NOT compile — overall_status MUST be "failed".
- Static analysis issues (info/warning level) do NOT mean the code fails to compile. Use them to provide style and quality feedback.
- Tests: if tests exist and fail — report errors. If tests are absent and the task/criteria require them — report as missing. If tests are absent and NOT required by task/criteria — ignore.
`, buildStatus, buildOutput.BuildOutput, analyzeStatus, buildOutput.AnalyzeOutput, testStatus, buildOutput.TestOutput)
}

func (s *aiService) buildCriteriaSection(criteria []*domain.TaskCriteria) string {
	if len(criteria) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\nTask-specific criteria to evaluate:\n")
	for i, c := range criteria {
		mandatory := "Optional"
		if c.IsMandatory {
			mandatory = "Mandatory"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s, Weight: %d] %s: %s\n",
			i+1, mandatory, c.Weight, c.CriterionName, c.CriterionDescription))
	}
	sb.WriteString(`
Only include a feedback item with type "criterion_check" for criteria that are NOT met.
Do NOT include feedback for criteria that are satisfied.
In the "description" field, start with "FAIL: " followed by evidence of what is missing or wrong.
Use severity 5 for failed mandatory criteria, severity 2 for failed optional criteria.`)
	return sb.String()
}

func (s *aiService) buildPrompt(code *string, task *domain.Task, criteria []*domain.TaskCriteria, buildOutput *BuildOutput) string {
	criteriaSection := s.buildCriteriaSection(criteria)
	buildSection := s.buildBuildResultsSection(buildOutput)

	taskDescription := ""
	if task != nil {
		taskDescription = fmt.Sprintf("\n\nTask description:\n%s\n", task.Description)
	}

	return fmt.Sprintf(`Analyze the following Flutter/Dart code and provide a detailed code review.
%s%s%s
Code to review:
%s

Respond ONLY with JSON in this exact format:
{
  "overall_status": "passed|failed|needs_improvement",
  "confidence": 0.0-1.0,
  "feedbacks": [
    {
      "type": "critical_error|logic_error|style_issue|performance|security_risk|improvement|criterion_check",
      "line_start": 10,
      "line_end": 15,
      "code_snippet": "problematic code here",
      "suggested_fix": "corrected code here",
      "description": "detailed explanation",
      "severity": 1-5
    }
  ]
}

Rules:
- overall_status: "failed" if build fails or critical errors exist, "needs_improvement" if moderate issues, "passed" if production-ready.
- severity: 5=critical (blocks functionality), 4=major, 3=moderate, 2=minor/style, 1=suggestion.
- For task criteria: only include a "criterion_check" feedback for criteria that are NOT met. Do not report passed criteria.`, taskDescription, criteriaSection, buildSection, *code)
}

func (s *aiService) buildGitHubProjectPrompt(files map[string]string, task *domain.Task, criteria []*domain.TaskCriteria, buildOutput *BuildOutput) string {
	var filesContent strings.Builder
	filesContent.WriteString("Flutter/Dart project files:\n\n")

	for filePath, content := range files {
		fmt.Fprintf(&filesContent, "=== File: %s ===\n", filePath)
		filesContent.WriteString(content)
		filesContent.WriteString("\n\n")
	}

	criteriaSection := s.buildCriteriaSection(criteria)
	buildSection := s.buildBuildResultsSection(buildOutput)

	taskDescription := ""
	if task != nil {
		taskDescription = fmt.Sprintf("\n\nTask description:\n%s\n", task.Description)
	}

	return fmt.Sprintf(`Analyze the following Flutter/Dart project and provide a detailed code review.
%s%s%s
%s

Respond ONLY with JSON in this exact format:
{
  "overall_status": "passed|failed|needs_improvement",
  "confidence": 0.0-1.0,
  "feedbacks": [
    {
      "type": "critical_error|logic_error|style_issue|performance|security_risk|improvement|criterion_check",
      "file_path": "lib/main.dart",
      "line_start": 10,
      "line_end": 15,
      "code_snippet": "problematic code here",
      "suggested_fix": "corrected code here",
      "description": "detailed explanation",
      "severity": 1-5
    }
  ]
}

Rules:
- overall_status: "failed" if build fails or critical errors exist, "needs_improvement" if moderate issues, "passed" if production-ready.
- severity: 5=critical (blocks functionality), 4=major, 3=moderate, 2=minor/style, 1=suggestion.
- Always include "file_path" in each feedback item.
- For task criteria: only include a "criterion_check" feedback for criteria that are NOT met. Do not report passed criteria.`, taskDescription, criteriaSection, buildSection, filesContent.String())
}
