// Package command holds the write-intent inputs accepted by use cases.
package command

import (
	"time"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

// EvaluateMarketSignalCommand carries a freshly closed candle (plus recent
// history for context) into the signal evaluation pipeline.
type EvaluateMarketSignalCommand struct {
	Candle        market.Candle
	RecentCandles []market.Candle
}

// EvaluateRiskCommand triggers risk evaluation for an already-persisted
// signal. CurrentPrice is the latest known market price at evaluation time
// (typically the close of the candle that produced the signal), used to
// measure slippage against the signal's proposed entry price.
type EvaluateRiskCommand struct {
	SignalID     shared.SignalID
	CurrentPrice shared.Price
}

// ExecuteApprovedOrderCommand triggers order execution for an
// already-persisted, allowed risk decision.
type ExecuteApprovedOrderCommand struct {
	RiskDecisionID shared.RiskDecisionID
}

// ReconcileBrokerStateCommand triggers a reconciliation pass. Reason is
// informational (e.g. "scheduled", "manual", "post-kill-switch").
type ReconcileBrokerStateCommand struct {
	Reason string
}

// ProcessMarketEventCommand carries a raw news item into the event
// intelligence pipeline.
type ProcessMarketEventCommand struct {
	Source      string
	URL         string
	Title       string
	Content     string
	PublishedAt time.Time
}

// ActivateKillSwitchCommand forces the system into ModeKillSwitch.
type ActivateKillSwitchCommand struct {
	ActivatedBy string
	Origin      string
	Reason      string
	AllowClose  bool
}

// ChangeOperationalModeCommand requests a controlled operational mode
// transition. RealConfirmationToken is only inspected when TargetMode is
// REAL.
type ChangeOperationalModeCommand struct {
	TargetMode            string
	ActedBy               string
	Origin                string
	Reason                string
	RealConfirmationToken string
}

// --- Quant Engine capability commands (backtest/optimization/dataset/
// diagnostics) -------------------------------------------------------------
//
// These reuse output.* types directly (BacktestConfig, ParameterRange, ...)
// rather than redeclaring field-for-field duplicates: they are plain
// attribute bags with no behavior, and they represent literally the same
// wire concept on both sides of the use case boundary — a second
// independent copy would only be able to drift out of sync with the first,
// never evolve meaningfully apart from it.

// RunBacktestCommand submits a new backtest job.
type RunBacktestCommand struct {
	Config  output.BacktestConfig
	Candles []market.Candle
}

// RunOptimizationCommand submits a new hyperparameter search job.
type RunOptimizationCommand struct {
	BaseConfig         output.BacktestConfig
	Candles            []market.Candle
	SearchSpace        []output.ParameterRange
	Algorithm          string
	Objective          string
	MaxIterations      uint32
	WalkForward        bool
	WalkForwardWindows uint32
	MonteCarlo         bool
	MonteCarloRuns     uint32
}

// ExportDatasetCommand submits a Parquet dataset export job.
type ExportDatasetCommand struct {
	DatasetName    string
	DatasetVersion string
	Symbols        []string
	Timeframes     []string
	StartAt        time.Time
	EndAt          time.Time
	LabelVersion   string
}

// ValidateStrategyCommand asks quant-engine whether a strategy
// name/version/params combination is valid and usable.
type ValidateStrategyCommand struct {
	StrategyName    string
	StrategyVersion string
	Params          map[string]string
}

// CalculateFeaturesCommand asks quant-engine to compute the v1 feature
// vector for a symbol/timeframe/candle window.
type CalculateFeaturesCommand struct {
	Symbol               string
	Timeframe            string
	Candles              []market.Candle
	FeatureSchemaVersion string
}

// EvaluateModelCommand asks quant-engine to run a model directly against
// an already-computed feature vector.
type EvaluateModelCommand struct {
	ModelName            string
	ModelVersion         string
	Features             []output.FeatureValue
	FeatureSchemaVersion string
}

// RegisterDecisionOutcomeCommand reports the realized outcome of a
// decision back to quant-engine for training-data purposes. The outcome
// is caller-supplied in this delivery — trading-core does not yet
// auto-trigger this from order reconciliation (see
// docs/decisions/0010-real-grpc-client-for-quant-engine.md).
type RegisterDecisionOutcomeCommand struct {
	DecisionID      string
	OrderCreated    bool
	Executed        bool
	RejectionReason string
	EntryPrice      string
	ExitPrice       string
	Quantity        string
	Fees            string
	Slippage        string
	Pnl             string
	ExitReason      string
	EntryAt         time.Time
	ExitAt          time.Time
}

// ReloadApprovedModelCommand asks quant-engine to validate and reload the
// currently APPROVED/CANARY/PRODUCTION model for the given stage.
type ReloadApprovedModelCommand struct {
	ModelName string
	Stage     string
}
