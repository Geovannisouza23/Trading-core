package output

import (
	"context"

	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

// QuantInput is the market data the Quant Engine evaluates a candidate
// signal from.
type QuantInput struct {
	Candle        market.Candle
	RecentCandles []market.Candle
}

// QuantSignal is the Quant Engine's raw evaluation output, before it is
// turned into a domain signal.TradeSignal by EvaluateMarketSignal.
type QuantSignal struct {
	HasSignal    bool
	Symbol       shared.Symbol
	Side         shared.Side
	EntryPrice   shared.Price
	StopPrice    shared.Price
	TargetPrice  shared.Price
	Confidence   shared.Confidence
	StrategyName string
	Regime       market.Regime
}

// MarketRegimeInput is the market data used to classify the current regime.
type MarketRegimeInput struct {
	Symbol        shared.Symbol
	RecentCandles []market.Candle
}

// QuantEngine is the port to the (currently external/Rust, future) signal
// evaluation service. It never places orders; it only proposes them.
type QuantEngine interface {
	EvaluateSignal(ctx context.Context, input QuantInput) (QuantSignal, error)
	AnalyzeMarketRegime(ctx context.Context, input MarketRegimeInput) (market.Regime, error)
}
