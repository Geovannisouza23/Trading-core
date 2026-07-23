// Package output defines every outbound port the application layer depends
// on. Concrete implementations live under internal/infrastructure and are
// wired together exclusively in internal/app.
package output

import (
	"context"

	"trading-core/internal/domain/shared"
)

// BrokerAccount is the broker's view of account funds, translated into
// internal value objects by the adapter before it ever reaches a use case.
type BrokerAccount struct {
	Balance          shared.Money
	Equity           shared.Money
	AvailableBalance shared.Money
}

// BrokerPosition is the broker's view of a single open position.
type BrokerPosition struct {
	Symbol     shared.Symbol
	Side       shared.Side
	Quantity   shared.Quantity
	EntryPrice shared.Price
	MarkPrice  shared.Price
}

// BrokerOrder is the broker's view of an order at any point in its
// lifecycle. RawStatus preserves the broker-native status string for
// auditability; Status is the same value normalized to lower case for
// adapter-internal comparisons.
type BrokerOrder struct {
	BrokerOrderID  string
	ClientOrderID  string
	Symbol         shared.Symbol
	Side           shared.Side
	Type           string
	Quantity       shared.Quantity
	FilledQuantity shared.Quantity
	AveragePrice   *shared.Price
	RawStatus      string
}

// PlaceOrderRequest is the input to Broker.PlaceOrder.
type PlaceOrderRequest struct {
	ClientOrderID shared.ClientOrderID
	Symbol        shared.Symbol
	Side          shared.Side
	Type          string
	Quantity      shared.Quantity
	Price         *shared.Price
}

// PlaceStopRequest is the input to Broker.PlaceStop.
type PlaceStopRequest struct {
	ClientOrderID shared.ClientOrderID
	Symbol        shared.Symbol
	Side          shared.Side
	Quantity      shared.Quantity
	StopPrice     shared.Price
}

// Broker is the only port allowed to reach a real exchange. Implementations
// (paper, testnet, real) must never be called directly by anything outside
// the ExecuteApprovedOrder use case and reconciliation.
type Broker interface {
	GetAccount(ctx context.Context) (BrokerAccount, error)
	GetPositions(ctx context.Context) ([]BrokerPosition, error)
	GetOpenOrders(ctx context.Context) ([]BrokerOrder, error)
	PlaceOrder(ctx context.Context, req PlaceOrderRequest) (BrokerOrder, error)
	PlaceStop(ctx context.Context, req PlaceStopRequest) (BrokerOrder, error)
	CancelOrder(ctx context.Context, brokerOrderID string) error
	GetOrder(ctx context.Context, brokerOrderID string) (BrokerOrder, error)
	FindOrderByClientOrderID(ctx context.Context, clientOrderID string) (BrokerOrder, error)
}
