package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckHealth is used by the /ready endpoint to verify the database is
// actually reachable, not just that the pool was constructed successfully
// at startup.
func CheckHealth(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
