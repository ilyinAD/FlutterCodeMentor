package repository

import (
	"go.uber.org/fx"
)

func FxModule() fx.Option {
	return fx.Module(
		"repository",
		fx.Provide(
			NewSubmissionRepository,
			NewTaskRepository,
			NewUserRepository,
			NewCourseRepository,
		),
	)
}
