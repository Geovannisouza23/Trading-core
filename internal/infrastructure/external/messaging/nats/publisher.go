package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"trading-core/internal/application/ports/output"
)

// These two subjects are fixed strings quant-engine's own consumers
// already hard-code (interfaces::consumer::decision_outcome_received::SUBJECT,
// interfaces::consumer::model_approved::SUBJECT) — unlike this repo's own
// domain events, they do NOT follow the
// "{prefix}.{aggregate_type}.{aggregate_id}.events" pattern, so they are
// not derived from Envelope/newEnvelope's subject conventions.
const (
	decisionOutcomeSubject = "quant.decision.outcome.received"
	modelApprovedSubject   = "quant.model.approved"
)

// activityChangedSubject has no consumer hard-coded on the Rust side —
// it exists for the Python training-pipeline's idle_watcher, a fixed
// string by the same reasoning as the two subjects above (a stable,
// audience-scoped contract, not derived from Envelope's
// aggregate-based subject convention). Kept under the "quant." prefix
// like its two siblings above even though it's about trading-core's own
// activity, not quant-engine's — Connect's provisioned JetStream stream
// filters strictly on "{streamPrefix}.>" (streamPrefix defaults to
// "quant", shared with quant-engine, see client.go), so any subject
// outside that prefix has no stream to land in and publish fails with
// "no response from stream".
const activityChangedSubject = "quant.activity.changed"

// decisionOutcomePayload mirrors
// interfaces::consumer::decision_outcome_received::DecisionOutcomeReceivedPayload
// field for field.
type decisionOutcomePayload struct {
	DecisionID      string     `json:"decision_id"`
	OrderCreated    bool       `json:"order_created"`
	Executed        bool       `json:"executed"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	EntryPrice      *string    `json:"entry_price,omitempty"`
	ExitPrice       *string    `json:"exit_price,omitempty"`
	Quantity        *string    `json:"quantity,omitempty"`
	Fees            *string    `json:"fees,omitempty"`
	Slippage        *string    `json:"slippage,omitempty"`
	Pnl             *string    `json:"pnl,omitempty"`
	ExitReason      *string    `json:"exit_reason,omitempty"`
	EntryAt         *time.Time `json:"entry_at,omitempty"`
	ExitAt          *time.Time `json:"exit_at,omitempty"`
}

// modelApprovedPayload mirrors
// interfaces::consumer::model_approved::ModelApprovedPayload field for
// field.
type modelApprovedPayload struct {
	ModelName string `json:"model_name"`
	Stage     string `json:"stage"`
}

// activityChangedPayload — Idle is the whole point: zero open positions
// and zero in-flight orders. ChangedAt is when the transition was
// observed, not when it was published.
type activityChangedPayload struct {
	Idle      bool      `json:"idle"`
	ChangedAt time.Time `json:"changed_at"`
}

// Publisher implements output.QuantEventPublisher.
type Publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{js: js}
}

var _ output.QuantEventPublisher = (*Publisher)(nil)
var _ output.ActivityPublisher = (*Publisher)(nil)

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nonZeroTimePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (p *Publisher) PublishDecisionOutcome(ctx context.Context, input output.RegisterDecisionOutcomeInput) error {
	payload := decisionOutcomePayload{
		DecisionID:      input.DecisionID,
		OrderCreated:    input.OrderCreated,
		Executed:        input.Executed,
		RejectionReason: nonEmptyPtr(input.RejectionReason),
		EntryPrice:      nonEmptyPtr(input.EntryPrice),
		ExitPrice:       nonEmptyPtr(input.ExitPrice),
		Quantity:        nonEmptyPtr(input.Quantity),
		Fees:            nonEmptyPtr(input.Fees),
		Slippage:        nonEmptyPtr(input.Slippage),
		Pnl:             nonEmptyPtr(input.Pnl),
		ExitReason:      nonEmptyPtr(input.ExitReason),
		EntryAt:         nonZeroTimePtr(input.EntryAt),
		ExitAt:          nonZeroTimePtr(input.ExitAt),
	}
	envelope := newEnvelope("DecisionOutcomeReceived", input.DecisionID, "decision", "decision-outcome-"+input.DecisionID, time.Now().UTC(), payload)
	return p.publish(ctx, decisionOutcomeSubject, envelope)
}

func (p *Publisher) PublishModelApproved(ctx context.Context, input output.ReloadApprovedModelInput) error {
	payload := modelApprovedPayload{
		ModelName: input.ModelName,
		Stage:     input.Stage,
	}
	envelope := newEnvelope("ModelApproved", input.ModelName, "model", "model-approved-"+input.ModelName+"-"+input.Stage, time.Now().UTC(), payload)
	return p.publish(ctx, modelApprovedSubject, envelope)
}

func (p *Publisher) PublishActivityChanged(ctx context.Context, idle bool, changedAt time.Time) error {
	payload := activityChangedPayload{
		Idle:      idle,
		ChangedAt: changedAt,
	}
	// Idempotency key includes the nanosecond timestamp: unlike the two
	// publishers above (one message per business event), this one fires
	// on every idle<->active edge, so each transition must get its own
	// JetStream dedup id — reusing a fixed key would make the second
	// transition in either direction silently dedup away.
	idempotencyKey := fmt.Sprintf("activity-changed-%t-%d", idle, changedAt.UnixNano())
	envelope := newEnvelope("ActivityChanged", "trading-core", "system", idempotencyKey, time.Now().UTC(), payload)
	return p.publish(ctx, activityChangedSubject, envelope)
}

func (p *Publisher) publish(ctx context.Context, subject string, envelope Envelope) error {
	if p.js == nil {
		return fmt.Errorf("nats is not connected; cannot publish to %s (set NATS_URL and ensure the broker is reachable)", subject)
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshaling envelope for %s: %w", subject, err)
	}
	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(envelope.IdempotencyKey)); err != nil {
		return fmt.Errorf("publishing to %s: %w", subject, err)
	}
	return nil
}
