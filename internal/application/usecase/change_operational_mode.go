package usecase

import (
	"context"
	"fmt"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/operation"
)

// ChangeOperationalMode implements input.ChangeOperationalModeUseCase. There
// is deliberately no HTTP endpoint that lets a caller request an arbitrary
// target mode (spec section 15 only exposes pause/resume/close-only/
// kill-switch); the only caller allowed to request ModeReal is the
// application bootstrap in internal/app, which sources the confirmation
// token from validated, persisted configuration.
type ChangeOperationalMode struct {
	modes    output.OperationalModeRepository
	accounts output.AccountRepository
	tx       output.TransactionManager
	eventBus output.EventBus
	clock    output.Clock
}

func NewChangeOperationalMode(
	modes output.OperationalModeRepository,
	accounts output.AccountRepository,
	tx output.TransactionManager,
	eventBus output.EventBus,
	clock output.Clock,
) *ChangeOperationalMode {
	return &ChangeOperationalMode{modes: modes, accounts: accounts, tx: tx, eventBus: eventBus, clock: clock}
}

var _ input.ChangeOperationalModeUseCase = (*ChangeOperationalMode)(nil)

func (uc *ChangeOperationalMode) Execute(ctx context.Context, cmd command.ChangeOperationalModeCommand) error {
	target := operation.Mode(cmd.TargetMode)
	if !target.Valid() {
		return fmt.Errorf("%w: unknown target mode %q", ErrOperationalModeBlocked, cmd.TargetMode)
	}

	now := uc.clock.Now()
	state, err := uc.modes.Get(ctx)
	if err != nil {
		return fmt.Errorf("loading operational mode: %w", err)
	}
	from := state.CurrentMode

	confirmation := operation.RealModeConfirmation{
		Enabled:           cmd.RealConfirmationToken != "",
		ConfirmationToken: cmd.RealConfirmationToken,
	}
	if err := state.TransitionTo(target, cmd.ActedBy, cmd.Origin, cmd.Reason, now, confirmation); err != nil {
		return fmt.Errorf("transitioning operational mode: %w", err)
	}

	acct, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("loading account: %w", err)
	}
	acct.SetOperationalMode(target, now)

	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.modes.Save(ctx, state); err != nil {
			return err
		}
		return uc.accounts.Save(ctx, acct)
	}); err != nil {
		return fmt.Errorf("persisting operational mode change: %w", err)
	}

	return uc.eventBus.Publish(ctx, operation.OperationalModeChanged{
		From: from, To: target, ChangedBy: cmd.ActedBy, Origin: cmd.Origin, Reason: cmd.Reason, OccurredAt_: now,
	})
}
