package main

import (
	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"github.com/ilyin-ad/flutter-code-mentor/internal/database"
	"github.com/ilyin-ad/flutter-code-mentor/internal/handler"
	"github.com/ilyin-ad/flutter-code-mentor/internal/repository"
	"github.com/ilyin-ad/flutter-code-mentor/internal/scheduler"
	"github.com/ilyin-ad/flutter-code-mentor/internal/server"
	"github.com/ilyin-ad/flutter-code-mentor/internal/service"
	"github.com/ilyin-ad/flutter-code-mentor/internal/usecase"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func BuildApp() fx.Option {
	return fx.Options(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),

		config.FxModule(),
		database.FxModule(),
		repository.FxModule(),
		service.FxModule(),
		usecase.FxModule(),
		handler.FxModule(),
		server.FxModule(),
		scheduler.FxModule(),

		fx.Provide(func() (*zap.Logger, error) {
			return zap.NewDevelopment()
		}),
	)
}
