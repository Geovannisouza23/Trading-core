package grpc

import (
	"fmt"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
	"trading-core/internal/domain/market"
)

// candleToProto maps a domain Candle onto the wire Candle message. Every
// decimal field is sent as a string — never a float — per the contract's
// own decimal-as-string convention.
func candleToProto(c market.Candle) *quantv1.Candle {
	return &quantv1.Candle{
		Symbol:    c.Symbol.String(),
		Timeframe: c.Timeframe.String(),
		OpenTime:  timestamppb.New(c.OpenTime),
		CloseTime: timestamppb.New(c.CloseTime),
		Open:      c.Open.Decimal().String(),
		High:      c.High.Decimal().String(),
		Low:       c.Low.Decimal().String(),
		Close:     c.Close.Decimal().String(),
		Volume:    c.Volume.Decimal().String(),
		Closed:    true,
	}
}

func candlesToProto(candles []market.Candle) []*quantv1.Candle {
	out := make([]*quantv1.Candle, len(candles))
	for i, c := range candles {
		out[i] = candleToProto(c)
	}
	return out
}

// decimalFromWire parses a decimal-as-string wire field, returning a clear
// error instead of panicking when the server sends something malformed.
func decimalFromWire(field, value string) (decimal.Decimal, error) {
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("quant engine returned an invalid %s %q: %w", field, value, err)
	}
	return parsed, nil
}

// regimeFromProto collapses the 10-state wire MarketRegime onto the 4-state
// domain market.Regime. The aggregation groups by trading posture, not by
// a 1:1 name match, since the domain enum is intentionally coarser:
//   - TRENDING_UP / TRENDING_DOWN / BREAKOUT -> Trending (all three are
//     "a directional move is underway or just started")
//   - RANGING -> RangeBound
//   - HIGH_VOLATILITY / LOW_VOLATILITY / PANIC -> Volatile (elevated or
//     collapsing volatility both call for the same defensive posture)
//   - ILLIQUID / NEWS_EVENT / UNSPECIFIED / UNKNOWN -> Unknown (not enough
//     of a clean signal to commit to one of the other three)
func regimeFromProto(regime quantv1.MarketRegime) market.Regime {
	switch regime {
	case quantv1.MarketRegime_MARKET_REGIME_TRENDING_UP,
		quantv1.MarketRegime_MARKET_REGIME_TRENDING_DOWN,
		quantv1.MarketRegime_MARKET_REGIME_BREAKOUT:
		return market.RegimeTrending
	case quantv1.MarketRegime_MARKET_REGIME_RANGING:
		return market.RegimeRangeBound
	case quantv1.MarketRegime_MARKET_REGIME_HIGH_VOLATILITY,
		quantv1.MarketRegime_MARKET_REGIME_LOW_VOLATILITY,
		quantv1.MarketRegime_MARKET_REGIME_PANIC:
		return market.RegimeVolatile
	default:
		return market.RegimeUnknown
	}
}
