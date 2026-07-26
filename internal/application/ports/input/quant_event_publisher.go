package input

import (
	"context"

	"trading-core/internal/application/command"
)

// QuantEventPublisherService is the async, NATS-backed counterpart to
// QuantDiagnosticsService's RegisterDecisionOutcome/ReloadApprovedModel —
// same operations, fire-and-forget over the message bus instead of a
// synchronous gRPC call.
type QuantEventPublisherService interface {
	PublishDecisionOutcome(ctx context.Context, cmd command.RegisterDecisionOutcomeCommand) error
	PublishModelApproved(ctx context.Context, cmd command.ReloadApprovedModelCommand) error
}
