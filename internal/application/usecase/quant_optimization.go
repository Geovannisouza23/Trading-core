package usecase

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
	"trading-core/internal/domain/shared"
)

// OptimizationService implements input.OptimizationService by delegating
// straight to output.OptimizationService.
type OptimizationService struct {
	optimizations output.OptimizationService
}

func NewOptimizationService(optimizations output.OptimizationService) *OptimizationService {
	return &OptimizationService{optimizations: optimizations}
}

var _ input.OptimizationService = (*OptimizationService)(nil)

func (s *OptimizationService) RunOptimization(ctx context.Context, cmd command.RunOptimizationCommand) (output.RunOptimizationResult, error) {
	if cmd.BaseConfig.Symbol == "" {
		return output.RunOptimizationResult{}, shared.NewValidationError("symbol", "must not be empty")
	}
	if len(cmd.SearchSpace) == 0 {
		return output.RunOptimizationResult{}, shared.NewValidationError("search_space", "must not be empty")
	}
	return s.optimizations.RunOptimization(ctx, output.RunOptimizationInput{
		BaseConfig:         cmd.BaseConfig,
		Candles:            cmd.Candles,
		SearchSpace:        cmd.SearchSpace,
		Algorithm:          cmd.Algorithm,
		Objective:          cmd.Objective,
		MaxIterations:      cmd.MaxIterations,
		WalkForward:        cmd.WalkForward,
		WalkForwardWindows: cmd.WalkForwardWindows,
		MonteCarlo:         cmd.MonteCarlo,
		MonteCarloRuns:     cmd.MonteCarloRuns,
	})
}

func (s *OptimizationService) GetOptimizationResult(ctx context.Context, q query.GetOptimizationResultQuery) (output.GetOptimizationResultResult, error) {
	if q.OptimizationID == "" {
		return output.GetOptimizationResultResult{}, shared.NewValidationError("optimization_id", "must not be empty")
	}
	return s.optimizations.GetOptimizationResult(ctx, q.OptimizationID)
}
