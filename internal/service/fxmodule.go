package service

import (
	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func FxModule() fx.Option {
	return fx.Module(
		"service",
		fx.Provide(
			func(cfg *config.Config, logger *zap.Logger) LLMClient {
				var client LLMClient
				if cfg.AI.Provider == "stub" {
					logger.Info("Using stub LLM client")
					client = NewStubLLMClient(logger)
				} else {
					client = NewOpenAICompatibleClient(cfg.AI.APIKey, cfg.AI.BaseURL, cfg.AI.Model, logger)
				}
				if cfg.AI.LogPrompts {
					logger.Info("Prompt logging enabled", zap.String("log_file", cfg.AI.PromptLogFile))
					client = NewLoggingLLMClient(client, cfg.AI.PromptLogFile, logger)
				}
				return client
			},
			func(client LLMClient, logger *zap.Logger) AIService {
				return NewAIService(client, logger)
			},
			func(cfg *config.Config, logger *zap.Logger) BuildService {
				return NewBuildService(cfg, logger)
			},
			func(logger *zap.Logger) GitHubService {
				return NewGitHubService(logger)
			},
		),
	)
}
