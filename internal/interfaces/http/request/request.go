// Package request holds the JSON request bodies handlers decode.
package request

// ReasonRequest is the body accepted by every /v1/system/* control
// endpoint: a free-text operator-supplied reason, persisted into the audit
// trail (operation.State / SystemIncident).
type ReasonRequest struct {
	Reason string `json:"reason"`
}

// KillSwitchRequest additionally lets the caller decide whether closing
// existing positions stays allowed while the kill switch is active.
type KillSwitchRequest struct {
	Reason     string `json:"reason"`
	AllowClose bool   `json:"allow_close"`
}

// CandleRequest is one OHLCV bar, as accepted by every quant-engine
// capability endpoint that needs candle history (backtest, optimization,
// feature calculation). Trading-core does not fetch historical candles on
// the caller's behalf — same as quant-engine's own gRPC contract, the
// caller supplies the window it wants evaluated.
type CandleRequest struct {
	OpenTime  string `json:"open_time"`  // RFC3339
	CloseTime string `json:"close_time"` // RFC3339
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
}

// BacktestConfigRequest mirrors quant-engine's BacktestConfig wire message.
type BacktestConfigRequest struct {
	Symbol                  string            `json:"symbol"`
	Timeframe               string            `json:"timeframe"`
	StrategyName            string            `json:"strategy_name"`
	StrategyVersion         string            `json:"strategy_version"`
	StrategyParams          map[string]string `json:"strategy_params"`
	InitialCapital          string            `json:"initial_capital"`
	FeeRate                 string            `json:"fee_rate"`
	Slippage                string            `json:"slippage"`
	ExecutionModel          string            `json:"execution_model"`
	From                    string            `json:"from"` // RFC3339
	To                      string            `json:"to"`   // RFC3339
	AllowShort              bool              `json:"allow_short"`
	ModelName               string            `json:"model_name"`
	ModelVersion            string            `json:"model_version"`
	GenerateTrainingRecords bool              `json:"generate_training_records"`
}

// BacktestRequest is the body of POST /v1/backtest/request.
type BacktestRequest struct {
	Config  BacktestConfigRequest `json:"config"`
	Candles []CandleRequest       `json:"candles"`
}

// ParameterRangeRequest is one dimension of an optimization search space.
type ParameterRangeRequest struct {
	Name    string   `json:"name"`
	Min     string   `json:"min"`
	Max     string   `json:"max"`
	Step    string   `json:"step"`
	Choices []string `json:"choices"`
}

// OptimizationRequest is the body of POST /v1/optimization/request.
type OptimizationRequest struct {
	BaseConfig         BacktestConfigRequest   `json:"base_config"`
	Candles            []CandleRequest         `json:"candles"`
	SearchSpace        []ParameterRangeRequest `json:"search_space"`
	Algorithm          string                  `json:"algorithm"` // "GRID" | "RANDOM"
	Objective          string                  `json:"objective"`
	MaxIterations      uint32                  `json:"max_iterations"`
	WalkForward        bool                    `json:"walk_forward"`
	WalkForwardWindows uint32                  `json:"walk_forward_windows"`
	MonteCarlo         bool                    `json:"monte_carlo"`
	MonteCarloRuns     uint32                  `json:"monte_carlo_runs"`
}

// DatasetExportRequest is the body of POST /v1/dataset/export.
type DatasetExportRequest struct {
	DatasetName    string   `json:"dataset_name"`
	DatasetVersion string   `json:"dataset_version"`
	Symbols        []string `json:"symbols"`
	Timeframes     []string `json:"timeframes"`
	From           string   `json:"from"` // RFC3339
	To             string   `json:"to"`   // RFC3339
	LabelVersion   string   `json:"label_version"`
}

// ValidateStrategyRequest is the body of POST /v1/strategy/validate.
type ValidateStrategyRequest struct {
	StrategyName    string            `json:"strategy_name"`
	StrategyVersion string            `json:"strategy_version"`
	Params          map[string]string `json:"params"`
}

// CalculateFeaturesRequest is the body of POST /v1/quant/features.
type CalculateFeaturesRequest struct {
	Symbol               string          `json:"symbol"`
	Timeframe            string          `json:"timeframe"`
	Candles              []CandleRequest `json:"candles"`
	FeatureSchemaVersion string          `json:"feature_schema_version"`
}

// FeatureValueRequest is one feature name/value pair fed into EvaluateModel.
type FeatureValueRequest struct {
	Name             string `json:"name"`
	Kind             string `json:"kind"` // "numeric" | "categorical" | "bool"
	NumericValue     string `json:"numeric_value"`
	CategoricalValue string `json:"categorical_value"`
	BoolValue        bool   `json:"bool_value"`
}

// EvaluateModelRequest is the body of POST /v1/quant/model/evaluate.
type EvaluateModelRequest struct {
	ModelName            string                `json:"model_name"`
	ModelVersion         string                `json:"model_version"`
	Features             []FeatureValueRequest `json:"features"`
	FeatureSchemaVersion string                `json:"feature_schema_version"`
}

// RegisterDecisionOutcomeRequest is the body of
// POST /v1/decisions/{decision_id}/outcome.
type RegisterDecisionOutcomeRequest struct {
	OrderCreated    bool   `json:"order_created"`
	Executed        bool   `json:"executed"`
	RejectionReason string `json:"rejection_reason"`
	EntryPrice      string `json:"entry_price"`
	ExitPrice       string `json:"exit_price"`
	Quantity        string `json:"quantity"`
	Fees            string `json:"fees"`
	Slippage        string `json:"slippage"`
	Pnl             string `json:"pnl"`
	ExitReason      string `json:"exit_reason"`
	EntryAt         string `json:"entry_at"` // RFC3339
	ExitAt          string `json:"exit_at"`  // RFC3339
}

// ReloadApprovedModelRequest is the body of POST /v1/quant/model/reload.
type ReloadApprovedModelRequest struct {
	ModelName string `json:"model_name"`
	Stage     string `json:"stage"` // "APPROVED" | "CANARY" | "PRODUCTION"
}
