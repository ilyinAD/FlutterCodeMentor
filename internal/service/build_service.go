package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"go.uber.org/zap"
)

type BuildService interface {
	BuildAndTest(ctx context.Context, projectPath string) (*BuildOutput, error)
	BuildCodeSnippet(ctx context.Context, code string) (*BuildOutput, error)
}

type BuildOutput struct {
	BuildSuccess    bool
	BuildOutput     string
	AnalyzeClean    bool
	AnalyzeOutput   string
	TestOutput      string
	TestsPassed     bool
	ExecutionTimeMs int
}

type buildService struct {
	dockerImage string
	timeout     time.Duration
	logger      *zap.Logger
}

func NewBuildService(cfg *config.Config, logger *zap.Logger) BuildService {
	return &buildService{
		dockerImage: cfg.Build.DockerImage,
		timeout:     time.Duration(cfg.Build.TimeoutSec) * time.Second,
		logger:      logger,
	}
}

func (s *buildService) BuildAndTest(ctx context.Context, projectPath string) (*BuildOutput, error) {
	startTime := time.Now()

	isFlutter := s.isFlutterProject(projectPath)

	script := s.buildScript(isFlutter)

	s.logger.Info("Starting Docker build/test",
		zap.String("project_path", projectPath),
		zap.Bool("is_flutter", isFlutter),
		zap.String("docker_image", s.dockerImage),
		zap.String("script", script),
		zap.String("docker_image", s.dockerImage),
	)

	buildCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(buildCtx, "docker", "run", "--rm",
		"--memory=4096m", "--cpus=2", "--pids-limit=512",
		"-v", projectPath+":/source:ro",
		s.dockerImage,
		"sh", "-c", "cp -r /source /app && cd /app && "+script,
	)

	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)

	executionTime := int(time.Since(startTime).Milliseconds())

	if buildCtx.Err() == context.DeadlineExceeded {
		return &BuildOutput{
			BuildSuccess:    false,
			AnalyzeOutput:   "Build timed out",
			ExecutionTimeMs: executionTime,
		}, nil
	}

	if err != nil && output == "" {
		return nil, fmt.Errorf("docker execution failed: %w", err)
	}

	result := s.parseOutput(output)
	result.ExecutionTimeMs = executionTime

	s.logger.Info("Docker build/test completed",
		zap.Bool("build_success", result.BuildSuccess),
		zap.Bool("tests_passed", result.TestsPassed),
		zap.Int("execution_time_ms", executionTime),
	)

	return result, nil
}

func (s *buildService) BuildCodeSnippet(ctx context.Context, code string) (*BuildOutput, error) {
	tmpDir, err := os.MkdirTemp("", "flutter-code-snippet-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	libDir := filepath.Join(tmpDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lib directory: %w", err)
	}

	pubspec := `name: submission_check
environment:
  sdk: ">=3.0.0 <4.0.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pubspec.yaml"), []byte(pubspec), 0644); err != nil {
		return nil, fmt.Errorf("failed to write pubspec.yaml: %w", err)
	}

	if err := os.WriteFile(filepath.Join(libDir, "main.dart"), []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write main.dart: %w", err)
	}

	return s.BuildAndTest(ctx, tmpDir)
}

func (s *buildService) isFlutterProject(projectPath string) bool {
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")
	data, err := os.ReadFile(pubspecPath)
	if err != nil {
		return false
	}
	content := string(data)
	return strings.Contains(content, "flutter:") && strings.Contains(content, "sdk: flutter")
}

func (s *buildService) buildScript(isFlutter bool) string {
	pubCmd := "dart"
	analyzeCmd := "dart analyze"
	buildCmd := "dart compile kernel lib/main.dart -o /tmp/app.dill"
	testCmd := "dart test"
	if isFlutter {
		pubCmd = "flutter"
		analyzeCmd = "flutter analyze"
		buildCmd = "flutter build bundle"
		testCmd = "flutter test"
	}

	return fmt.Sprintf(`
echo "===PUB_GET_START===";
if [ ! -d "android" ] && [ "%s" = "flutter" ]; then
  flutter create --no-overwrite . 2>&1;
fi
%s pub get 2>&1;
PUB_EXIT=$?;
echo "===PUB_GET_EXIT=$PUB_EXIT===";

echo "===BUILD_START===";
%s 2>&1;
BUILD_EXIT=$?;
echo "===BUILD_EXIT=$BUILD_EXIT===";

echo "===ANALYZE_START===";
if [ $BUILD_EXIT -eq 0 ]; then
  %s 2>&1;
  ANALYZE_EXIT=$?;
else
  echo "Skipped: build failed";
  ANALYZE_EXIT=-1;
fi
echo "===ANALYZE_EXIT=$ANALYZE_EXIT===";

echo "===TEST_START===";
if [ $BUILD_EXIT -eq 0 ]; then
  %s 2>&1;
  TEST_EXIT=$?;
else
  echo "Skipped: build failed";
  TEST_EXIT=-1;
fi
echo "===TEST_EXIT=$TEST_EXIT===";
`, pubCmd, pubCmd, buildCmd, analyzeCmd, testCmd)
}

func (s *buildService) parseOutput(output string) *BuildOutput {
	result := &BuildOutput{}

	buildOutput, buildExit := extractSection(output, "BUILD")
	result.BuildOutput = truncateOutput(buildOutput, 4000)
	result.BuildSuccess = buildExit == 0

	analyzeOutput, analyzeExit := extractSection(output, "ANALYZE")
	result.AnalyzeOutput = truncateOutput(filterAnalyzeOutput(analyzeOutput), 4000)
	result.AnalyzeClean = analyzeExit == 0

	testOutput, testExit := extractSection(output, "TEST")
	result.TestOutput = truncateOutput(testOutput, 4000)
	result.TestsPassed = testExit == 0

	return result
}

func filterAnalyzeOutput(output string) string {
	idx := strings.Index(output, "Analyzing")
	if idx == -1 {
		return output
	}
	return strings.TrimSpace(output[idx:])
}

func extractSection(output, name string) (string, int) {
	startMarker := fmt.Sprintf("===%s_START===", name)
	exitMarker := fmt.Sprintf("===%s_EXIT=", name)

	startIdx := strings.Index(output, startMarker)
	exitIdx := strings.Index(output, exitMarker)

	if startIdx == -1 || exitIdx == -1 {
		return "", -1
	}

	sectionContent := output[startIdx+len(startMarker) : exitIdx]
	sectionContent = strings.TrimSpace(sectionContent)

	exitCode := -1
	exitLine := output[exitIdx:]
	if endIdx := strings.Index(exitLine, "===\n"); endIdx != -1 {
		codeStr := exitLine[len(exitMarker):endIdx]
		fmt.Sscanf(codeStr, "%d", &exitCode)
	} else if endIdx := strings.Index(exitLine, "==="); endIdx > len(exitMarker) {
		codeStr := exitLine[len(exitMarker):endIdx]
		fmt.Sscanf(codeStr, "%d", &exitCode)
	}

	return sectionContent, exitCode
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
