// Package redis is a thin, functional wrapper around go-redis. Backs two
// real use cases: market.MarketDataCache (candle caching, see
// market_data_cache.go) and a Redis-backed output.IdempotencyRepository
// (cross-service dedup with quant-engine's NATS consumers, see
// idempotency_repository.go) — both optional, both fall back to their
// pre-existing behavior (no cache; Postgres-only idempotency) if Redis is
// unreachable.
package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *goredis.Client
}

func NewClient(addr, password string, db int) *Client {
	return &Client{rdb: goredis.NewClient(&goredis.Options{Addr: addr, Password: password, DB: db})}
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// SetNX atomically claims key (via Redis's own NX flag — no
// read-then-write race), reporting true only for the caller that won.
// Used for idempotency reservation, the same guarantee
// output.IdempotencyRepository.Reserve documents for its Postgres-backed
// implementation.
func (c *Client) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

// ErrCacheMiss is returned by MarketDataCache.GetCandles when the key is
// absent — distinguishable from a real Redis error so callers can treat
// "not cached yet" as the expected, common case, not a failure to log.
var ErrCacheMiss = errors.New("redis: cache miss")
