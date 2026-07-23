// Package logging builds the application's structured logger from
// validated configuration.
package logging

import (
	"log/slog"
	"os"

	"trading-core/internal/config"
)

// NewLogger builds a slog.Logger configured per ObservabilityConfig. JSON
// output is the default so logs are directly consumable by log aggregators;
// secrets are never passed to this logger (config.Redacted() variants exist
// for exactly this reason).
func NewLogger(cfg config.ObservabilityConfig) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}

	var handler slog.Handler
	if cfg.LogFormat == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
	)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
