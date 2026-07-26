package grpc

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"trading-core/internal/application/ports/output"
	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
)

var executionModelToProto = map[string]quantv1.ExecutionModel{
	"":             quantv1.ExecutionModel_EXECUTION_MODEL_UNSPECIFIED,
	"CLOSE_PRICE":  quantv1.ExecutionModel_EXECUTION_MODEL_CLOSE_PRICE,
	"NEXT_OPEN":    quantv1.ExecutionModel_EXECUTION_MODEL_NEXT_OPEN,
	"OHLC":         quantv1.ExecutionModel_EXECUTION_MODEL_OHLC,
	"CONSERVATIVE": quantv1.ExecutionModel_EXECUTION_MODEL_CONSERVATIVE,
}

func jobStatusFromProto(status quantv1.JobStatus) string {
	switch status {
	case quantv1.JobStatus_JOB_STATUS_QUEUED:
		return output.JobStatusQueued
	case quantv1.JobStatus_JOB_STATUS_RUNNING:
		return output.JobStatusRunning
	case quantv1.JobStatus_JOB_STATUS_COMPLETED:
		return output.JobStatusCompleted
	case quantv1.JobStatus_JOB_STATUS_FAILED:
		return output.JobStatusFailed
	case quantv1.JobStatus_JOB_STATUS_CANCELLED:
		return output.JobStatusCancelled
	default:
		return output.JobStatusUnspecified
	}
}

func backtestConfigToProto(cfg output.BacktestConfig) *quantv1.BacktestConfig {
	return &quantv1.BacktestConfig{
		Symbol:                  cfg.Symbol,
		Timeframe:               cfg.Timeframe,
		StrategyName:            cfg.StrategyName,
		StrategyVersion:         cfg.StrategyVersion,
		StrategyParams:          cfg.StrategyParams,
		InitialCapital:          cfg.InitialCapital,
		FeeRate:                 cfg.FeeRate,
		Slippage:                cfg.Slippage,
		ExecutionModel:          executionModelToProto[cfg.ExecutionModel],
		StartAt:                 timestamppb.New(cfg.StartAt),
		EndAt:                   timestamppb.New(cfg.EndAt),
		AllowShort:              cfg.AllowShort,
		ModelName:               cfg.ModelName,
		ModelVersion:            cfg.ModelVersion,
		GenerateTrainingRecords: cfg.GenerateTrainingRecords,
	}
}

func backtestMetricsFromProto(m *quantv1.BacktestMetrics) *output.BacktestMetrics {
	if m == nil {
		return nil
	}
	return &output.BacktestMetrics{
		TotalReturn:              m.GetTotalReturn(),
		NetProfit:                m.GetNetProfit(),
		GrossProfit:              m.GetGrossProfit(),
		GrossLoss:                m.GetGrossLoss(),
		WinRate:                  m.GetWinRate(),
		LossRate:                 m.GetLossRate(),
		TotalTrades:              m.GetTotalTrades(),
		WinningTrades:            m.GetWinningTrades(),
		LosingTrades:             m.GetLosingTrades(),
		AverageTrade:             m.GetAverageTrade(),
		AverageWinner:            m.GetAverageWinner(),
		AverageLoser:             m.GetAverageLoser(),
		LargestWinner:            m.GetLargestWinner(),
		LargestLoser:             m.GetLargestLoser(),
		ProfitFactor:             m.GetProfitFactor(),
		PayoffRatio:              m.GetPayoffRatio(),
		Expectancy:               m.GetExpectancy(),
		MaxDrawdown:              m.GetMaxDrawdown(),
		AverageDrawdown:          m.GetAverageDrawdown(),
		RecoveryFactor:           m.GetRecoveryFactor(),
		SharpeRatio:              m.GetSharpeRatio(),
		SortinoRatio:             m.GetSortinoRatio(),
		CalmarRatio:              m.GetCalmarRatio(),
		Volatility:               m.GetVolatility(),
		ExposureTimePct:          m.GetExposureTimePct(),
		AverageHoldingPeriodSecs: m.GetAverageHoldingPeriodSecs(),
		MaxConsecutiveWins:       m.GetMaxConsecutiveWins(),
		MaxConsecutiveLosses:     m.GetMaxConsecutiveLosses(),
		BreakEvenTrades:          m.GetBreakEvenTrades(),
		TotalFees:                m.GetTotalFees(),
		TotalSlippage:            m.GetTotalSlippage(),
		FinalEquity:              m.GetFinalEquity(),
		BrierScore:               m.GetBrierScore(),
		LogLoss:                  m.GetLogLoss(),
		Precision:                m.GetPrecision(),
		Recall:                   m.GetRecall(),
		F1Score:                  m.GetF1Score(),
		CalibrationError:         m.GetCalibrationError(),
		PredictionCoverage:       m.GetPredictionCoverage(),
		PerformanceByRegime:      m.GetPerformanceByRegime(),
		PerformanceByStrategy:    m.GetPerformanceByStrategy(),
		PerformanceByModel:       m.GetPerformanceByModel(),
	}
}

