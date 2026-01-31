package main

import (
	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"go.uber.org/fx"
)

func main() {
	config.LoadEnv(".env")

	fx.New(BuildApp()).Run()
}
