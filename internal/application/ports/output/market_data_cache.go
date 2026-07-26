package output

import (
	"context"

	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

// MarketDataCache caches the most recent closed candles per
// symbol/timeframe — an optional read-through layer in front of whatever
// fetches candle history (the Binance REST client, or the live feed
// writing through as candles close). A cache miss is not an error: the
// bool return distinguishes "not cached yet" from a real failure, same
// shape as a Go map's comma-ok idiom.
type MarketDataCache interface {
	GetCandles(ctx context.Context, symbol shared.Symbol, timeframe shared.Timeframe) ([]market.Candle, bool, error)
	SetCandles(ctx context.Context, symbol shared.Symbol, timeframe shared.Timeframe, candles []market.Candle) error
}
