// Package risk models deterministic risk evaluation: specifications, a
// composite policy combining them, position sizing strategies and the
// RiskDecision aggregate that records the outcome.
package risk

// ReasonCode is a stable, machine-readable code explaining why a risk
// decision blocked (or allowed) a signal. These are the only values the
// dashboard and audit log ever need to interpret a decision.
type ReasonCode string

const (
	ReasonMaxDailyLossReached    ReasonCode = "MAX_DAILY_LOSS_REACHED"
	ReasonMaxWeeklyLossReached   ReasonCode = "MAX_WEEKLY_LOSS_REACHED"
	ReasonMaxDrawdownReached     ReasonCode = "MAX_DRAWDOWN_REACHED"
	ReasonMaxOpenPositions       ReasonCode = "MAX_OPEN_POSITIONS_REACHED"
	ReasonMaxLeverageExceeded    ReasonCode = "MAX_LEVERAGE_EXCEEDED"
	ReasonInvalidStop            ReasonCode = "INVALID_STOP"
	ReasonSignalExpired          ReasonCode = "SIGNAL_EXPIRED"
	ReasonMarketDataStale        ReasonCode = "MARKET_DATA_STALE"
	ReasonCriticalEventActive    ReasonCode = "CRITICAL_EVENT_ACTIVE"
	ReasonBrokerUnhealthy        ReasonCode = "BROKER_UNHEALTHY"
	ReasonInsufficientBalance    ReasonCode = "INSUFFICIENT_BALANCE"
	ReasonMaxSlippageExceeded    ReasonCode = "MAX_SLIPPAGE_EXCEEDED"
	ReasonOperationalModeBlocked ReasonCode = "OPERATIONAL_MODE_BLOCKS_ENTRY"
)
