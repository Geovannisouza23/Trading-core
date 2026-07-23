package usecase

import (
	"context"
	"fmt"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/operation"
)

// ActivateKillSwitch implements input.ActivateKillSwitchUseCase.
type ActivateKillSwitch struct {
	modes     output.OperationalModeRepository
	accounts  output.AccountRepository
	incidents output.SystemIncidentRepository
	outbox    output.OutboxRepository
	tx        output.TransactionManager
	notifier  output.Notifier
	clock     output.Clock
}

func NewActivateKillSwitch(
	modes output.OperationalModeRepository,
	accounts output.AccountRepository,
	incidents output.SystemIncidentRepository,
	outbox output.OutboxRepository,
	tx output.TransactionManager,
	notifier output.Notifier,
	clock output.Clock,
) *ActivateKillSwitch {
	return &ActivateKillSwitch{
		modes:     modes,
		accounts:  accounts,
		incidents: incidents,
		outbox:    outbox,
		tx:        tx,
		notifier:  notifier,
		clock:     clock,
	}
}

var _ input.ActivateKillSwitchUseCase = (*ActivateKillSwitch)(nil)

func (uc *ActivateKillSwitch) Execute(ctx context.Context, cmd command.ActivateKillSwitchCommand) error {
	now := uc.clock.Now()

	state, err := uc.modes.Get(ctx)
	if err != nil {
		return fmt.Errorf("loading operational mode: %w", err)
	}
	if err := state.TransitionTo(operation.ModeKillSwitch, cmd.ActivatedBy, cmd.Origin, cmd.Reason, now, operation.RealModeConfirmation{}); err != nil {
		return fmt.Errorf("transitioning to kill switch: %w", err)
	}

	acct, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("loading account: %w", err)
	}
	acct.SetOperationalMode(operation.ModeKillSwitch, now)

	incident, err := operation.NewIncident(
		operation.IncidentTypeKillSwitchActivated,
		operation.IncidentSeverityCritical,
		cmd.Reason,
		cmd.Origin,
		"",
		now,
	)
	if err != nil {
		return fmt.Errorf("building incident: %w", err)
	}

	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.modes.Save(ctx, state); err != nil {
			return err
		}
		if err := uc.accounts.Save(ctx, acct); err != nil {
			return err
		}
		if err := uc.incidents.Save(ctx, incident); err != nil {
			return err
		}
		evt, err := newOutboxEvent("KillSwitchActivated", operation.KillSwitchActivated{
			ActivatedBy: cmd.ActivatedBy,
			Origin:      cmd.Origin,
			Reason:      cmd.Reason,
			AllowClose:  cmd.AllowClose,
			OccurredAt_: now,
		}, now)
		if err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, evt)
	}); err != nil {
		return fmt.Errorf("persisting kill switch activation: %w", err)
	}

	// Notification delivery is best-effort: the kill switch must take
	// effect even if the alerting channel is unreachable.
	_ = uc.notifier.Notify(ctx, output.Notification{
		Title:    "KILL SWITCH ACTIVATED",
		Message:  fmt.Sprintf("Kill switch activated by %s (%s): %s", cmd.ActivatedBy, cmd.Origin, cmd.Reason),
		Severity: output.NotificationCritical,
	})

	return nil
}
