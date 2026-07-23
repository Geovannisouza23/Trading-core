// Package outbox subscribes to the EventBus (fed by
// internal/infrastructure/database/outbox's worker) and forwards every
// dashboard-relevant domain event to the WebSocket hub. It carries no
// business logic: receive event, forward payload.
package outbox

import (
	"context"

	"trading-core/internal/application/ports/output"
)

// Broadcaster is the minimal surface this consumer needs from the
// WebSocket hub.
type Broadcaster interface {
	Broadcast(msgType string, payload any)
}

var dashboardEventNames = []string{
	"OrderCreated", "OrderSubmitted", "OrderPartiallyFilled", "OrderFilled", "OrderCancelled",
	"PositionOpened", "PositionUpdated", "PositionClosed",
	"SignalCreated", "RiskDecisionCreated",
	"CriticalEventDetected", "KillSwitchActivated", "OperationalModeChanged",
	"ReconciliationDivergenceDetected",
}

// Consumer wires the EventBus to a Broadcaster.
type Consumer struct {
	hub Broadcaster
}

func NewConsumer(hub Broadcaster) *Consumer {
	return &Consumer{hub: hub}
}

// Subscribe registers a handler for every dashboard-relevant event name.
// Call once at startup.
func (c *Consumer) Subscribe(bus output.EventBus) {
	for _, name := range dashboardEventNames {
		eventName := name
		bus.Subscribe(eventName, func(ctx context.Context, event output.DomainEvent) error {
			c.hub.Broadcast(eventName, event)
			return nil
		})
	}
}
