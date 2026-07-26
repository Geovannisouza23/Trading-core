package grpc

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

func TestRegimeFromProtoCoversAllTenWireValues(t *testing.T) {
	cases := map[quantv1.MarketRegime]market.Regime{
		quantv1.MarketRegime_MARKET_REGIME_TRENDING_UP:     market.RegimeTrending,
		quantv1.MarketRegime_MARKET_REGIME_TRENDING_DOWN:   market.RegimeTrending,
		quantv1.MarketRegime_MARKET_REGIME_BREAKOUT:        market.RegimeTrending,
		quantv1.MarketRegime_MARKET_REGIME_RANGING:         market.RegimeRangeBound,
		quantv1.MarketRegime_MARKET_REGIME_HIGH_VOLATILITY: market.RegimeVolatile,
		quantv1.MarketRegime_MARKET_REGIME_LOW_VOLATILITY:  market.RegimeVolatile,
		quantv1.MarketRegime_MARKET_REGIME_PANIC:           market.RegimeVolatile,
		quantv1.MarketRegime_MARKET_REGIME_ILLIQUID:        market.RegimeUnknown,
		quantv1.MarketRegime_MARKET_REGIME_NEWS_EVENT:      market.RegimeUnknown,
		quantv1.MarketRegime_MARKET_REGIME_UNSPECIFIED:     market.RegimeUnknown,
		quantv1.MarketRegime_MARKET_REGIME_UNKNOWN:         market.RegimeUnknown,
	}

	for wire, want := range cases {
		got := regimeFromProto(wire)
		if got != want {
			t.Errorf("regimeFromProto(%s) = %s, want %s", wire, got, want)
		}
	}
}

func TestDecimalFromWireParsesAValidDecimalString(t *testing.T) {
	value, err := decimalFromWire("entry_price", "50123.456789")
	if err != nil {
		t.Fatalf("decimalFromWire: %v", err)
	}
	if !value.Equal(decimal.RequireFromString("50123.456789")) {
		t.Fatalf("got %s, want 50123.456789", value)
	}
}

func TestDecimalFromWireReturnsAClearErrorOnMalformedInput(t *testing.T) {
	_, err := decimalFromWire("entry_price", "not-a-number")
	if err == nil {
		t.Fatal("expected an error for a malformed decimal string, got nil")
	}
}

func TestCandleToProtoRoundTripsEveryField(t *testing.T) {
	symbol := shared.MustNewSymbol("BTCUSDT")
	timeframe := shared.MustNewTimeframe("1m")
	openTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	closeTime := openTime.Add(time.Minute)

	candle, err := market.NewClosedCandle(
		symbol, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewPrice(decimal.NewFromInt(105)),
		shared.MustNewPrice(decimal.NewFromInt(99)),
		shared.MustNewPrice(decimal.NewFromInt(103)),
		shared.MustNewQuantity(decimal.NewFromInt(10)),
		openTime, closeTime,
	)
	if err != nil {
		t.Fatalf("building candle: %v", err)
	}

	proto := candleToProto(candle)

	if proto.GetSymbol() != "BTCUSDT" {
		t.Errorf("symbol = %q, want BTCUSDT", proto.GetSymbol())
	}
	if proto.GetTimeframe() != "1m" {
		t.Errorf("timeframe = %q, want 1m", proto.GetTimeframe())
	}
	if proto.GetOpen() != "100" || proto.GetHigh() != "105" || proto.GetLow() != "99" || proto.GetClose() != "103" {
		t.Errorf("OHLC = %s/%s/%s/%s, want 100/105/99/103", proto.GetOpen(), proto.GetHigh(), proto.GetLow(), proto.GetClose())
	}
	if proto.GetVolume() != "10" {
		t.Errorf("volume = %q, want 10", proto.GetVolume())
	}
	if !proto.GetClosed() {
		t.Error("expected Closed to always be true — only closed candles reach this mapper")
	}
	if !proto.GetOpenTime().AsTime().Equal(openTime) {
		t.Errorf("open_time = %s, want %s", proto.GetOpenTime().AsTime(), openTime)
	}
	if !proto.GetCloseTime().AsTime().Equal(closeTime) {
		t.Errorf("close_time = %s, want %s", proto.GetCloseTime().AsTime(), closeTime)
	}
}

