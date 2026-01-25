package migrations

import "go.uber.org/fx"

func FxModule() fx.Option {
	return fx.Options(
		fx.Invoke(Migrate),
	)
}
