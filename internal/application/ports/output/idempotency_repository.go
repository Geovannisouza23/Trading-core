package output

import (
	"context"
	"time"
)

// IdempotencyRepository provides a generic, atomic "claim this key once"
// primitive used by consumers and use cases that must never process the
// same logical operation twice (e.g. a candle-closed message redelivered by
// the broker, or a second concurrent attempt to execute the same signal).
type IdempotencyRepository interface {
	// Reserve attempts to atomically claim (scope, key). It reports true
	// only for the caller that won the race; every other caller gets
	// false with no error. Implementations must rely on a unique
	// constraint, never a read-then-write check.
	Reserve(ctx context.Context, scope, key string, ttl time.Duration) (reserved bool, err error)
}
