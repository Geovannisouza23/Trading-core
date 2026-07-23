// Package failure_test injects the failure scenarios spec section 29.4
// calls out. Scenarios that need a real (but deliberately broken)
// PostgreSQL/HTTP endpoint run directly; scenarios that need a real,
// healthy PostgreSQL (e.g. version-conflict, double-processing) reuse the
// tests/integration Testcontainers fixtures and are covered there instead
// of being duplicated here.
package failure_test

import (
	"context"
	"testing"
	"time"

	"trading-core/internal/config"
	"trading-core/internal/infrastructure/database"
)

func TestDatabaseUnavailableAtStartupFailsFast(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:            "127.0.0.1",
		Port:            1, // reserved/unused port: connection must be refused immediately
		User:            "trading",
		Password:        "trading",
		Name:            "trading_core",
		SSLMode:         "disable",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		ConnectTimeout:  2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.NewPool(ctx, cfg)
	if err == nil {
		t.Fatal("expected an error connecting to an unreachable database, got nil")
	}
}
