package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/command"
	"trading-core/internal/application/mapper"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/event"
	"trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"
)

// SymbolTradingRules carries the broker-imposed sizing constraints for a
// symbol. The Broker port (section 13.1 of the spec) does not expose an
// exchange-info method, so the MVP uses a single conservative default set
// injected at construction instead of a per-symbol lookup.
type SymbolTradingRules struct {
	MinQuantity shared.Quantity
	StepSize    decimal.Decimal
	MinNotional shared.Money
}

// EvaluateRisk implements input.EvaluateRiskUseCase.
type EvaluateRisk struct {
	accounts   output.AccountRepository
	positions  output.PositionRepository
	orders     output.OrderRepository
	events     output.MarketEventRepository
	signals    output.TradeSignalRepository
	decisions  output.RiskDecisionRepository
	broker     output.Broker
	eventBus   output.EventBus
	clock      output.Clock
	policy     *risk.CompositeRiskPolicy
	sizer      risk.PositionSizer
	thresholds risk.Thresholds
	rules      SymbolTradingRules
}

func NewEvaluateRisk(
	accounts output.AccountRepository,
	positions output.PositionRepository,
	orders output.OrderRepository,
	events output.MarketEventRepository,
	signals output.TradeSignalRepository,
	decisions output.RiskDecisionRepository,
	broker output.Broker,
	eventBus output.EventBus,
	clock output.Clock,
	policy *risk.CompositeRiskPolicy,
	sizer risk.PositionSizer,
	thresholds risk.Thresholds,
	rules SymbolTradingRules,
) *EvaluateRisk {
	return &EvaluateRisk{
		accounts:   accounts,
		positions:  positions,
		orders:     orders,
		events:     events,
		signals:    signals,
		decisions:  decisions,
		broker:     broker,
		eventBus:   eventBus,
		clock:      clock,
		policy:     policy,
		sizer:      sizer,
		thresholds: thresholds,
		rules:      rules,
	}
}

var _ input.EvaluateRiskUseCase = (*EvaluateRisk)(nil)

func (uc *EvaluateRisk) Execute(ctx context.Context, cmd command.EvaluateRiskCommand) (input.EvaluateRiskResult, error) {
	now := uc.clock.Now()

	tradeSignal, err := uc.signals.GetByID(ctx, cmd.SignalID)
	if err != nil {
		return input.EvaluateRiskResult{}, fmt.Errorf("loading signal: %w", err)
	}
	acct, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return input.EvaluateRiskResult{}, fmt.Errorf("loading account: %w", err)
	}
	openPositions, err := uc.positions.ListOpen(ctx)
	if err != nil {
		return input.EvaluateRiskResult{}, fmt.Errorf("loading open positions: %w", err)
	}
	openOrders, err := uc.orders.ListOpen(ctx)
	if err != nil {
		return input.EvaluateRiskResult{}, fmt.Errorf("loading open orders: %w", err)
	}
	activeEvents, err := uc.events.ListActive(ctx)
	if err != nil {
		return input.EvaluateRiskResult{}, fmt.Errorf("loading active market events: %w", err)
	}

	brokerHealthy := true
	if _, err := uc.broker.GetAccount(ctx); err != nil {
		brokerHealthy = false
	}

	eventRestrictions := restrictionsFor(activeEvents, tradeSignal.Symbol, now)

	riskCtx := risk.Context{
		Now:               now,
		Account:           acct,
		OpenPositions:     openPositions,
		OpenOrders:        openOrders,
		Signal:            tradeSignal,
		ActiveEvents:      activeEvents,
		MarketDataAge:     now.Sub(tradeSignal.CreatedAt),
		BrokerHealthy:     brokerHealthy,
		ExpectedFillPrice: cmd.CurrentPrice,
		RequestedLeverage: uc.thresholds.MaxLeverage,
		Thresholds:        uc.thresholds,
	}

	allowedByMode := acct.OperationalMode.AllowsNewEntries()
	var evaluation risk.Evaluation
	if !allowedByMode {
		evaluation = risk.Evaluation{
			Allowed:      false,
			ReasonCodes:  []risk.ReasonCode{risk.ReasonOperationalModeBlocked},
			AppliedRules: []string{"OperationalModeAllowsEntries"},
		}
	} else {
		evaluation = uc.policy.Evaluate(riskCtx)
	}

	sizingResult := risk.SizingResult{Rejected: true, Reason: "risk specifications blocked the signal"}
	if evaluation.Allowed {
		sizingResult = uc.sizer.Size(risk.SizingInput{
			AvailableBalance: acct.AvailableBalance,
			RiskPerTrade:     uc.thresholds.PerTradePct,
			EntryPrice:       tradeSignal.EntryPrice,
			StopPrice:        tradeSignal.StopPrice,
			MinQuantity:      uc.rules.MinQuantity,
			StepSize:         uc.rules.StepSize,
			MinNotional:      uc.rules.MinNotional,
			MaxLeverage:      uc.thresholds.MaxLeverage,
			ExistingExposure: shared.ZeroMoney(),
		})
	}

	originalSize := shared.ZeroQuantity()
	if !sizingResult.Rejected {
		originalSize = sizingResult.Quantity
	}

	decision := risk.NewDecision(tradeSignal.ID, evaluation, originalSize, sizingResult, eventRestrictions, now)

	if err := uc.decisions.Save(ctx, decision); err != nil {
		return input.EvaluateRiskResult{}, fmt.Errorf("persisting risk decision: %w", err)
	}
	if err := uc.eventBus.Publish(ctx, risk.RiskDecisionCreated{Decision: decision, OccurredAt_: now}); err != nil {
		return input.EvaluateRiskResult{}, fmt.Errorf("publishing RiskDecisionCreated: %w", err)
	}

	return input.EvaluateRiskResult{Decision: mapper.ToRiskDecisionDTO(decision)}, nil
}

func restrictionsFor(activeEvents []event.MarketEvent, symbol shared.Symbol, now time.Time) []string {
	var restrictions []string
	for i := range activeEvents {
		ev := &activeEvents[i]
		if ev.IsExpired(now) || !ev.AffectsSymbol(symbol) {
			continue
		}
		restrictions = append(restrictions, fmt.Sprintf("%s:%s", ev.EventType, ev.Action))
	}
	return restrictions
}
