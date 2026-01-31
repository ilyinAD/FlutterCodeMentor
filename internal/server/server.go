package server

import (
	"context"
	"fmt"

	"github.com/ilyin-ad/flutter-code-mentor/api"
	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"github.com/ilyin-ad/flutter-code-mentor/internal/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Server struct {
	echo   *echo.Echo
	cfg    *config.Config
	logger *zap.Logger
}

type Handlers struct {
	*handler.SubmissionHandler
	*handler.TaskHandler
	*handler.UserHandler
	*handler.CourseHandler
}

func NewServer(
	cfg *config.Config,
	submissionHandler *handler.SubmissionHandler,
	taskHandler *handler.TaskHandler,
	userHandler *handler.UserHandler,
	courseHandler *handler.CourseHandler,
	logger *zap.Logger,
) *Server {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	handlers := &Handlers{
		SubmissionHandler: submissionHandler,
		TaskHandler:       taskHandler,
		UserHandler:       userHandler,
		CourseHandler:     courseHandler,
	}

	api.RegisterHandlers(e, handlers)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "healthy",
		})
	})

	logger.Info("Server initialized successfully")

	return &Server{
		echo:   e,
		cfg:    cfg,
		logger: logger,
	}
}

func registerHooks(lc fx.Lifecycle, server *Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf(":%s", server.cfg.Server.Port)
			server.logger.Info("Starting HTTP server",
				zap.String("address", addr),
			)
			go func() {
				if err := server.echo.Start(addr); err != nil {
					server.logger.Info("Shutting down the server")
				}
			}()
			server.logger.Info("Server started successfully",
				zap.String("address", addr),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.logger.Info("Stopping HTTP server")
			return server.echo.Shutdown(ctx)
		},
	})
}
