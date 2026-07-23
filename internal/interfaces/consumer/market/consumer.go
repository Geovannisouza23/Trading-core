// Package market receives closed candles (from a live feed or a manual/
// test trigger) and drives them through the signal -> risk -> execution
// pipeline. It only sequences calls to use cases; every decision (whether
// a signal exists, whether risk allows it, how the order executes) is made
// inside the use cases themselves.
package market

import (
	"context"
	"log/slog"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/domain/shared"
)

type Consumer struct {
	evaluateSignal input.EvaluateMarketSignalUseCase
	evaluateRisk   input.EvaluateRiskUseCase
	executeOrder   input.ExecuteApprovedOrderUseCase
	logger         *slog.Logger
}

func NewConsumer(
	evaluateSignal input.EvaluateMarketSignalUseCase,
	evaluateRisk input.EvaluateRiskUseCase,
	executeOrder input.ExecuteApprovedOrderUseCase,
	logger *slog.Logger,
) *Consumer {
	return &Consumer{evaluateSignal: evaluateSignal, evaluateRisk: evaluateRisk, executeOrder: executeOrder, logger: logger}
}

// HandleCandle runs the full pipeline for a single closed candle. Every
// stage's failure is logged and stops the pipeline for this candle only —
// it never blocks the feed from processing the next one.
func (c *Consumer) HandleCandle(ctx context.Context, cmd command.EvaluateMarketSignalCommand) {
	symbol := cmd.Candle.Symbol.String()

	signalResult, err := c.evaluateSignal.Execute(ctx, cmd)
	if err != nil {
		c.logger.Warn("market consumer: signal evaluation failed", "symbol", symbol, "error", err)
		return
	}
	if !signalResult.SignalCreated {
		return
	}
	c.logger.Info("market consumer: signal created", "symbol", symbol, "side", signalResult.Signal.Side)

	signalID, err := shared.ParseSignalID(signalResult.Signal.ID)
	if err != nil {
		c.logger.Error("market consumer: invalid signal id returned by use case", "error", err)
		return
	}

	riskResult, err := c.evaluateRisk.Execute(ctx, command.EvaluateRiskCommand{
		SignalID:     signalID,
		CurrentPrice: cmd.Candle.Close,
	})
	if err != nil {
		c.logger.Warn("market consumer: risk evaluation failed", "symbol", symbol, "error", err)
		return
	}
	if !riskResult.Decision.Allowed {
		c.logger.Info("market consumer: risk decision blocked signal", "symbol", symbol, "reasons", riskResult.Decision.ReasonCodes)
		return
	}

	riskDecisionID, err := shared.ParseRiskDecisionID(riskResult.Decision.ID)
	if err != nil {
		c.logger.Error("market consumer: invalid risk decision id returned by use case", "error", err)
		return
	}

	executeResult, err := c.executeOrder.Execute(ctx, command.ExecuteApprovedOrderCommand{RiskDecisionID: riskDecisionID})
	if err != nil {
		c.logger.Warn("market consumer: order execution failed", "symbol", symbol, "error", err)
		return
	}
	if executeResult.Skipped {
		c.logger.Info("market consumer: order execution skipped", "symbol", symbol, "reason", executeResult.SkipReason)
		return
	}
	c.logger.Info("market consumer: order executed", "symbol", symbol, "order_id", executeResult.Order.ID, "status", executeResult.Order.Status)
}
