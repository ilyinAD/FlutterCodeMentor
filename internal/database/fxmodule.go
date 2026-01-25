package database

import (
	"context"

	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"github.com/ilyin-ad/flutter-code-mentor/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

func FxModule() fx.Option {
	return fx.Module(
		"database",
		fx.Provide(NewPostgresPoolWithConfig),
		fx.Provide(NewPgxPoolConfig),
		fx.Invoke(registerHooks),
		migrations.FxModule(),
	)
}

func registerHooks(lc fx.Lifecycle, pool *pgxpool.Pool) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			Close(pool)
			return nil
		},
	})
}

func NewPostgresPoolWithConfig(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx := context.Background()
	return NewPostgresPool(ctx, cfg)
}
