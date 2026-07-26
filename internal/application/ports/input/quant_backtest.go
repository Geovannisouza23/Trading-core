package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
)

// BacktestService groups the backtest job lifecycle (submit/poll/stream)
// behind one port, mirroring QueryService's "cohesive group of thin
// pass-through operations" shape — none of these three carry business
// logic beyond validating input and delegating to output.BacktestService.
type BacktestService interface {
	RunBacktest(ctx context.Context, cmd command.RunBacktestCommand) (output.RunBacktestResult, error)
	GetBacktestResult(ctx context.Context, q query.GetBacktestResultQuery) (output.GetBacktestResultResult, error)
	StreamBacktestProgress(ctx context.Context, q query.StreamBacktestProgressQuery, onProgress func(output.BacktestProgress) error) error
}
