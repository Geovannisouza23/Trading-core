package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"trading-core/internal/application/ports/output"
	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
)

var searchAlgorithmToProto = map[string]quantv1.SearchAlgorithm{
	"":       quantv1.SearchAlgorithm_SEARCH_ALGORITHM_UNSPECIFIED,
	"GRID":   quantv1.SearchAlgorithm_SEARCH_ALGORITHM_GRID,
	"RANDOM": quantv1.SearchAlgorithm_SEARCH_ALGORITHM_RANDOM,
}

var optimizationObjectiveToProto = map[string]quantv1.OptimizationObjective{
	"":                     quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_UNSPECIFIED,
	"NET_PROFIT":           quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_NET_PROFIT,
	"PROFIT_FACTOR":        quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_PROFIT_FACTOR,
	"SHARPE":               quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_SHARPE,
	"SORTINO":              quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_SORTINO,
	"MIN_DRAWDOWN":         quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_MIN_DRAWDOWN,
	"RISK_ADJUSTED_RETURN": quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_RISK_ADJUSTED_RETURN,
	"STABILITY":            quantv1.OptimizationObjective_OPTIMIZATION_OBJECTIVE_STABILITY,
}

func parameterRangesToProto(ranges []output.ParameterRange) []*quantv1.ParameterRange {
	out := make([]*quantv1.ParameterRange, len(ranges))
	for i, r := range ranges {
		out[i] = &quantv1.ParameterRange{
			Name:    r.Name,
			Min:     r.Min,
			Max:     r.Max,
			Step:    r.Step,
			Choices: r.Choices,
		}
	}
	return out
}

func optimizationCandidateFromProto(c *quantv1.OptimizationCandidateResult) *output.OptimizationCandidateResult {
	if c == nil {
		return nil
	}
	params := make([]output.ParameterAssignment, len(c.GetParameters()))
	for i, p := range c.GetParameters() {
		params[i] = output.ParameterAssignment{Name: p.GetName(), Value: p.GetValue()}
	}
	return &output.OptimizationCandidateResult{
		Parameters: params,
		Score:      c.GetScore(),
		Metrics:    backtestMetricsFromProto(c.GetMetrics()),
	}
}

func (c *Client) RunOptimization(ctx context.Context, input output.RunOptimizationInput) (output.RunOptimizationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.RunOptimization(ctx, &quantv1.RunOptimizationRequest{
		RequestId:          uuid.NewString(),
		BaseConfig:         backtestConfigToProto(input.BaseConfig),
		Candles:            candlesToProto(input.Candles),
		SearchSpace:        parameterRangesToProto(input.SearchSpace),
		Algorithm:          searchAlgorithmToProto[input.Algorithm],
		Objective:          optimizationObjectiveToProto[input.Objective],
		MaxIterations:      input.MaxIterations,
		WalkForward:        input.WalkForward,
		WalkForwardWindows: input.WalkForwardWindows,
		MonteCarlo:         input.MonteCarlo,
		MonteCarloRuns:     input.MonteCarloRuns,
	})
	if err != nil {
		return output.RunOptimizationResult{}, fmt.Errorf("quant engine RunOptimization: %w", err)
	}
	return output.RunOptimizationResult{
		OptimizationID: resp.GetOptimizationId(),
		Status:         jobStatusFromProto(resp.GetStatus()),
	}, nil
}

func (c *Client) GetOptimizationResult(ctx context.Context, optimizationID string) (output.GetOptimizationResultResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.GetOptimizationResult(ctx, &quantv1.GetOptimizationResultRequest{OptimizationId: optimizationID})
	if err != nil {
		return output.GetOptimizationResultResult{}, fmt.Errorf("quant engine GetOptimizationResult: %w", err)
	}

	topResults := make([]output.OptimizationCandidateResult, len(resp.GetTopResults()))
	for i, r := range resp.GetTopResults() {
		topResults[i] = *optimizationCandidateFromProto(r)
	}

	result := output.GetOptimizationResultResult{
		OptimizationID: resp.GetOptimizationId(),
		Status:         jobStatusFromProto(resp.GetStatus()),
		Best:           optimizationCandidateFromProto(resp.GetBest()),
		TopResults:     topResults,
		ErrorMessage:   resp.GetErrorMessage(),
	}
	if resp.GetCompletedAt() != nil {
		completedAt := resp.GetCompletedAt().AsTime()
		result.CompletedAt = &completedAt
	}
	return result, nil
}

var _ output.OptimizationService = (*Client)(nil)
