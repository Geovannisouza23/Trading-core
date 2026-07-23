// Package websocket implements a Binance Futures kline WebSocket stream
// client. This is the exchange-facing WebSocket client — not to be
// confused with internal/interfaces/websocket, which serves the dashboard.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

type klineMessage struct {
	K struct {
		Symbol   string `json:"s"`
		Interval string `json:"i"`
		Open     string `json:"o"`
		Close    string `json:"c"`
		High     string `json:"h"`
		Low      string `json:"l"`
		Volume   string `json:"v"`
		IsClosed bool   `json:"x"`
	} `json:"k"`
}

// CandleHandler is invoked once per closed candle received from the stream.
type CandleHandler func(candle market.Candle)

type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL}
}

// Stream connects to a single symbol/timeframe kline stream and invokes
// handler for every closed candle until ctx is cancelled or the connection
// drops. Callers own the reconnect policy (this call returns on any error).
func (c *Client) Stream(ctx context.Context, symbol shared.Symbol, timeframe shared.Timeframe, handler CandleHandler) error {
	streamName := fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol.String()), timeframe.String())
	url := fmt.Sprintf("%s/ws/%s", c.baseURL, streamName)

	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, url, nil)
	if err != nil {
		return fmt.Errorf("marketdata: dialing kline stream: %w", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("marketdata: reading kline stream: %w", err)
		}
		var msg klineMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if !msg.K.IsClosed {
			continue
		}
		candle, err := toCandle(symbol, timeframe, msg, time.Now().UTC())
		if err != nil {
			continue
		}
		handler(candle)
	}
}

func toCandle(symbol shared.Symbol, timeframe shared.Timeframe, msg klineMessage, now time.Time) (market.Candle, error) {
	openDec, err := decimal.NewFromString(msg.K.Open)
	if err != nil {
		return market.Candle{}, err
	}
	highDec, err := decimal.NewFromString(msg.K.High)
	if err != nil {
		return market.Candle{}, err
	}
	lowDec, err := decimal.NewFromString(msg.K.Low)
	if err != nil {
		return market.Candle{}, err
	}
	closeDec, err := decimal.NewFromString(msg.K.Close)
	if err != nil {
		return market.Candle{}, err
	}
	volumeDec, err := decimal.NewFromString(msg.K.Volume)
	if err != nil {
		return market.Candle{}, err
	}

	open, err := shared.NewPrice(openDec)
	if err != nil {
		return market.Candle{}, err
	}
	high, err := shared.NewPrice(highDec)
	if err != nil {
		return market.Candle{}, err
	}
	low, err := shared.NewPrice(lowDec)
	if err != nil {
		return market.Candle{}, err
	}
	closePrice, err := shared.NewPrice(closeDec)
	if err != nil {
		return market.Candle{}, err
	}
	volume, err := shared.NewQuantity(volumeDec)
	if err != nil {
		return market.Candle{}, err
	}

	// The trimmed stream payload does not carry explicit open/close
	// timestamps; callers needing exact candle boundaries should use the
	// REST client instead. Here close time is the message arrival time and
	// open time is approximated one timeframe duration earlier.
	return market.NewClosedCandle(symbol, timeframe, open, high, low, closePrice, volume, now.Add(-time.Minute), now)
}
