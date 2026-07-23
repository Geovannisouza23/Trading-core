//go:build integration

// Package integration_test exercises real PostgreSQL behavior (via
// Testcontainers) that unit tests, which never touch a database, cannot:
// migrations, optimistic locking, unique constraints, and the
// transaction manager. Run with `make test-integration` (requires Docker).
package integration_test

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/tests/fixtures"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return fixtures.NewPostgresPool(t)
}
