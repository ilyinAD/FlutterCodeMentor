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
			func(cfg *config.Config, logger *zap.Logger) AIService {
				return NewAIService(cfg.DeepSeekAPIKey, cfg.DeepSeekAPIURL, logger)
			},
			func(logger *zap.Logger) GitHubService {
				return NewGitHubService(logger)
			},
		),
	)
}
