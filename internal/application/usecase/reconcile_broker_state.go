package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/operation"
)

// ReconcileBrokerState implements input.ReconcileBrokerStateUseCase. It
// always runs (regardless of operational mode) and never silently corrects
// a critical divergence: it raises an incident, publishes an event and, for
// critical divergences, forces the system into CLOSE_ONLY.
type ReconcileBrokerState struct {
	accounts  output.AccountRepository
	positions output.PositionRepository
	orders    output.OrderRepository
	incidents output.SystemIncidentRepository
	modes     output.OperationalModeRepository
	broker    output.Broker
	outbox    output.OutboxRepository
	tx        output.TransactionManager
	clock     output.Clock
	tolerance decimal.Decimal
}

func NewReconcileBrokerState(
	accounts output.AccountRepository,
	positions output.PositionRepository,
	orders output.OrderRepository,
	incidents output.SystemIncidentRepository,
	modes output.OperationalModeRepository,
	broker output.Broker,
	outbox output.OutboxRepository,
	tx output.TransactionManager,
	clock output.Clock,
	tolerance decimal.Decimal,
) *ReconcileBrokerState {
	return &ReconcileBrokerState{
		accounts: accounts, positions: positions, orders: orders,
		incidents: incidents, modes: modes, broker: broker,
		outbox: outbox, tx: tx, clock: clock, tolerance: tolerance,
	}
}

var _ input.ReconcileBrokerStateUseCase = (*ReconcileBrokerState)(nil)

func (uc *ReconcileBrokerState) Execute(ctx context.Context, cmd command.ReconcileBrokerStateCommand) (input.ReconciliationReport, error) {
	now := uc.clock.Now()
	report := input.ReconciliationReport{}

	brokerAccount, err := uc.broker.GetAccount(ctx)
	if err != nil {
		uc.raiseDivergence(ctx, operation.IncidentSeverityCritical, fmt.Sprintf("broker unreachable during reconciliation: %v", err), now)
		report.DivergencesFound++
		report.CriticalDivergence = true
		report.Details = append(report.Details, "broker account unreachable")
		return report, nil
	}

	localAccount, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return report, fmt.Errorf("loading local account: %w", err)
	}

	if uc.diverges(localAccount.Balance.Decimal(), brokerAccount.Balance.Decimal()) {
		report.DivergencesFound++
		report.Details = append(report.Details, fmt.Sprintf("balance divergence: local=%s broker=%s", localAccount.Balance, brokerAccount.Balance))
	}
	if uc.diverges(localAccount.Equity.Decimal(), brokerAccount.Equity.Decimal()) {
		report.DivergencesFound++
		report.Details = append(report.Details, fmt.Sprintf("equity divergence: local=%s broker=%s", localAccount.Equity, brokerAccount.Equity))
	}

	localPositions, err := uc.positions.ListOpen(ctx)
	if err != nil {
		return report, fmt.Errorf("loading local positions: %w", err)
	}
	brokerPositions, err := uc.broker.GetPositions(ctx)
	if err != nil {
		report.DivergencesFound++
		report.CriticalDivergence = true
		report.Details = append(report.Details, "broker positions unreachable")
	} else {
		byMe := make(map[string]bool, len(brokerPositions))
		for _, bp := range brokerPositions {
			byMe[bp.Symbol.String()] = true
		}
		for _, lp := range localPositions {
			if !byMe[lp.Symbol.String()] {
				report.DivergencesFound++
				report.CriticalDivergence = true
				report.Details = append(report.Details, fmt.Sprintf("local open position %s not found at broker", lp.Symbol))
				continue
			}
			for _, bp := range brokerPositions {
				if bp.Symbol.Equal(lp.Symbol) && uc.diverges(lp.Quantity.Decimal(), bp.Quantity.Decimal()) {
					report.DivergencesFound++
					report.Details = append(report.Details, fmt.Sprintf("position quantity divergence on %s: local=%s broker=%s", lp.Symbol, lp.Quantity, bp.Quantity))
				}
			}
		}
	}

	localOrders, err := uc.orders.ListOpen(ctx)
	if err != nil {
		return report, fmt.Errorf("loading local open orders: %w", err)
	}
	brokerOrders, err := uc.broker.GetOpenOrders(ctx)
	if err != nil {
		report.DivergencesFound++
		report.CriticalDivergence = true
		report.Details = append(report.Details, "broker open orders unreachable")
	} else {
		byBrokerID := make(map[string]output.BrokerOrder, len(brokerOrders))
		for _, bo := range brokerOrders {
			byBrokerID[bo.BrokerOrderID] = bo
		}
		for _, lo := range localOrders {
			if _, ok := byBrokerID[lo.BrokerOrderID]; !ok && lo.BrokerOrderID != "" {
				report.DivergencesFound++
				report.Details = append(report.Details, fmt.Sprintf("local open order %s (broker id %s) not found at broker", lo.ClientOrderID, lo.BrokerOrderID))
			}
		}
	}

	if report.DivergencesFound == 0 {
		return report, nil
	}

	severity := operation.IncidentSeverityMedium
	if report.CriticalDivergence {
		severity = operation.IncidentSeverityCritical
	}
	uc.raiseDivergence(ctx, severity, fmt.Sprintf("%d divergence(s) found: %v", report.DivergencesFound, report.Details), now)

	if report.CriticalDivergence {
		if err := uc.forceCloseOnly(ctx, now); err != nil {
			return report, fmt.Errorf("forcing close-only after critical divergence: %w", err)
		}
	}

	return report, nil
}

func (uc *ReconcileBrokerState) diverges(local, broker decimal.Decimal) bool {
	diff := local.Sub(broker).Abs()
	if local.IsZero() {
		return !diff.IsZero()
	}
	relative := diff.Div(local.Abs())
	return relative.GreaterThan(uc.tolerance)
}

func (uc *ReconcileBrokerState) raiseDivergence(ctx context.Context, severity operation.IncidentSeverity, description string, now time.Time) {
	incident, err := operation.NewIncident(
		operation.IncidentTypeReconciliationDivergence,
		severity,
		description,
		"ReconcileBrokerState",
		"",
		now,
	)
	if err != nil {
		return
	}
	_ = uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.incidents.Save(ctx, incident); err != nil {
			return err
		}
		evt, err := newOutboxEvent("ReconciliationDivergenceDetected", operation.ReconciliationDivergenceDetected{
			IncidentID:  incident.ID.String(),
			Severity:    severity,
			Description: description,
			OccurredAt_: now,
		}, now)
		if err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, evt)
	})
}

func (uc *ReconcileBrokerState) forceCloseOnly(ctx context.Context, now time.Time) error {
	state, err := uc.modes.Get(ctx)
	if err != nil {
		return err
	}
	if state.CurrentMode == operation.ModeCloseOnly || state.CurrentMode == operation.ModeKillSwitch {
		return nil
	}
	if err := state.TransitionTo(operation.ModeCloseOnly, "system", "reconciliation", "critical divergence detected", now, operation.RealModeConfirmation{}); err != nil {
		return err
	}
	acct, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return err
	}
	acct.SetOperationalMode(operation.ModeCloseOnly, now)
	return uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.modes.Save(ctx, state); err != nil {
			return err
		}
		return uc.accounts.Save(ctx, acct)
	})
}
