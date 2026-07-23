// Command migrate applies or reverts SQL migrations. It intentionally does
// not use Uber Fx: it needs only configuration and the migration runner,
// and running it as a plain sequential program keeps its exit code and
// stdout output simple to script in CI/CD and docker-compose init steps.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"trading-core/internal/config"
	"trading-core/internal/infrastructure/database"
	"trading-core/internal/infrastructure/database/migration"
	"trading-core/internal/infrastructure/database/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "loading config:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connecting to database:", err)
		os.Exit(1)
	}
	defer pool.Close()

	runner, err := migration.NewRunner(pool, migrations.FS)
	if err != nil {
		fmt.Fprintln(os.Stderr, "building migration runner:", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	err = migration.Execute(ctx, runner, os.Args[1:], func(format string, a ...any) {
		logger.Info(fmt.Sprintf(format, a...))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}
