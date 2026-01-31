package usecase

import (
	"go.uber.org/fx"
)

func FxModule() fx.Option {
	return fx.Module(
		"usecase",
		fx.Provide(
			NewSubmissionUseCase,
			NewTaskUseCase,
			NewUserUseCase,
			NewCourseUseCase,
			NewReviewUseCase,
		),
	)
}