func tradeRecordFromProto(t *quantv1.TradeRecord) output.TradeRecord {
	side := "SELL"
	if t.GetSide() == quantv1.Action_ACTION_BUY {
		side = "BUY"
	}
	return output.TradeRecord{
		TradeID:    t.GetTradeId(),
		Side:       side,
		EntryPrice: t.GetEntryPrice(),
		ExitPrice:  t.GetExitPrice(),
		Quantity:   t.GetQuantity(),
		EnteredAt:  t.GetEnteredAt().AsTime(),
		ExitedAt:   t.GetExitedAt().AsTime(),
		Pnl:        t.GetPnl(),
		Fees:       t.GetFees(),
		Slippage:   t.GetSlippage(),
		ExitReason: t.GetExitReason(),
		MFE:        t.GetMfe(),
		MAE:        t.GetMae(),
	}
}

func tradeRecordsFromProto(records []*quantv1.TradeRecord) []output.TradeRecord {
	out := make([]output.TradeRecord, len(records))
	for i, r := range records {
		out[i] = tradeRecordFromProto(r)
	}
	return out
}

func equityCurveFromProto(points []*quantv1.EquityPoint) []output.EquityPoint {
	out := make([]output.EquityPoint, len(points))
	for i, p := range points {
		out[i] = output.EquityPoint{At: p.GetAt().AsTime(), Equity: p.GetEquity()}
	}
	return out
}

func (c *Client) RunBacktest(ctx context.Context, input output.RunBacktestInput) (output.RunBacktestResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.RunBacktest(ctx, &quantv1.RunBacktestRequest{
		RequestId: uuid.NewString(),
		Config:    backtestConfigToProto(input.Config),
		Candles:   candlesToProto(input.Candles),
	})
	if err != nil {
		return output.RunBacktestResult{}, fmt.Errorf("quant engine RunBacktest: %w", err)
	}
	return output.RunBacktestResult{
		BacktestID: resp.GetBacktestId(),
		Status:     jobStatusFromProto(resp.GetStatus()),
	}, nil
}

func (c *Client) GetBacktestResult(ctx context.Context, backtestID string) (output.GetBacktestResultResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.GetBacktestResult(ctx, &quantv1.GetBacktestResultRequest{BacktestId: backtestID})
	if err != nil {
		return output.GetBacktestResultResult{}, fmt.Errorf("quant engine GetBacktestResult: %w", err)
	}

	result := output.GetBacktestResultResult{
		BacktestID:   resp.GetBacktestId(),
		Status:       jobStatusFromProto(resp.GetStatus()),
		Metrics:      backtestMetricsFromProto(resp.GetMetrics()),
		Trades:       tradeRecordsFromProto(resp.GetTrades()),
		EquityCurve:  equityCurveFromProto(resp.GetEquityCurve()),
		ErrorMessage: resp.GetErrorMessage(),
	}
	if resp.GetCompletedAt() != nil {
		completedAt := resp.GetCompletedAt().AsTime()
		result.CompletedAt = &completedAt
	}
	return result, nil
}

func (c *Client) StreamBacktestProgress(ctx context.Context, backtestID string, onProgress func(output.BacktestProgress) error) error {
	stream, err := c.stub.StreamBacktestProgress(ctx, &quantv1.StreamBacktestProgressRequest{BacktestId: backtestID})
	if err != nil {
		return fmt.Errorf("quant engine StreamBacktestProgress: %w", err)
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("quant engine StreamBacktestProgress: %w", err)
		}

		progress := output.BacktestProgress{
			BacktestID:       msg.GetBacktestId(),
			Status:           jobStatusFromProto(msg.GetStatus()),
			PercentComplete:  msg.GetPercentComplete(),
			CandlesProcessed: msg.GetCandlesProcessed(),
			TotalCandles:     msg.GetTotalCandles(),
			TradesSoFar:      msg.GetTradesSoFar(),
			UpdatedAt:        msg.GetUpdatedAt().AsTime(),
		}
		if err := onProgress(progress); err != nil {
			return err
		}
	}
}

var _ output.BacktestService = (*Client)(nil)
