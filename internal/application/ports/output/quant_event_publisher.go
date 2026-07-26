package output

import "context"

// QuantEventPublisher publishes to the two NATS subjects quant-engine's
// own durable consumers already expect
// (interfaces::consumer::{decision_outcome_received,model_approved} in
// the Rust repo) — the async, fire-and-forget, fan-out-to-every-instance
// counterpart to the synchronous gRPC RPCs QuantDiagnostics already
// covers (RegisterDecisionOutcome, ReloadApprovedModel). Reuses those
// same input types: this is the same operation over a different
// transport, not a different one.
type QuantEventPublisher interface {
	PublishDecisionOutcome(ctx context.Context, input RegisterDecisionOutcomeInput) error
	PublishModelApproved(ctx context.Context, input ReloadApprovedModelInput) error
}
