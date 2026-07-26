package usecase

import (
	"context"
	"log/slog"
	"time"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/shared"
)

// natsPublishDedupTTL bounds how long a publish is remembered for dedup
// purposes — long enough to absorb a client retrying the same HTTP call
// (network blip, double-click), short enough that a genuinely repeated
// outcome/approval for the same key is never suppressed for long.
const natsPublishDedupTTL = 5 * time.Minute

// QuantEventPublisherService implements input.QuantEventPublisherService
// by delegating to output.QuantEventPublisher — same validation as the
// synchronous QuantDiagnosticsService versions of these two operations
// (they wrap the same command types), just a different output port. The
// idempotency repository is a second, Redis-backed binding of
// output.IdempotencyRepository scoped to "nats-publish" (see
// provideQuantEventPublisherUseCase) — it does not touch the
// Postgres-backed store used elsewhere, and a duplicate call is treated
// as an already-satisfied no-op rather than an error. Like every other
// Redis use in this codebase, the dedup check is optional: if Reserve
// itself fails (Redis unreachable), that degrades this one publish's
// dedup guarantee, it never fails the publish — verified empirically
// against a docker-compose stack with Redis down.
type QuantEventPublisherService struct {
	publisher  output.QuantEventPublisher
	idempotent output.IdempotencyRepository
	logger     *slog.Logger
}

func NewQuantEventPublisherService(publisher output.QuantEventPublisher, idempotent output.IdempotencyRepository, logger *slog.Logger) *QuantEventPublisherService {
	return &QuantEventPublisherService{publisher: publisher, idempotent: idempotent, logger: logger}
}

var _ input.QuantEventPublisherService = (*QuantEventPublisherService)(nil)

// reserve reports whether the caller should proceed with the publish:
// true both when it genuinely won the reservation and when Redis itself
// is unavailable (fail open — a missed dedup window is preferable to a
// dropped publish for an optional guard).
func (s *QuantEventPublisherService) reserve(ctx context.Context, key string) bool {
	reserved, err := s.idempotent.Reserve(ctx, "nats-publish", key, natsPublishDedupTTL)
	if err != nil {
		s.logger.Warn("nats-publish idempotency check failed; publishing without the dedup guard", "key", key, "error", err)
		return true
	}
	return reserved
}

func (s *QuantEventPublisherService) PublishDecisionOutcome(ctx context.Context, cmd command.RegisterDecisionOutcomeCommand) error {
	if cmd.DecisionID == "" {
		return shared.NewValidationError("decision_id", "must not be empty")
	}
	if !s.reserve(ctx, "decision-outcome-"+cmd.DecisionID) {
		return nil
	}
	return s.publisher.PublishDecisionOutcome(ctx, output.RegisterDecisionOutcomeInput{
		DecisionID:      cmd.DecisionID,
		OrderCreated:    cmd.OrderCreated,
		Executed:        cmd.Executed,
		RejectionReason: cmd.RejectionReason,
		EntryPrice:      cmd.EntryPrice,
		ExitPrice:       cmd.ExitPrice,
		Quantity:        cmd.Quantity,
		Fees:            cmd.Fees,
		Slippage:        cmd.Slippage,
		Pnl:             cmd.Pnl,
		ExitReason:      cmd.ExitReason,
		EntryAt:         cmd.EntryAt,
		ExitAt:          cmd.ExitAt,
	})
}

func (s *QuantEventPublisherService) PublishModelApproved(ctx context.Context, cmd command.ReloadApprovedModelCommand) error {
	if cmd.ModelName == "" {
		return shared.NewValidationError("model_name", "must not be empty")
	}
	if cmd.Stage == "" {
		return shared.NewValidationError("stage", "must not be empty")
	}
	if !s.reserve(ctx, "model-approved-"+cmd.ModelName+"-"+cmd.Stage) {
		return nil
	}
	return s.publisher.PublishModelApproved(ctx, output.ReloadApprovedModelInput{
		ModelName: cmd.ModelName,
		Stage:     cmd.Stage,
	})
}
