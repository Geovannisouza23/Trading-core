package risk

import (
	"time"

	"trading-core/internal/domain/account"
	"trading-core/internal/domain/event"
	"trading-core/internal/domain/order"
	"trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
	"trading-core/internal/domain/signal"
)

// Thresholds carries every deterministic risk limit as domain value
// objects. It is built by the application layer from internal/config and
// never imports configuration itself, keeping the domain framework-free.
type Thresholds struct {
	PerTradePct      shared.Percentage
	MaxDailyLossPct  shared.Percentage
	MaxWeeklyLossPct shared.Percentage
	MaxDrawdown      shared.Drawdown
	MaxOpenPositions int
	MaxLeverage      shared.Leverage
	MaxSignalAge     time.Duration
	MaxMarketDataAge time.Duration
	MaxSlippage      shared.Slippage
}

// Context is the full set of facts a Specification needs to reach a
// deterministic decision. It is assembled once per evaluation by
// EvaluateRisk and passed by value to every specification.
type Context struct {
	Now               time.Time
	Account           *account.Account
	OpenPositions     []position.Position
	OpenOrders        []order.Order
	Signal            *signal.TradeSignal
	ActiveEvents      []event.MarketEvent
	MarketDataAge     time.Duration
	BrokerHealthy     bool
	ExpectedFillPrice shared.Price
	RequestedLeverage shared.Leverage
	Thresholds        Thresholds
}

// Result is the outcome of a single specification evaluation.
type Result struct {
	Satisfied  bool
	ReasonCode ReasonCode
}

func satisfied() Result { return Result{Satisfied: true} }

func violated(code ReasonCode) Result { return Result{Satisfied: false, ReasonCode: code} }

// Specification is a small, independent, deterministic risk rule.
type Specification interface {
	Name() string
	Evaluate(ctx Context) Result
}
