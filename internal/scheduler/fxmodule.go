package scheduler

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func FxModule() fx.Option {
	return fx.Module(
		"scheduler",
		fx.Provide(NewScheduler),
		fx.Invoke(func(lc fx.Lifecycle, s *Scheduler, logger *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					logger.Info("Starting code review scheduler")
					return s.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					logger.Info("Stopping code review scheduler")
					return s.Stop()
				},
			})
		}),
	)
}
