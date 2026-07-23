// Package testnet builds a Binance Futures Testnet-backed Broker, using
// credentials kept separate from the real adapter (spec section 14).
package testnet

import (
	"time"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/infrastructure/external/broker/binance"
)

// NewBroker builds an output.Broker pointed at Binance Futures Testnet.
func NewBroker(baseURL, apiKey, apiSecret string, recvWindowMs int, timeout time.Duration) output.Broker {
	client := binance.NewClient(baseURL, apiKey, apiSecret, recvWindowMs, timeout)
	return binance.NewBroker(client)
}
