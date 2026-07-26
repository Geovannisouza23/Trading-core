package output

import (
	"context"
	"time"
)

// ActivityPublisher publishes quant.activity.changed — a fixed-subject,
// fire-and-forget signal on the same shared NATS/JetStream stream
// QuantEventPublisher uses, but for a different audience: the Python
// training-pipeline's idle_watcher, not quant-engine. It reports whether
// the system currently has zero open positions and zero in-flight
// orders, so CPU-heavy offline work (model training) only ever runs
// when it can't compete with live order execution for resources.
type ActivityPublisher interface {
	PublishActivityChanged(ctx context.Context, idle bool, changedAt time.Time) error
}
