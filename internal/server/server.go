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
)

type Server struct {
	echo *echo.Echo
	cfg  *config.Config
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

	return &Server{
		echo: e,
		cfg:  cfg,
	}
}

func registerHooks(lc fx.Lifecycle, server *Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf(":%s", server.cfg.Server.Port)
			go func() {
				if err := server.echo.Start(addr); err != nil {
					server.echo.Logger.Info("shutting down the server")
				}
			}()
			server.echo.Logger.Infof("Server started on %s", addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.echo.Shutdown(ctx)
		},
	})
}
