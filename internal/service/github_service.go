package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

type GitHubService interface {
	CloneRepository(ctx context.Context, githubURL string) (string, error)
	GetDartFiles(repoPath string) ([]string, error)
	ReadFile(filePath string) (string, error)
	Cleanup(repoPath string) error
}

type githubService struct {
	logger  *zap.Logger
	tempDir string
}

func NewGitHubService(logger *zap.Logger) GitHubService {
	tempDir := filepath.Join(os.TempDir(), "flutter-code-mentor")
	os.MkdirAll(tempDir, 0755)

	return &githubService{
		logger:  logger,
		tempDir: tempDir,
	}
}

func (s *githubService) CloneRepository(ctx context.Context, githubURL string) (string, error) {
	s.logger.Info("Cloning GitHub repository", zap.String("url", githubURL))

	repoName := s.extractRepoName(githubURL)
	repoPath := filepath.Join(s.tempDir, repoName)

	if _, err := os.Stat(repoPath); err == nil {
		s.logger.Info("Repository already exists, removing old clone", zap.String("path", repoPath))
		os.RemoveAll(repoPath)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", githubURL, repoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Failed to clone repository",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	s.logger.Info("Repository cloned successfully", zap.String("path", repoPath))
	return repoPath, nil
}

func (s *githubService) GetDartFiles(repoPath string) ([]string, error) {
	s.logger.Info("Searching for Dart files", zap.String("path", repoPath))

	var dartFiles []string

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dirName := info.Name()
			if dirName == ".git" || dirName == "build" || dirName == ".dart_tool" || dirName == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".dart") {
			relPath, err := filepath.Rel(repoPath, path)
			if err != nil {
				return err
			}
			dartFiles = append(dartFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	s.logger.Info("Found Dart files", zap.Int("count", len(dartFiles)))
	return dartFiles, nil
}

func (s *githubService) ReadFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

func (s *githubService) Cleanup(repoPath string) error {
	s.logger.Info("Cleaning up repository", zap.String("path", repoPath))
	return os.RemoveAll(repoPath)
}

func (s *githubService) extractRepoName(githubURL string) string {
	parts := strings.Split(strings.TrimSuffix(githubURL, ".git"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repo"
}
