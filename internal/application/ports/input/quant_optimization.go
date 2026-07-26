package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
)

// OptimizationService groups the hyperparameter search job lifecycle
// (submit/poll) behind one port.
type OptimizationService interface {
	RunOptimization(ctx context.Context, cmd command.RunOptimizationCommand) (output.RunOptimizationResult, error)
	GetOptimizationResult(ctx context.Context, q query.GetOptimizationResultQuery) (output.GetOptimizationResultResult, error)
}
