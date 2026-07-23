// Package rest implements a Binance Futures klines (candlestick) REST
// client, used for historical backfill and for polling when a WebSocket
// stream isn't available.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string) *Client {
	return &Client{httpClient: &http.Client{Timeout: 10 * time.Second}, baseURL: baseURL}
}

// FetchKlines fetches up to `limit` recent closed candles for symbol/timeframe.
func (c *Client) FetchKlines(ctx context.Context, symbol shared.Symbol, timeframe shared.Timeframe, limit int) ([]market.Candle, error) {
	url := fmt.Sprintf("%s/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", c.baseURL, symbol.String(), timeframe.String(), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("marketdata: http %d fetching klines for %s", resp.StatusCode, symbol)
	}

	var raw [][]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("marketdata: decoding klines: %w", err)
	}

	candles := make([]market.Candle, 0, len(raw))
	for _, k := range raw {
		candle, err := parseKline(symbol, timeframe, k)
		if err != nil {
			continue
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

// parseKline decodes one element of Binance's kline array response:
// [openTime, open, high, low, close, volume, closeTime, ...].
func parseKline(symbol shared.Symbol, timeframe shared.Timeframe, k []any) (market.Candle, error) {
	if len(k) < 7 {
		return market.Candle{}, fmt.Errorf("marketdata: malformed kline row")
	}
	open, err := decimalFromAny(k[1])
	if err != nil {
		return market.Candle{}, err
	}
	high, err := decimalFromAny(k[2])
	if err != nil {
		return market.Candle{}, err
	}
	low, err := decimalFromAny(k[3])
	if err != nil {
		return market.Candle{}, err
	}
	closePrice, err := decimalFromAny(k[4])
	if err != nil {
		return market.Candle{}, err
	}
	volume, err := decimalFromAny(k[5])
	if err != nil {
		return market.Candle{}, err
	}

	openTime := millisFromAny(k[0])
	closeTime := millisFromAny(k[6])

	openPrice, err := shared.NewPrice(open)
	if err != nil {
		return market.Candle{}, err
	}
	highPrice, err := shared.NewPrice(high)
	if err != nil {
		return market.Candle{}, err
	}
	lowPrice, err := shared.NewPrice(low)
	if err != nil {
		return market.Candle{}, err
	}
	closePricePrice, err := shared.NewPrice(closePrice)
	if err != nil {
		return market.Candle{}, err
	}
	volumeQty, err := shared.NewQuantity(volume)
	if err != nil {
		return market.Candle{}, err
	}

	return market.NewClosedCandle(symbol, timeframe, openPrice, highPrice, lowPrice, closePricePrice, volumeQty, openTime, closeTime)
}

func decimalFromAny(v any) (decimal.Decimal, error) {
	s, ok := v.(string)
	if !ok {
		return decimal.Decimal{}, fmt.Errorf("marketdata: expected string, got %T", v)
	}
	return decimal.NewFromString(s)
}

func millisFromAny(v any) time.Time {
	f, ok := v.(float64)
	if !ok {
		return time.Time{}
	}
	return time.UnixMilli(int64(f)).UTC()
}
