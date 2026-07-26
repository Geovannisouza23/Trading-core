package usecase

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
	"trading-core/internal/domain/shared"
)

// BacktestService implements input.BacktestService by delegating straight
// to output.BacktestService — no business logic beyond a blank-ID guard,
// same shape as QueryService.
type BacktestService struct {
	backtests output.BacktestService
}

func NewBacktestService(backtests output.BacktestService) *BacktestService {
	return &BacktestService{backtests: backtests}
}

var _ input.BacktestService = (*BacktestService)(nil)

func (s *BacktestService) RunBacktest(ctx context.Context, cmd command.RunBacktestCommand) (output.RunBacktestResult, error) {
	if cmd.Config.Symbol == "" {
		return output.RunBacktestResult{}, shared.NewValidationError("symbol", "must not be empty")
	}
	return s.backtests.RunBacktest(ctx, output.RunBacktestInput{Config: cmd.Config, Candles: cmd.Candles})
}

func (s *BacktestService) GetBacktestResult(ctx context.Context, q query.GetBacktestResultQuery) (output.GetBacktestResultResult, error) {
	if q.BacktestID == "" {
		return output.GetBacktestResultResult{}, shared.NewValidationError("backtest_id", "must not be empty")
	}
	return s.backtests.GetBacktestResult(ctx, q.BacktestID)
}

func (s *BacktestService) StreamBacktestProgress(ctx context.Context, q query.StreamBacktestProgressQuery, onProgress func(output.BacktestProgress) error) error {
	if q.BacktestID == "" {
		return shared.NewValidationError("backtest_id", "must not be empty")
	}
	return s.backtests.StreamBacktestProgress(ctx, q.BacktestID, onProgress)
}
