package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"go.uber.org/zap"
)

type AIService interface {
	ReviewCode(ctx context.Context, code *string, task *domain.Task, criteria []*domain.TaskCriteria) (*CodeReviewResult, error)
	ReviewGitHubProject(ctx context.Context, files map[string]string, task *domain.Task, criteria []*domain.TaskCriteria) (*CodeReviewResult, error)
}

type aiService struct {
	apiKey string
	apiURL string
	client *http.Client
	logger *zap.Logger
}

func NewAIService(apiKey, apiURL string, logger *zap.Logger) AIService {
	return &aiService{
		apiKey: apiKey,
		apiURL: apiURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
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

type deepseekRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
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

func (s *aiService) ReviewCode(ctx context.Context, code *string, task *domain.Task, criteria []*domain.TaskCriteria) (*CodeReviewResult, error) {
	startTime := time.Now()

	s.logger.Info("Starting AI code review",
		zap.Int("code_length", len(*code)),
		zap.Int("criteria_count", len(criteria)),
	)

	prompt := s.buildPrompt(code, task, criteria)

	reqBody := deepseekRequest{
		Model: "deepseek-chat",
		Messages: []message{
			{
				Role:    "system",
				Content: "You are an expert Flutter/Dart code reviewer. Analyze code and provide structured feedback in JSON format.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	s.logger.Info("Sending request to AI API",
		zap.String("url", s.apiURL),
		zap.String("model", "deepseek-chat"),
	)

	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	s.logger.Info("Received response from AI API",
		zap.Int("status_code", resp.StatusCode),
	)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deepseekResp deepseekResponse
	if err := json.NewDecoder(resp.Body).Decode(&deepseekResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(deepseekResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	content := deepseekResp.Choices[0].Message.Content

	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

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

	s.logger.Info("AI code review completed successfully",
		zap.String("overall_status", result.OverallStatus),
		zap.Float64("confidence", result.AIConfidence),
		zap.Int("execution_time_ms", executionTime),
		zap.Int("feedbacks_count", len(result.Feedbacks)),
	)

	return result, nil
}

func (s *aiService) buildPrompt(code *string, task *domain.Task, criteria []*domain.TaskCriteria) string {
	criteriaSection := ""
	if len(criteria) > 0 {
		criteriaSection = "\n\nTask-specific criteria to check:\n"
		for i, c := range criteria {
			mandatory := "Optional"
			if c.IsMandatory {
				mandatory = "Mandatory"
			}
			criteriaSection += fmt.Sprintf("%d. [%s, Weight: %d] %s: %s\n",
				i+1, mandatory, c.Weight, c.CriterionName, c.CriterionDescription)
		}
	}

	taskDescription := ""
	if task != nil {
		taskDescription = fmt.Sprintf("\n\nTask description:\n%s\n", task.Description)
	}

	return fmt.Sprintf(`Analyze the following Flutter/Dart code and provide a detailed code review.
%s%s
Code to review:
%s

Provide your response in the following JSON format:
{
  "overall_status": "passed|failed|needs_improvement",
  "confidence": 0.95,
  "feedbacks": [
    {
      "type": "critical_error|logic_error|style_issue|performance|security_risk|improvement",
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

IMPORTANT: Pay special attention to the task-specific criteria listed above. Check if the code meets these requirements and include them in your feedback if they are not satisfied.`, taskDescription, criteriaSection, *code)
}

func (s *aiService) ReviewGitHubProject(ctx context.Context, files map[string]string, task *domain.Task, criteria []*domain.TaskCriteria) (*CodeReviewResult, error) {
	startTime := time.Now()

	s.logger.Info("Starting AI GitHub project review",
		zap.Int("files_count", len(files)),
		zap.Int("criteria_count", len(criteria)),
	)

	prompt := s.buildGitHubProjectPrompt(files, task, criteria)

	reqBody := deepseekRequest{
		Model: "deepseek-chat",
		Messages: []message{
			{
				Role:    "system",
				Content: "You are an expert Flutter/Dart code reviewer. Analyze Flutter/Dart projects and provide structured feedback in JSON format.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	s.logger.Info("Sending request to AI API for GitHub project review",
		zap.String("url", s.apiURL),
		zap.String("model", "deepseek-chat"),
	)

	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	s.logger.Info("Received response from AI API",
		zap.Int("status_code", resp.StatusCode),
	)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deepseekResp deepseekResponse
	if err := json.NewDecoder(resp.Body).Decode(&deepseekResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(deepseekResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	content := deepseekResp.Choices[0].Message.Content

	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

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

	s.logger.Info("AI GitHub project review completed successfully",
		zap.String("overall_status", result.OverallStatus),
		zap.Float64("confidence", result.AIConfidence),
		zap.Int("execution_time_ms", executionTime),
		zap.Int("feedbacks_count", len(result.Feedbacks)),
	)

	return result, nil
}

func (s *aiService) buildGitHubProjectPrompt(files map[string]string, task *domain.Task, criteria []*domain.TaskCriteria) string {
	var filesContent strings.Builder
	filesContent.WriteString("Flutter/Dart project files:\n\n")

	for filePath, content := range files {
		filesContent.WriteString(fmt.Sprintf("=== File: %s ===\n", filePath))
		filesContent.WriteString(content)
		filesContent.WriteString("\n\n")
	}

	criteriaSection := ""
	if len(criteria) > 0 {
		criteriaSection = "\n\nTask-specific criteria to check:\n"
		for i, c := range criteria {
			mandatory := "Optional"
			if c.IsMandatory {
				mandatory = "Mandatory"
			}
			criteriaSection += fmt.Sprintf("%d. [%s, Weight: %d] %s: %s\n",
				i+1, mandatory, c.Weight, c.CriterionName, c.CriterionDescription)
		}
	}

	taskDescription := ""
	if task != nil {
		taskDescription = fmt.Sprintf("\n\nTask description:\n%s\n", task.Description)
	}

	return fmt.Sprintf(`Analyze the following Flutter/Dart project and provide a detailed code review.
%s%s
%s

Provide your response in the following JSON format:
{
  "overall_status": "passed|failed|needs_improvement",
  "confidence": 0.95,
  "feedbacks": [
    {
      "type": "critical_error|logic_error|style_issue|performance|security_risk|improvement",
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
IMPORTANT: Pay special attention to the task-specific criteria listed above. Check if the project meets these requirements and include them in your feedback if they are not satisfied.`, taskDescription, criteriaSection, filesContent.String())
}
