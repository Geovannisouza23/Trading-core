package usecase

import (
	"context"
	"fmt"
	"time"

	"trading-core/internal/application/command"
	"trading-core/internal/application/mapper"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/signal"
)

// EvaluateMarketSignal implements input.EvaluateMarketSignalUseCase.
type EvaluateMarketSignal struct {
	quantEngine    output.QuantEngine
	signals        output.TradeSignalRepository
	eventBus       output.EventBus
	clock          output.Clock
	maxCandleAge   time.Duration
	signalValidity time.Duration
}

func NewEvaluateMarketSignal(
	quantEngine output.QuantEngine,
	signals output.TradeSignalRepository,
	eventBus output.EventBus,
	clock output.Clock,
	maxCandleAge time.Duration,
	signalValidity time.Duration,
) *EvaluateMarketSignal {
	return &EvaluateMarketSignal{
		quantEngine:    quantEngine,
		signals:        signals,
		eventBus:       eventBus,
		clock:          clock,
		maxCandleAge:   maxCandleAge,
		signalValidity: signalValidity,
	}
}

var _ input.EvaluateMarketSignalUseCase = (*EvaluateMarketSignal)(nil)

func (uc *EvaluateMarketSignal) Execute(ctx context.Context, cmd command.EvaluateMarketSignalCommand) (input.EvaluateMarketSignalResult, error) {
	now := uc.clock.Now()

	if cmd.Candle.AgeExceeds(now, uc.maxCandleAge) {
		return input.EvaluateMarketSignalResult{}, fmt.Errorf("%w: candle closed at %s", ErrStaleMarketData, cmd.Candle.CloseTime)
	}

	quantSignal, err := uc.quantEngine.EvaluateSignal(ctx, output.QuantInput{
		Candle:        cmd.Candle,
		RecentCandles: cmd.RecentCandles,
	})
	if err != nil {
		return input.EvaluateMarketSignalResult{}, fmt.Errorf("quant engine evaluation failed: %w", err)
	}
	if !quantSignal.HasSignal {
		return input.EvaluateMarketSignalResult{SignalCreated: false}, nil
	}

	tradeSignal, err := signal.New(
		quantSignal.Symbol,
		quantSignal.Side,
		quantSignal.EntryPrice,
		quantSignal.StopPrice,
		quantSignal.TargetPrice,
		quantSignal.Confidence,
		quantSignal.StrategyName,
		quantSignal.Regime,
		now,
		now.Add(uc.signalValidity),
	)
	if err != nil {
		return input.EvaluateMarketSignalResult{}, fmt.Errorf("quant engine produced an invalid signal: %w", err)
	}

	if err := uc.signals.Save(ctx, tradeSignal); err != nil {
		return input.EvaluateMarketSignalResult{}, fmt.Errorf("persisting signal: %w", err)
	}

	if err := uc.eventBus.Publish(ctx, signal.SignalCreated{Signal: tradeSignal, OccurredAt_: now}); err != nil {
		return input.EvaluateMarketSignalResult{}, fmt.Errorf("publishing SignalCreated: %w", err)
	}

	signalDTO := mapper.ToSignalDTO(tradeSignal)
	return input.EvaluateMarketSignalResult{SignalCreated: true, Signal: &signalDTO}, nil
}
