// Package market models raw market data primitives (candles, regimes) that
// feed the signal evaluation pipeline.
package market

import (
	"time"

	"trading-core/internal/domain/shared"
)

// Candle is a closed OHLCV bar for a symbol/timeframe pair.
type Candle struct {
	Symbol    shared.Symbol
	Timeframe shared.Timeframe
	Open      shared.Price
	High      shared.Price
	Low       shared.Price
	Close     shared.Price
	Volume    shared.Quantity
	OpenTime  time.Time
	CloseTime time.Time
}

// NewClosedCandle validates OHLC consistency and builds a Candle. Only
// closed candles are accepted by the domain: partial/streaming candles are
// an infrastructure concern filtered out before reaching this constructor.
func NewClosedCandle(
	symbol shared.Symbol,
	timeframe shared.Timeframe,
	open, high, low, close shared.Price,
	volume shared.Quantity,
	openTime, closeTime time.Time,
) (Candle, error) {
	if !closeTime.After(openTime) {
		return Candle{}, shared.NewValidationError("close_time", "must be after open_time")
	}
	if high.LessThan(open) || high.LessThan(close) || high.LessThan(low) {
		return Candle{}, shared.NewValidationError("high", "must be greater than or equal to open, close and low")
	}
	if low.GreaterThan(open) || low.GreaterThan(close) {
		return Candle{}, shared.NewValidationError("low", "must be less than or equal to open and close")
	}
	return Candle{
		Symbol:    symbol,
		Timeframe: timeframe,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		OpenTime:  openTime,
		CloseTime: closeTime,
	}, nil
}

// Age returns how old the candle is relative to `now`, measured from its
// close time.
func (c Candle) Age(now time.Time) time.Duration {
	return now.Sub(c.CloseTime)
}

// AgeExceeds reports whether the candle is older than maxAge relative to
// `now`. Used to enforce MarketDataFreshSpecification.
func (c Candle) AgeExceeds(now time.Time, maxAge time.Duration) bool {
	return c.Age(now) > maxAge
}

// CandleClosed is published whenever a new closed candle is ingested.
type CandleClosed struct {
	Candle      Candle
	OccurredAt_ time.Time
}

func (e CandleClosed) EventName() string     { return "CandleClosed" }
func (e CandleClosed) OccurredAt() time.Time { return e.OccurredAt_ }
