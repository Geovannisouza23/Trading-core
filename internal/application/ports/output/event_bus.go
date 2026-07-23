package output

import (
	"context"
	"time"
)

// DomainEvent is satisfied implicitly (structural typing) by every domain
// event struct across internal/domain/*/events.go. No domain package needs
// to import this one.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// EventHandler processes a single published event.
type EventHandler func(ctx context.Context, event DomainEvent) error

// EventBus decouples publishers (use cases, the outbox worker) from
// subscribers (the WebSocket hub, consumers, notifiers). The MVP
// implementation is in-memory; this interface is what lets it be swapped
// for Pub/Sub, NATS or Kafka later without touching a single use case.
type EventBus interface {
	Publish(ctx context.Context, event DomainEvent) error
	Subscribe(eventName string, handler EventHandler)
}
