package output

import (
	"context"
	"time"

	"trading-core/internal/domain/market"
)

// BacktestConfig mirrors quant-engine's BacktestConfig wire message.
// Decimal fields stay strings — this data only ever gets displayed to an
// operator or round-tripped back to quant-engine, never fed into
// risk/execution, so there is no domain invariant here to protect with a
// value object. json tags double this type as the HTTP request/response
// shape too (see docs/decisions/0010-real-grpc-client-for-quant-engine.md
// for why this port intentionally departs from the rest of ports/output,
// none of which carry json tags: unlike a repository or broker port, nothing
// here is a domain entity with invariants to protect from a wire format).
type BacktestConfig struct {
	Symbol                  string            `json:"symbol"`
	Timeframe               string            `json:"timeframe"`
	StrategyName            string            `json:"strategy_name"`
	StrategyVersion         string            `json:"strategy_version"`
	StrategyParams          map[string]string `json:"strategy_params,omitempty"`
	InitialCapital          string            `json:"initial_capital"`
	FeeRate                 string            `json:"fee_rate"`
	Slippage                string            `json:"slippage"`
	ExecutionModel          string            `json:"execution_model"` // "CLOSE_PRICE" | "NEXT_OPEN" | "OHLC" | "CONSERVATIVE"
	StartAt                 time.Time         `json:"start_at"`
	EndAt                   time.Time         `json:"end_at"`
	AllowShort              bool              `json:"allow_short"`
	ModelName               string            `json:"model_name,omitempty"`
	ModelVersion            string            `json:"model_version,omitempty"`
	GenerateTrainingRecords bool              `json:"generate_training_records"`
}

// BacktestMetrics mirrors quant-engine's BacktestMetrics wire message
// field for field. Some fields are documented by the contract as "empty
// string means undefined" (ProfitFactor, SharpeRatio) rather than zero.
type BacktestMetrics struct {
	TotalReturn              string            `json:"total_return"`
	NetProfit                string            `json:"net_profit"`
	GrossProfit              string            `json:"gross_profit"`
	GrossLoss                string            `json:"gross_loss"`
	WinRate                  string            `json:"win_rate"`
	LossRate                 string            `json:"loss_rate"`
	TotalTrades              uint32            `json:"total_trades"`
	WinningTrades            uint32            `json:"winning_trades"`
	LosingTrades             uint32            `json:"losing_trades"`
	AverageTrade             string            `json:"average_trade"`
	AverageWinner            string            `json:"average_winner"`
	AverageLoser             string            `json:"average_loser"`
	LargestWinner            string            `json:"largest_winner"`
	LargestLoser             string            `json:"largest_loser"`
	ProfitFactor             string            `json:"profit_factor"` // "" = undefined
	PayoffRatio              string            `json:"payoff_ratio"`
	Expectancy               string            `json:"expectancy"`
	MaxDrawdown              string            `json:"max_drawdown"`
	AverageDrawdown          string            `json:"average_drawdown"`
	RecoveryFactor           string            `json:"recovery_factor"`
	SharpeRatio              string            `json:"sharpe_ratio"` // "" = undefined
	SortinoRatio             string            `json:"sortino_ratio"`
	CalmarRatio              string            `json:"calmar_ratio"`
	Volatility               string            `json:"volatility"`
	ExposureTimePct          string            `json:"exposure_time_pct"`
	AverageHoldingPeriodSecs string            `json:"average_holding_period_secs"`
	MaxConsecutiveWins       uint32            `json:"max_consecutive_wins"`
	MaxConsecutiveLosses     uint32            `json:"max_consecutive_losses"`
	BreakEvenTrades          uint32            `json:"break_even_trades"`
	TotalFees                string            `json:"total_fees"`
	TotalSlippage            string            `json:"total_slippage"`
	FinalEquity              string            `json:"final_equity"`
	BrierScore               string            `json:"brier_score"`
	LogLoss                  string            `json:"log_loss"`
	Precision                string            `json:"precision"`
	Recall                   string            `json:"recall"`
	F1Score                  string            `json:"f1_score"`
	CalibrationError         string            `json:"calibration_error"`
	PredictionCoverage       string            `json:"prediction_coverage"`
	PerformanceByRegime      map[string]string `json:"performance_by_regime,omitempty"`
	PerformanceByStrategy    map[string]string `json:"performance_by_strategy,omitempty"`
	PerformanceByModel       map[string]string `json:"performance_by_model,omitempty"`
}

type TradeRecord struct {
	TradeID    string    `json:"trade_id"`
	Side       string    `json:"side"` // "BUY" | "SELL"
	EntryPrice string    `json:"entry_price"`
	ExitPrice  string    `json:"exit_price"`
	Quantity   string    `json:"quantity"`
	EnteredAt  time.Time `json:"entered_at"`
	ExitedAt   time.Time `json:"exited_at"`
	Pnl        string    `json:"pnl"`
	Fees       string    `json:"fees"`
	Slippage   string    `json:"slippage"`
	ExitReason string    `json:"exit_reason"`
	MFE        string    `json:"mfe"`
	MAE        string    `json:"mae"`
}

type EquityPoint struct {
	At     time.Time `json:"at"`
	Equity string    `json:"equity"`
}

// JobStatus values, mapped 1:1 from quant-engine's JobStatus enum.
const (
	JobStatusUnspecified = "UNSPECIFIED"
	JobStatusQueued      = "QUEUED"
	JobStatusRunning     = "RUNNING"
	JobStatusCompleted   = "COMPLETED"
	JobStatusFailed      = "FAILED"
	JobStatusCancelled   = "CANCELLED"
)

type RunBacktestInput struct {
	Config  BacktestConfig
	Candles []market.Candle
}

type RunBacktestResult struct {
	BacktestID string `json:"backtest_id"`
	Status     string `json:"status"`
}

type GetBacktestResultResult struct {
	BacktestID   string           `json:"backtest_id"`
	Status       string           `json:"status"`
	Metrics      *BacktestMetrics `json:"metrics,omitempty"`
	Trades       []TradeRecord    `json:"trades,omitempty"`
	EquityCurve  []EquityPoint    `json:"equity_curve,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

type BacktestProgress struct {
	BacktestID       string    `json:"backtest_id"`
	Status           string    `json:"status"`
	PercentComplete  string    `json:"percent_complete"`
	CandlesProcessed uint64    `json:"candles_processed"`
	TotalCandles     uint64    `json:"total_candles"`
	TradesSoFar      uint64    `json:"trades_so_far"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// BacktestService is the port to quant-engine's backtest job lifecycle:
// submit (RunBacktest), poll (GetBacktestResult), and follow progress
// (StreamBacktestProgress). Segregated from QuantEngine (the realtime
// signal path) since callers of one rarely need the other.
type BacktestService interface {
	RunBacktest(ctx context.Context, input RunBacktestInput) (RunBacktestResult, error)
	GetBacktestResult(ctx context.Context, backtestID string) (GetBacktestResultResult, error)
	// StreamBacktestProgress invokes onProgress once per update received
	// from the server-streaming RPC. Returning a non-nil error from
	// onProgress stops iteration and is propagated to the caller (e.g.
	// the HTTP handler uses this to stop when the client disconnects).
	StreamBacktestProgress(ctx context.Context, backtestID string, onProgress func(BacktestProgress) error) error
}
