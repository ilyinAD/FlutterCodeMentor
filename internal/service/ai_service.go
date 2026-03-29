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

	compileStatus := "FAILED"
	if buildOutput.CompileSuccess {
		compileStatus = "PASSED"
	}
	testStatus := "FAILED"
	if buildOutput.TestsPassed {
		testStatus = "PASSED"
	}

	return fmt.Sprintf(`

== Automated Build & Test Results (ground truth) ==
Static Analysis / Compilation (dart analyze): %s
Analysis output:
%s

Tests: %s
Test output:
%s

IMPORTANT: These are REAL build/test results from running the code. Use them as ground truth.
- If static analysis reports errors, the code does NOT compile — overall_status MUST be "failed".
- Tests: if tests exist and fail — report errors. If tests are absent and the task/criteria require them — report as missing. If tests are absent and NOT required by task/criteria — ignore.
- Tests are only checked when analysis passes.
`, compileStatus, buildOutput.AnalyzeOutput, testStatus, buildOutput.TestOutput)
}

func (s *aiService) buildCriteriaSection(criteria []*domain.TaskCriteria) string {
	if len(criteria) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\nTask-specific criteria to evaluate (you MUST report PASS/FAIL for EACH):\n")
	for i, c := range criteria {
		mandatory := "Optional"
		if c.IsMandatory {
			mandatory = "Mandatory"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s, Weight: %d] %s: %s\n",
			i+1, mandatory, c.Weight, c.CriterionName, c.CriterionDescription))
	}
	sb.WriteString(`
For EACH criterion above, you MUST include a feedback item with type "criterion_check".
In the "description" field, start with "PASS: " or "FAIL: " followed by evidence.
Use severity 5 for failed mandatory criteria, severity 2 for failed optional criteria, severity 1 for passed criteria.`)
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

Provide your response in the following JSON format:
{
  "overall_status": "passed|failed|needs_improvement",
  "confidence": 0.95,
  "feedbacks": [
    {
      "type": "critical_error|logic_error|style_issue|performance|security_risk|improvement|criterion_check",
      "line_start": 10,
      "line_end": 15,
      "code_snippet": "problematic code here",
      "suggested_fix": "corrected code here",
      "description": "detailed explanation of the issue",
      "severity": 1-5
    }
  ]
}

Review criteria:
1. **Critical Errors**: Syntax errors, null safety violations, type mismatches
2. **Logic Errors**: Incorrect business logic, potential runtime errors
3. **Style Issues**: Code formatting, naming conventions, Flutter best practices
4. **Performance**: Inefficient algorithms, unnecessary rebuilds, memory leaks
5. **Security**: Exposed sensitive data, insecure API calls
6. **Improvements**: Better patterns, code organization, widget composition

Severity levels:
- 5: Critical (blocks functionality)
- 4: Major (significant impact)
- 3: Moderate (noticeable issue)
- 2: Minor (cosmetic or style)
- 1: Suggestion (optional improvement)

Overall status:
- "passed": Code is production-ready with minor or no issues
- "needs_improvement": Code works but has moderate issues
- "failed": Code has critical errors or major problems

Provide confidence as a decimal between 0 and 1.

IMPORTANT: Pay special attention to the task-specific criteria listed above. For each criterion, include a feedback item with type "criterion_check".`, taskDescription, criteriaSection, buildSection, *code)
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

Provide your response in the following JSON format:
{
  "overall_status": "passed|failed|needs_improvement",
  "confidence": 0.95,
  "feedbacks": [
    {
      "type": "critical_error|logic_error|style_issue|performance|security_risk|improvement|criterion_check",
      "file_path": "lib/main.dart",
      "line_start": 10,
      "line_end": 15,
      "code_snippet": "problematic code here",
      "suggested_fix": "corrected code here",
      "description": "detailed explanation of the issue",
      "severity": 1-5
    }
  ]
}

Review criteria:
1. **Critical Errors**: Syntax errors, null safety violations, type mismatches
2. **Logic Errors**: Incorrect business logic, potential runtime errors
3. **Style Issues**: Code formatting, naming conventions, Flutter best practices
4. **Performance**: Inefficient algorithms, unnecessary rebuilds, memory leaks
5. **Security**: Exposed sensitive data, insecure API calls
6. **Improvements**: Better patterns, code organization, widget composition
7. **Project Structure**: Proper file organization, separation of concerns

Severity levels:
- 5: Critical (blocks functionality)
- 4: Major (significant impact)
- 3: Moderate (noticeable issue)
- 2: Minor (cosmetic or style)
- 1: Suggestion (optional improvement)

Overall status:
- "passed": Code is production-ready with minor or no issues
- "needs_improvement": Code works but has moderate issues
- "failed": Code has critical errors or major problems

Provide confidence as a decimal between 0 and 1.
IMPORTANT: Always include "file_path" field in each feedback item to indicate which file the issue is in.
IMPORTANT: Pay special attention to the task-specific criteria listed above. For each criterion, include a feedback item with type "criterion_check".`, taskDescription, criteriaSection, buildSection, filesContent.String())
}
