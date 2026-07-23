// Package fixtures holds test-only helpers shared across tests/integration
// and tests/failure. It is a regular (non "_test.go") package so both can
// import it; nothing under cmd/ or internal/ ever does.
package fixtures

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"trading-core/internal/config"
	"trading-core/internal/infrastructure/database"
	"trading-core/internal/infrastructure/database/migration"
	"trading-core/internal/infrastructure/database/migrations"
)

// NewPostgresPool starts a fresh Postgres container, applies every
// migration, and returns a pool built through the same database.NewPool
// path cmd/api and cmd/migrate use. The container and pool are torn down
// automatically via t.Cleanup.
func NewPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("trading_core"),
		postgres.WithUsername("trading"),
		postgres.WithPassword("trading"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminating postgres container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("container mapped port: %v", err)
	}
	port, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		t.Fatalf("parsing mapped port: %v", err)
	}

	dbCfg := config.DatabaseConfig{
		Host:            host,
		Port:            port,
		User:            "trading",
		Password:        "trading",
		Name:            "trading_core",
		SSLMode:         "disable",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		ConnectTimeout:  10 * time.Second,
	}

	pool, err := database.NewPool(ctx, dbCfg)
	if err != nil {
		t.Fatalf("connecting pool: %v", err)
	}
	t.Cleanup(pool.Close)

	runner, err := migration.NewRunner(pool, migrations.FS)
	if err != nil {
		t.Fatalf("building migration runner: %v", err)
	}
	if err := runner.Up(ctx); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	return pool
}
