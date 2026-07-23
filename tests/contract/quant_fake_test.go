package contract_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
	"trading-core/internal/domain/signal"
	quantfake "trading-core/internal/infrastructure/external/quant/grpc"
)

// TestQuantFakeSignalSatisfiesDomainContract locks in the contract every
// QuantEngine implementation (this fake today, a real gRPC client later)
// must honor: whenever HasSignal is true, every other field must be
// sufficient to build a valid domain signal.TradeSignal — the exact
// conversion application/usecase.EvaluateMarketSignal performs.
func TestQuantFakeSignalSatisfiesDomainContract(t *testing.T) {
	engine := quantfake.NewEngine()
	symbol := shared.MustNewSymbol("BTCUSDT")
	timeframe := shared.MustNewTimeframe("1m")
	now := time.Now()

	previous, err := market.NewClosedCandle(
		symbol, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewPrice(decimal.NewFromInt(101)),
		shared.MustNewPrice(decimal.NewFromInt(99)),
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewQuantity(decimal.NewFromInt(10)),
		now.Add(-2*time.Minute), now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatalf("building previous candle: %v", err)
	}

	// A clear upward move should trigger a BUY signal from the fake's
	// momentum rule.
	current, err := market.NewClosedCandle(
		symbol, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewPrice(decimal.NewFromInt(105)),
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewPrice(decimal.NewFromInt(105)),
		shared.MustNewQuantity(decimal.NewFromInt(10)),
		now.Add(-time.Minute), now,
	)
	if err != nil {
		t.Fatalf("building current candle: %v", err)
	}

	result, err := engine.EvaluateSignal(context.Background(), output.QuantInput{
		Candle:        current,
		RecentCandles: []market.Candle{previous},
	})
	if err != nil {
		t.Fatalf("EvaluateSignal: %v", err)
	}
	if !result.HasSignal {
		t.Fatal("expected a signal for a clear upward momentum move")
	}
	if result.Side != shared.SideBuy {
		t.Fatalf("got side %s, want BUY", result.Side)
	}

	// The contract: this output must build a valid domain TradeSignal with
	// no further adjustment (this is exactly what EvaluateMarketSignal does).
	_, err = signal.New(
		result.Symbol, result.Side, result.EntryPrice, result.StopPrice, result.TargetPrice,
		result.Confidence, result.StrategyName, result.Regime, now, now.Add(30*time.Second),
	)
	if err != nil {
		t.Fatalf("quant fake output failed to build a valid domain signal: %v", err)
	}
}

func TestQuantFakeNoSignalOnFlatMove(t *testing.T) {
	engine := quantfake.NewEngine()
	symbol := shared.MustNewSymbol("BTCUSDT")
	timeframe := shared.MustNewTimeframe("1m")
	now := time.Now()

	flat, err := market.NewClosedCandle(
		symbol, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewQuantity(decimal.NewFromInt(10)),
		now.Add(-time.Minute), now,
	)
	if err != nil {
		t.Fatalf("building candle: %v", err)
	}

	result, err := engine.EvaluateSignal(context.Background(), output.QuantInput{
		Candle:        flat,
		RecentCandles: []market.Candle{flat},
	})
	if err != nil {
		t.Fatalf("EvaluateSignal: %v", err)
	}
	if result.HasSignal {
		t.Fatal("expected no signal when price does not move")
	}
}
