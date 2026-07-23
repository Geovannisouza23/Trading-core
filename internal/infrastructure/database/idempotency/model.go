// Package idempotency implements output.IdempotencyRepository against the
// idempotency_keys table.
package idempotency

import "time"

type model struct {
	Scope     string
	Key       string
	CreatedAt time.Time
	ExpiresAt time.Time
}
