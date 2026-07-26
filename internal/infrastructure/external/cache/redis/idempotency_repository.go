package redis

import (
	"context"
	"time"

	"trading-core/internal/application/ports/output"
)

// IdempotencyRepository implements output.IdempotencyRepository on top of
// an atomic Redis `SET key value NX EX ttl`. It is a second binding of
// the same port the Postgres-backed internal/infrastructure/database/idempotency
// repository already satisfies — this one is purpose-scoped to the
// nats-publish dedup path (see provideQuantEventPublisherUseCase in
// internal/app/providers.go), not a replacement for the existing
// Postgres-backed store used by order execution and market-event
// processing. Being on the same Redis instance quant-engine's Inbox
// checks (via its own Redis client) is what makes the dedup genuinely
// shared across the two services, not just within trading-core.
type IdempotencyRepository struct {
	client *Client
}

func NewIdempotencyRepository(client *Client) *IdempotencyRepository {
	return &IdempotencyRepository{client: client}
}

var _ output.IdempotencyRepository = (*IdempotencyRepository)(nil)

func (r *IdempotencyRepository) Reserve(ctx context.Context, scope, key string, ttl time.Duration) (bool, error) {
	reserved, err := r.client.SetNX(ctx, scope+":"+key, "1", ttl)
	if err != nil {
		return false, err
	}
	return reserved, nil
}
