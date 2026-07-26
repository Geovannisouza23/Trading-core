// Package nats is the real, JetStream-backed output.EventBus
// implementation, plus a dedicated publisher for quant-engine's two
// fixed-subject NATS consumers (quant.decision.outcome.received,
// quant.model.approved — see publisher.go). Optional by default
// (NatsConfig.Required=false): an unreachable broker falls back to
// messaging/memory, mirroring quant-engine's own
// app::providers::build_event_bus exactly.
package nats

import (
	"time"

	"github.com/google/uuid"
)

// Envelope mirrors quant-engine's contracts::events::EventEnvelope<T>
// field for field (src/contracts/events/envelope.rs) — every message this
// package puts on NATS uses this exact JSON shape, whether it's one of
// trading-core's own domain events or a quant-engine-specific payload
// (DecisionOutcomePayload, ModelApprovedPayload).
type Envelope struct {
	EventID          uuid.UUID  `json:"event_id"`
	EventType        string     `json:"event_type"`
	EventVersion     uint32     `json:"event_version"`
	AggregateID      string     `json:"aggregate_id"`
	AggregateType    string     `json:"aggregate_type"`
	AggregateVersion uint64     `json:"aggregate_version"`
	CorrelationID    *uuid.UUID `json:"correlation_id,omitempty"`
	CausationID      *uuid.UUID `json:"causation_id,omitempty"`
	IdempotencyKey   string     `json:"idempotency_key"`
	OccurredAt       time.Time  `json:"occurred_at"`
	Producer         string     `json:"producer"`
	Payload          any        `json:"payload"`
}

const producer = "trading-core"

// newEnvelope builds an Envelope with sane defaults (new event_id,
// producer="trading-core") for the given aggregate/payload. version
// defaults to 1 for every publish call site in this package today — none
// of them track a running aggregate_version yet, unlike quant-engine's
// own publishers, which increment it per use case.
func newEnvelope(eventType, aggregateID, aggregateType, idempotencyKey string, occurredAt time.Time, payload any) Envelope {
	return Envelope{
		EventID:          uuid.New(),
		EventType:        eventType,
		EventVersion:     1,
		AggregateID:      aggregateID,
		AggregateType:    aggregateType,
		AggregateVersion: 1,
		IdempotencyKey:   idempotencyKey,
		OccurredAt:       occurredAt,
		Producer:         producer,
		Payload:          payload,
	}
}
