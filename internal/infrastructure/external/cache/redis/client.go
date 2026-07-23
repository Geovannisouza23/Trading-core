// Package redis is a thin, functional wrapper around go-redis. Nothing in
// the current pipeline requires a cache yet; this exists as the documented
// extension point (e.g. for future market data or exchange-info caching).
package redis

import (
	"context"
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

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}
