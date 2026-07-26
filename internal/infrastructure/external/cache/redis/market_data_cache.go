package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

// marketDataCacheTTL is deliberately short: this cache exists to absorb
// bursts of repeated reads for the same symbol/timeframe window (e.g.
// several backtest/optimization requests submitted close together), not
// to serve stale history — quant-engine's contract in ADR
// 0010-real-grpc-client-for-quant-engine.md is that trading-core never
// invents candle data on the caller's behalf, only caches what it was
// already given or fetched.
const marketDataCacheTTL = 5 * time.Minute

// cachedCandle mirrors market.Candle as a JSON-safe DTO — decimal
// fields stay strings (the project-wide convention), never floats.
type cachedCandle struct {
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Open      string    `json:"open"`
	High      string    `json:"high"`
	Low       string    `json:"low"`
	Close     string    `json:"close"`
	Volume    string    `json:"volume"`
}

// MarketDataCache implements output.MarketDataCache.
type MarketDataCache struct {
	client *Client
}

func NewMarketDataCache(client *Client) *MarketDataCache {
	return &MarketDataCache{client: client}
}

var _ output.MarketDataCache = (*MarketDataCache)(nil)

func candlesCacheKey(symbol shared.Symbol, timeframe shared.Timeframe) string {
	return fmt.Sprintf("market-data:candles:%s:%s", symbol.String(), timeframe.String())
}

func (c *MarketDataCache) GetCandles(ctx context.Context, symbol shared.Symbol, timeframe shared.Timeframe) ([]market.Candle, bool, error) {
	raw, err := c.client.Get(ctx, candlesCacheKey(symbol, timeframe))
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading candle cache: %w", err)
	}

	var cached []cachedCandle
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, false, fmt.Errorf("decoding cached candles: %w", err)
	}

	candles := make([]market.Candle, 0, len(cached))
	for i, cc := range cached {
		candle, err := candleFromCache(symbol, timeframe, cc)
		if err != nil {
			return nil, false, fmt.Errorf("cached candle %d: %w", i, err)
		}
		candles = append(candles, candle)
	}
	return candles, true, nil
}

func (c *MarketDataCache) SetCandles(ctx context.Context, symbol shared.Symbol, timeframe shared.Timeframe, candles []market.Candle) error {
	cached := make([]cachedCandle, len(candles))
	for i, candle := range candles {
		cached[i] = cachedCandle{
			OpenTime:  candle.OpenTime,
			CloseTime: candle.CloseTime,
			Open:      candle.Open.Decimal().String(),
			High:      candle.High.Decimal().String(),
			Low:       candle.Low.Decimal().String(),
			Close:     candle.Close.Decimal().String(),
			Volume:    candle.Volume.Decimal().String(),
		}
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("encoding candles for cache: %w", err)
	}
	if err := c.client.Set(ctx, candlesCacheKey(symbol, timeframe), string(data), marketDataCacheTTL); err != nil {
		return fmt.Errorf("writing candle cache: %w", err)
	}
	return nil
}

func candleFromCache(symbol shared.Symbol, timeframe shared.Timeframe, cc cachedCandle) (market.Candle, error) {
	open, err := priceFromDecimalString(cc.Open)
	if err != nil {
		return market.Candle{}, fmt.Errorf("open: %w", err)
	}
	high, err := priceFromDecimalString(cc.High)
	if err != nil {
		return market.Candle{}, fmt.Errorf("high: %w", err)
	}
	low, err := priceFromDecimalString(cc.Low)
	if err != nil {
		return market.Candle{}, fmt.Errorf("low: %w", err)
	}
	closePrice, err := priceFromDecimalString(cc.Close)
	if err != nil {
		return market.Candle{}, fmt.Errorf("close: %w", err)
	}
	volumeValue, err := decimal.NewFromString(cc.Volume)
	if err != nil {
		return market.Candle{}, fmt.Errorf("volume: %w", err)
	}
	volume, err := shared.NewQuantity(volumeValue)
	if err != nil {
		return market.Candle{}, fmt.Errorf("volume: %w", err)
	}
	return market.NewClosedCandle(symbol, timeframe, open, high, low, closePrice, volume, cc.OpenTime, cc.CloseTime)
}

func priceFromDecimalString(raw string) (shared.Price, error) {
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return shared.Price{}, err
	}
	return shared.NewPrice(value)
}
