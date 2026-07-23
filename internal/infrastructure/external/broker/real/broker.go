// Package real builds a live Binance Futures-backed Broker. This is the
// only adapter capable of moving real money, so it carries its own,
// independent safety gate in addition to the ones already enforced by
// internal/config validation and domain/operation.State.TransitionTo.
package real

import (
	"errors"
	"time"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/infrastructure/external/broker/binance"
)

// NewBroker builds an output.Broker pointed at live Binance Futures. It
// refuses to construct unless both explicit safety flags are set — a third,
// independent guard on top of config validation and the confirmation token
// checked at the moment the system transitions into REAL mode.
func NewBroker(baseURL, apiKey, apiSecret string, recvWindowMs int, timeout time.Duration, realTradingEnabled bool, confirmation string) (output.Broker, error) {
	if !realTradingEnabled || confirmation == "" {
		return nil, errors.New("real broker requires explicit BROKER_REAL_TRADING_ENABLED=true and a non-empty confirmation phrase")
	}
	client := binance.NewClient(baseURL, apiKey, apiSecret, recvWindowMs, timeout)
	return binance.NewBroker(client), nil
}
