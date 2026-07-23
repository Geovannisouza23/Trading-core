package risk

import "trading-core/internal/domain/shared"

// MaxDailyLossSpecification blocks new entries once realized+unrealized
// daily P&L breaches the configured percentage of account balance.
type MaxDailyLossSpecification struct{}

func (MaxDailyLossSpecification) Name() string { return "MaxDailyLossSpecification" }

func (MaxDailyLossSpecification) Evaluate(ctx Context) Result {
	if !ctx.Account.DailyPnL.IsNegative() {
		return satisfied()
	}
	limit := ctx.Thresholds.MaxDailyLossPct.Of(ctx.Account.Balance)
	loss := ctx.Account.DailyPnL.Neg()
	if loss.GreaterThanOrEqual(limit) {
		return violated(ReasonMaxDailyLossReached)
	}
	return satisfied()
}

// MaxWeeklyLossSpecification blocks new entries once weekly P&L breaches the
// configured percentage of account balance.
type MaxWeeklyLossSpecification struct{}

func (MaxWeeklyLossSpecification) Name() string { return "MaxWeeklyLossSpecification" }

func (MaxWeeklyLossSpecification) Evaluate(ctx Context) Result {
	if !ctx.Account.WeeklyPnL.IsNegative() {
		return satisfied()
	}
	limit := ctx.Thresholds.MaxWeeklyLossPct.Of(ctx.Account.Balance)
	loss := ctx.Account.WeeklyPnL.Neg()
	if loss.GreaterThanOrEqual(limit) {
		return violated(ReasonMaxWeeklyLossReached)
	}
	return satisfied()
}

// MaxDrawdownSpecification blocks new entries once current drawdown from
// peak equity breaches the configured limit.
type MaxDrawdownSpecification struct{}

func (MaxDrawdownSpecification) Name() string { return "MaxDrawdownSpecification" }

func (MaxDrawdownSpecification) Evaluate(ctx Context) Result {
	if ctx.Account.CurrentDrawdown.GreaterThanOrEqual(ctx.Thresholds.MaxDrawdown) {
		return violated(ReasonMaxDrawdownReached)
	}
	return satisfied()
}

// MaxOpenPositionsSpecification caps concurrent open positions.
type MaxOpenPositionsSpecification struct{}

func (MaxOpenPositionsSpecification) Name() string { return "MaxOpenPositionsSpecification" }

func (MaxOpenPositionsSpecification) Evaluate(ctx Context) Result {
	if len(ctx.OpenPositions) >= ctx.Thresholds.MaxOpenPositions {
		return violated(ReasonMaxOpenPositions)
	}
	return satisfied()
}

// MaxLeverageSpecification caps the leverage requested for the new position.
type MaxLeverageSpecification struct{}

func (MaxLeverageSpecification) Name() string { return "MaxLeverageSpecification" }

func (MaxLeverageSpecification) Evaluate(ctx Context) Result {
	if ctx.RequestedLeverage.GreaterThan(ctx.Thresholds.MaxLeverage) {
		return violated(ReasonMaxLeverageExceeded)
	}
	return satisfied()
}

// ValidStopSpecification requires the signal to carry a coherent, non-zero
// distance stop loss. TradeSignal's constructor already enforces side
// coherence; this specification guards against a nil signal reaching risk
// evaluation and against a degenerate (zero-distance) stop.
type ValidStopSpecification struct{}

func (ValidStopSpecification) Name() string { return "ValidStopSpecification" }

func (ValidStopSpecification) Evaluate(ctx Context) Result {
	if ctx.Signal == nil {
		return violated(ReasonInvalidStop)
	}
	if ctx.Signal.StopPrice.Equal(ctx.Signal.EntryPrice) {
		return violated(ReasonInvalidStop)
	}
	return satisfied()
}

// SignalNotExpiredSpecification requires the signal to still be within its
// validity window and within the maximum allowed age.
type SignalNotExpiredSpecification struct{}

func (SignalNotExpiredSpecification) Name() string { return "SignalNotExpiredSpecification" }

func (SignalNotExpiredSpecification) Evaluate(ctx Context) Result {
	if ctx.Signal == nil {
		return violated(ReasonSignalExpired)
	}
	if ctx.Signal.IsExpired(ctx.Now) {
		return violated(ReasonSignalExpired)
	}
	if ctx.Now.Sub(ctx.Signal.CreatedAt) > ctx.Thresholds.MaxSignalAge {
		return violated(ReasonSignalExpired)
	}
	return satisfied()
}

// MarketDataFreshSpecification requires the market data backing the signal
// to be recent enough to be trustworthy.
type MarketDataFreshSpecification struct{}

func (MarketDataFreshSpecification) Name() string { return "MarketDataFreshSpecification" }

func (MarketDataFreshSpecification) Evaluate(ctx Context) Result {
	if ctx.MarketDataAge > ctx.Thresholds.MaxMarketDataAge {
		return violated(ReasonMarketDataStale)
	}
	return satisfied()
}

// NoCriticalEventSpecification blocks new entries while a critical market
// event affecting the signal's symbol is active.
type NoCriticalEventSpecification struct{}

func (NoCriticalEventSpecification) Name() string { return "NoCriticalEventSpecification" }

func (NoCriticalEventSpecification) Evaluate(ctx Context) Result {
	if ctx.Signal == nil {
		return satisfied()
	}
	for i := range ctx.ActiveEvents {
		ev := &ctx.ActiveEvents[i]
		if ev.IsCritical() && ev.AffectsSymbol(ctx.Signal.Symbol) && !ev.IsExpired(ctx.Now) {
			return violated(ReasonCriticalEventActive)
		}
	}
	return satisfied()
}

// BrokerHealthySpecification requires the broker adapter's circuit breaker
// to currently be closed (healthy).
type BrokerHealthySpecification struct{}

func (BrokerHealthySpecification) Name() string { return "BrokerHealthySpecification" }

func (BrokerHealthySpecification) Evaluate(ctx Context) Result {
	if !ctx.BrokerHealthy {
		return violated(ReasonBrokerUnhealthy)
	}
	return satisfied()
}

// SufficientBalanceSpecification requires the account to have a positive
// available balance before position sizing is attempted.
type SufficientBalanceSpecification struct{}

func (SufficientBalanceSpecification) Name() string { return "SufficientBalanceSpecification" }

func (SufficientBalanceSpecification) Evaluate(ctx Context) Result {
	if !ctx.Account.AvailableBalance.IsPositive() {
		return violated(ReasonInsufficientBalance)
	}
	return satisfied()
}

// MaximumSlippageSpecification blocks entries when the expected fill price
// has already drifted too far from the signal's entry price.
type MaximumSlippageSpecification struct{}

func (MaximumSlippageSpecification) Name() string { return "MaximumSlippageSpecification" }

func (MaximumSlippageSpecification) Evaluate(ctx Context) Result {
	if ctx.Signal == nil {
		return violated(ReasonMaxSlippageExceeded)
	}
	observed := shared.SlippageBetween(ctx.Signal.EntryPrice, ctx.ExpectedFillPrice)
	if observed.GreaterThan(ctx.Thresholds.MaxSlippage) {
		return violated(ReasonMaxSlippageExceeded)
	}
	return satisfied()
}
