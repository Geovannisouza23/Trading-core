// Package order subscribes to order lifecycle events on the EventBus and
// records metrics. It carries no business logic — order state transitions
// themselves are owned entirely by domain/order and
// application/usecase.ExecuteApprovedOrder.
package order

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"trading-core/internal/application/ports/output"
)

// Counters is the minimal metrics surface this consumer needs, typed
// directly against the OTel metric API so this package does not depend on
// internal/infrastructure/observability/metrics. Only Created and
// Submitted are wired here: those are the only two order-lifecycle events
// the outbox currently publishes that have a matching counter in spec
// section 28 (orders_created_total, orders_submitted_total). Rejected/
// Failed/Unknown are recorded where they actually happen, inside
// ExecuteApprovedOrder, since those states are not outbox-published.
type Counters struct {
	Created   metric.Int64Counter
	Submitted metric.Int64Counter
}

type Consumer struct {
	counters Counters
}

func NewConsumer(counters Counters) *Consumer {
	return &Consumer{counters: counters}
}

// Subscribe registers a handler per order event name. Call once at startup.
func (c *Consumer) Subscribe(bus output.EventBus) {
	bus.Subscribe("OrderCreated", c.record(c.counters.Created))
	bus.Subscribe("OrderSubmitted", c.record(c.counters.Submitted))
}

func (c *Consumer) record(counter metric.Int64Counter) output.EventHandler {
	return func(ctx context.Context, event output.DomainEvent) error {
		if counter != nil {
			counter.Add(ctx, 1)
		}
		return nil
	}
}
