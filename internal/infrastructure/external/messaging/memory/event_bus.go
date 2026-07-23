// Package memory implements output.EventBus in-process, with no external
// broker. This is the MVP implementation; any of the other messaging/*
// packages (or a future Pub/Sub, NATS, Kafka adapter) can replace it
// without any use case changing, since they all depend only on
// output.EventBus.
package memory

import (
	"context"
	"errors"
	"sync"

	"trading-core/internal/application/ports/output"
)

type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]output.EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]output.EventHandler)}
}

var _ output.EventBus = (*EventBus)(nil)

// Publish invokes every handler subscribed to event.EventName()
// synchronously, in registration order, and joins any handler errors.
func (b *EventBus) Publish(ctx context.Context, event output.DomainEvent) error {
	b.mu.RLock()
	handlers := append([]output.EventHandler(nil), b.handlers[event.EventName()]...)
	b.mu.RUnlock()

	var errs []error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (b *EventBus) Subscribe(eventName string, handler output.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}