func TestQuantSignalFromProtoMapsHoldAndUnspecifiedToNoSignal(t *testing.T) {
	symbol := shared.MustNewSymbol("BTCUSDT")
	timeframe := shared.MustNewTimeframe("1m")
	now := time.Now()
	candle, err := market.NewClosedCandle(
		symbol, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(100)), shared.MustNewPrice(decimal.NewFromInt(101)),
		shared.MustNewPrice(decimal.NewFromInt(99)), shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewQuantity(decimal.NewFromInt(1)), now.Add(-time.Minute), now,
	)
	if err != nil {
		t.Fatalf("building candle: %v", err)
	}

	for _, action := range []quantv1.Action{quantv1.Action_ACTION_HOLD, quantv1.Action_ACTION_UNSPECIFIED} {
		resp := &quantv1.EvaluateSignalResponse{Action: action}
		signal, err := quantSignalFromProto(candle, resp)
		if err != nil {
			t.Fatalf("quantSignalFromProto(%s): %v", action, err)
		}
		if signal.HasSignal {
			t.Errorf("action %s: expected HasSignal=false", action)
		}
	}
}

func TestQuantSignalFromProtoMapsBuyWithValidPrices(t *testing.T) {
	symbol := shared.MustNewSymbol("BTCUSDT")
	timeframe := shared.MustNewTimeframe("1m")
	now := time.Now()
	candle, err := market.NewClosedCandle(
		symbol, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(100)), shared.MustNewPrice(decimal.NewFromInt(101)),
		shared.MustNewPrice(decimal.NewFromInt(99)), shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewQuantity(decimal.NewFromInt(1)), now.Add(-time.Minute), now,
	)
	if err != nil {
		t.Fatalf("building candle: %v", err)
	}

	resp := &quantv1.EvaluateSignalResponse{
		Action:             quantv1.Action_ACTION_BUY,
		EntryPrice:         "100",
		StopPrice:          "98",
		TargetPrice:        "104",
		AdjustedConfidence: "0.75",
		Strategy:           "momentum",
		MarketRegime:       quantv1.MarketRegime_MARKET_REGIME_TRENDING_UP,
	}

	signal, err := quantSignalFromProto(candle, resp)
	if err != nil {
		t.Fatalf("quantSignalFromProto: %v", err)
	}
	if !signal.HasSignal {
		t.Fatal("expected HasSignal=true for ACTION_BUY")
	}
	if signal.Side != shared.SideBuy {
		t.Errorf("side = %s, want BUY", signal.Side)
	}
	if signal.Regime != market.RegimeTrending {
		t.Errorf("regime = %s, want TRENDING", signal.Regime)
	}
	if signal.StrategyName != "momentum" {
		t.Errorf("strategy = %q, want momentum", signal.StrategyName)
	}
}

func TestQuantSignalFromProtoRejectsAMalformedPriceInsteadOfPanicking(t *testing.T) {
	symbol := shared.MustNewSymbol("BTCUSDT")
	timeframe := shared.MustNewTimeframe("1m")
	now := time.Now()
	candle, err := market.NewClosedCandle(
		symbol, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(100)), shared.MustNewPrice(decimal.NewFromInt(101)),
		shared.MustNewPrice(decimal.NewFromInt(99)), shared.MustNewPrice(decimal.NewFromInt(100)),
		shared.MustNewQuantity(decimal.NewFromInt(1)), now.Add(-time.Minute), now,
	)
	if err != nil {
		t.Fatalf("building candle: %v", err)
	}

	resp := &quantv1.EvaluateSignalResponse{
		Action:     quantv1.Action_ACTION_BUY,
		EntryPrice: "not-a-decimal",
	}

	_, err = quantSignalFromProto(candle, resp)
	if err == nil {
		t.Fatal("expected an error for a malformed entry_price, got nil")
	}
}
