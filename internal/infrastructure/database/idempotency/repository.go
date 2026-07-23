package idempotency

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.IdempotencyRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.IdempotencyRepository = (*Repository)(nil)

// Reserve relies entirely on the (scope, key) primary key constraint for
// atomicity: concurrent callers racing for the same key will have exactly
// one succeed, which is what makes this safe without an explicit lock.
func (r *Repository) Reserve(ctx context.Context, scope, key string, ttl time.Duration) (bool, error) {
	now := time.Now().UTC()
	tag, err := transaction.Executor(ctx, r.pool).Exec(ctx, reserveQuery, scope, key, now, now.Add(ttl))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
