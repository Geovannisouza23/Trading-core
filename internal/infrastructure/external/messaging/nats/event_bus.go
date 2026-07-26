package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"trading-core/internal/application/ports/output"
)

// EventBus is the real, JetStream-backed output.EventBus. Local dispatch
// (registered handlers, e.g. the order/outbox consumers) stays exactly
// synchronous and in-process — identical to messaging/memory.EventBus —
// so existing behavior never regresses just because NATS is now wired in.
// The NATS publish is an additional, best-effort side effect on top of
// that, mirroring quant-engine's own
// `let _ = self.event_bus.publish(event).await;`: a broker hiccup never
// fails the local dispatch or the use case that triggered it.
type EventBus struct {
	conn   *natsgo.Conn
	js     jetstream.JetStream
	prefix string
	logger *slog.Logger

	mu       sync.RWMutex
	handlers map[string][]output.EventHandler
}

func NewEventBus(conn *natsgo.Conn, js jetstream.JetStream, streamPrefix string, logger *slog.Logger) *EventBus {
	return &EventBus{
		conn:     conn,
		js:       js,
		prefix:   streamPrefix,
		logger:   logger,
		handlers: make(map[string][]output.EventHandler),
	}
}

var _ output.EventBus = (*EventBus)(nil)

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

	if err := b.publishToNats(ctx, event); err != nil {
		b.logger.WarnContext(ctx, "publishing domain event to nats failed; local dispatch already ran",
			"event", event.EventName(), "error", err)
	}

	return errors.Join(errs...)
}

// publishToNats serializes event into the same Envelope shape
// quant-engine uses, on subject
// "{prefix}.trading-core.{event_name}.events". DomainEvent carries no
// aggregate ID (only EventName/OccurredAt), so "trading-core" stands in
// as a fixed aggregate_type for this repo's own internal events — this
// is deliberately coarser than quant-engine's own per-aggregate subjects,
// since nothing downstream needs finer routing yet.
func (b *EventBus) publishToNats(ctx context.Context, event output.DomainEvent) error {
	envelope := newEnvelope(
		event.EventName(),
		"trading-core",
		"trading-core",
		fmt.Sprintf("%s-%d", event.EventName(), event.OccurredAt().UnixNano()),
		event.OccurredAt(),
		event,
	)
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshaling envelope: %w", err)
	}

	subject := fmt.Sprintf("%s.trading-core.%s.events", b.prefix, event.EventName())
	_, err = b.js.Publish(ctx, subject, payload, jetstream.WithMsgID(envelope.IdempotencyKey))
	if err != nil {
		return fmt.Errorf("publishing to %s: %w", subject, err)
	}
	return nil
}

func (b *EventBus) Subscribe(eventName string, handler output.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

// Close drains the underlying NATS connection. Registered as an
// fx.Lifecycle.OnStop hook by internal/app/providers.go.
func (b *EventBus) Close() error {
	return b.conn.Drain()
}
