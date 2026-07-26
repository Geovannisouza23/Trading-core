package nats

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewEnvelopeMatchesQuantEngineEventEnvelopeShape locks the exact
// JSON key set and casing this package puts on the wire against
// contracts::events::EventEnvelope<T> (quant-engine's
// src/contracts/events/envelope.rs) — a drift here (a renamed field, a
// dropped key) would silently break quant-engine's deserialization
// without failing anything on this side, since Publish never gets a
// response body to validate against.
func TestNewEnvelopeMatchesQuantEngineEventEnvelopeShape(t *testing.T) {
	occurredAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	envelope := newEnvelope("DecisionOutcomeReceived", "decision-123", "decision", "decision-outcome-decision-123", occurredAt, map[string]string{"decision_id": "decision-123"})

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshaling envelope: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshaling envelope back to a map: %v", err)
	}

	wantKeys := []string{
		"event_id", "event_type", "event_version",
		"aggregate_id", "aggregate_type", "aggregate_version",
		"idempotency_key", "occurred_at", "producer", "payload",
	}
	for _, key := range wantKeys {
		if _, ok := decoded[key]; !ok {
			t.Errorf("expected JSON key %q, got keys %v", key, keysOf(decoded))
		}
	}

	// correlation_id/causation_id are nil in every call site today and
	// must be omitted (omitempty), not serialized as JSON null —
	// quant-engine's Option<Uuid> deserializes either way, but an
	// explicit `null` would be a silent behavior change worth catching.
	for _, key := range []string{"correlation_id", "causation_id"} {
		if _, ok := decoded[key]; ok {
			t.Errorf("expected %q to be omitted when nil, got %v", key, decoded[key])
		}
	}

	if decoded["producer"] != "trading-core" {
		t.Errorf("producer = %v, want %q", decoded["producer"], "trading-core")
	}
	if decoded["event_type"] != "DecisionOutcomeReceived" {
		t.Errorf("event_type = %v, want %q", decoded["event_type"], "DecisionOutcomeReceived")
	}
	if decoded["aggregate_type"] != "decision" {
		t.Errorf("aggregate_type = %v, want %q", decoded["aggregate_type"], "decision")
	}
	if decoded["idempotency_key"] != "decision-outcome-decision-123" {
		t.Errorf("idempotency_key = %v, want %q", decoded["idempotency_key"], "decision-outcome-decision-123")
	}
}

func TestNewEnvelopeRoundTripsThroughJSON(t *testing.T) {
	occurredAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	original := newEnvelope("ModelApproved", "primary", "model", "model-approved-primary-CANARY", occurredAt, modelApprovedPayload{ModelName: "primary", Stage: "CANARY"})

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshaling envelope: %v", err)
	}

	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshaling envelope: %v", err)
	}

	if decoded.EventID != original.EventID {
		t.Errorf("event_id round-trip mismatch: got %v, want %v", decoded.EventID, original.EventID)
	}
	if !decoded.OccurredAt.Equal(original.OccurredAt) {
		t.Errorf("occurred_at round-trip mismatch: got %v, want %v", decoded.OccurredAt, original.OccurredAt)
	}
	if decoded.EventVersion != 1 || decoded.AggregateVersion != 1 {
		t.Errorf("expected default versions of 1, got event_version=%d aggregate_version=%d", decoded.EventVersion, decoded.AggregateVersion)
	}
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
