package migration

import (
	"context"
	"fmt"
	"strconv"
)

// Execute dispatches a CLI-style migration command ("up", "down", "version",
// "force <version>") against runner, writing human-readable output to the
// provided log function. It is the single piece of logic shared by
// cmd/migrate and any test harness that needs to drive migrations.
func Execute(ctx context.Context, runner *Runner, args []string, log func(format string, a ...any)) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: migrate <up|down|version|force> [args]")
	}

	switch args[0] {
	case "up":
		if err := runner.Up(ctx); err != nil {
			return err
		}
		version, err := runner.Version(ctx)
		if err != nil {
			return err
		}
		log("migrated up to version %d", version)
		return nil

	case "down":
		steps := 1
		if len(args) > 1 {
			parsed, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid step count %q: %w", args[1], err)
			}
			steps = parsed
		}
		if err := runner.Down(ctx, steps); err != nil {
			return err
		}
		version, err := runner.Version(ctx)
		if err != nil {
			return err
		}
		log("migrated down to version %d", version)
		return nil

	case "version":
		version, err := runner.Version(ctx)
		if err != nil {
			return err
		}
		log("current version: %d", version)
		return nil

	case "force":
		if len(args) < 2 {
			return fmt.Errorf("usage: migrate force <version>")
		}
		version, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version %q: %w", args[1], err)
		}
		if err := runner.Force(ctx, version); err != nil {
			return err
		}
		log("forced version to %d", version)
		return nil

	default:
		return fmt.Errorf("unknown migrate command %q (want up|down|version|force)", args[0])
	}
}
