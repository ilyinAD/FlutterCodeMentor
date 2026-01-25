package handler

import (
	"go.uber.org/fx"
)

func FxModule() fx.Option {
	return fx.Module(
		"handler",
		fx.Provide(
			NewSubmissionHandler,
			NewTaskHandler,
			NewUserHandler,
			NewCourseHandler,
		),
	)
}
