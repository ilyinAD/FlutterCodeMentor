package server

import (
	"go.uber.org/fx"
)

func FxModule() fx.Option {
	return fx.Module(
		"server",
		fx.Provide(NewServer),
		fx.Invoke(registerHooks),
	)
}
