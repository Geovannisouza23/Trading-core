// Package grpc will eventually hold a gRPC client for the external Rust
// Quant Engine (contract: internal/contracts/grpc/quant/quant_engine.proto).
// Until that service exists, Engine is a small, fully deterministic fake
// implementing output.QuantEngine directly — no network, no protobuf.
package grpc

import (
	"context"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

// Engine is a deterministic Quant Engine stand-in: a simple momentum rule
// over the last two candles, with fixed stop/target offsets. It exists so
// the rest of the pipeline (risk, execution, dashboard) can be exercised
// end-to-end without the Rust service.
type Engine struct {
	momentumThreshold decimal.Decimal
	stopOffsetPct     decimal.Decimal
	targetOffsetPct   decimal.Decimal
}

// NewEngine builds a fake Quant Engine. Defaults are chosen so a handful of
// consecutive up (or down) candles reliably produce a signal in tests and
// local runs.
func NewEngine() *Engine {
	return &Engine{
		momentumThreshold: decimal.NewFromFloat(0.001), // 0.1% move triggers a signal
		stopOffsetPct:     decimal.NewFromFloat(0.01),  // 1% stop distance
		targetOffsetPct:   decimal.NewFromFloat(0.02),  // 2% target distance
	}
}

var _ output.QuantEngine = (*Engine)(nil)

func (e *Engine) EvaluateSignal(ctx context.Context, input output.QuantInput) (output.QuantSignal, error) {
	if len(input.RecentCandles) == 0 {
		return output.QuantSignal{HasSignal: false}, nil
	}
	previous := input.RecentCandles[len(input.RecentCandles)-1]
	current := input.Candle

	move := current.Close.Decimal().Sub(previous.Close.Decimal()).Div(previous.Close.Decimal())

	regime, _ := e.AnalyzeMarketRegime(ctx, output.MarketRegimeInput{Symbol: current.Symbol, RecentCandles: input.RecentCandles})

	switch {
	case move.GreaterThanOrEqual(e.momentumThreshold):
		return e.buildSignal(current, shared.SideBuy, regime)
	case move.LessThanOrEqual(e.momentumThreshold.Neg()):
		return e.buildSignal(current, shared.SideSell, regime)
	default:
		return output.QuantSignal{HasSignal: false}, nil
	}
}

func (e *Engine) buildSignal(candle market.Candle, side shared.Side, regime market.Regime) (output.QuantSignal, error) {
	entry := candle.Close
	var stopValue, targetValue decimal.Decimal
	if side == shared.SideBuy {
		stopValue = entry.Decimal().Mul(decimal.NewFromInt(1).Sub(e.stopOffsetPct))
		targetValue = entry.Decimal().Mul(decimal.NewFromInt(1).Add(e.targetOffsetPct))
	} else {
		stopValue = entry.Decimal().Mul(decimal.NewFromInt(1).Add(e.stopOffsetPct))
		targetValue = entry.Decimal().Mul(decimal.NewFromInt(1).Sub(e.targetOffsetPct))
	}

	stop, err := shared.NewPrice(stopValue)
	if err != nil {
		return output.QuantSignal{}, err
	}
	target, err := shared.NewPrice(targetValue)
	if err != nil {
		return output.QuantSignal{}, err
	}
	confidence := shared.MustNewConfidence(decimal.NewFromFloat(0.6))

	return output.QuantSignal{
		HasSignal:    true,
		Symbol:       candle.Symbol,
		Side:         side,
		EntryPrice:   entry,
		StopPrice:    stop,
		TargetPrice:  target,
		Confidence:   confidence,
		StrategyName: "fake-momentum",
		Regime:       regime,
	}, nil
}

func (e *Engine) AnalyzeMarketRegime(ctx context.Context, input output.MarketRegimeInput) (market.Regime, error) {
	if len(input.RecentCandles) < 2 {
		return market.RegimeUnknown, nil
	}
	first := input.RecentCandles[0].Close.Decimal()
	last := input.RecentCandles[len(input.RecentCandles)-1].Close.Decimal()
	if first.IsZero() {
		return market.RegimeUnknown, nil
	}
	change := last.Sub(first).Div(first).Abs()
	if change.GreaterThanOrEqual(decimal.NewFromFloat(0.02)) {
		return market.RegimeTrending, nil
	}
	return market.RegimeRangeBound, nil
}
