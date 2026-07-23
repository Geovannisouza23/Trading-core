package main

import (
	"go.uber.org/fx"

	"trading-core/internal/app"
)

func main() {
	fx.New(
		app.WorkerOnlyModule(),
	).Run()
}
