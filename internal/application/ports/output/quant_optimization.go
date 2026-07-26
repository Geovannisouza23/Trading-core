package output

import (
	"context"
	"time"

	"trading-core/internal/domain/market"
)

// ParameterRange describes one dimension of a RunOptimization search
// space: either a continuous min/max/step range or a discrete Choices
// set (mutually exclusive, matching the wire contract).
type ParameterRange struct {
	Name    string   `json:"name"`
	Min     string   `json:"min,omitempty"`
	Max     string   `json:"max,omitempty"`
	Step    string   `json:"step,omitempty"`
	Choices []string `json:"choices,omitempty"`
}

type RunOptimizationInput struct {
	BaseConfig         BacktestConfig
	Candles            []market.Candle
	SearchSpace        []ParameterRange
	Algorithm          string
	Objective          string
	MaxIterations      uint32
	WalkForward        bool
	WalkForwardWindows uint32
	MonteCarlo         bool
	MonteCarloRuns     uint32
}

type RunOptimizationResult struct {
	OptimizationID string `json:"optimization_id"`
	Status         string `json:"status"`
}

type ParameterAssignment struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type OptimizationCandidateResult struct {
	Parameters []ParameterAssignment `json:"parameters"`
	Score      string                `json:"score"`
	Metrics    *BacktestMetrics      `json:"metrics,omitempty"`
}

type GetOptimizationResultResult struct {
	OptimizationID string                        `json:"optimization_id"`
	Status         string                        `json:"status"`
	Best           *OptimizationCandidateResult  `json:"best,omitempty"`
	TopResults     []OptimizationCandidateResult `json:"top_results,omitempty"`
	ErrorMessage   string                        `json:"error_message,omitempty"`
	CompletedAt    *time.Time                    `json:"completed_at,omitempty"`
}

// OptimizationService is the port to quant-engine's hyperparameter
// search job lifecycle: submit (RunOptimization), poll
// (GetOptimizationResult). There is no streaming-progress RPC for
// optimization on the wire contract (unlike backtest).
type OptimizationService interface {
	RunOptimization(ctx context.Context, input RunOptimizationInput) (RunOptimizationResult, error)
	GetOptimizationResult(ctx context.Context, optimizationID string) (GetOptimizationResultResult, error)
}
