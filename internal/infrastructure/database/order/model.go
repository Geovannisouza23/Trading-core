// Package order implements output.OrderRepository against the orders table.
package order

import (
	"time"

	"github.com/shopspring/decimal"
)

type model struct {
	ID                    string
	ClientOrderID         string
	BrokerOrderID         string
	Symbol                string
	Side                  string
	Type                  string
	Quantity              decimal.Decimal
	FilledQuantity        decimal.Decimal
	RequestedPrice        *decimal.Decimal
	AverageExecutionPrice *decimal.Decimal
	StopPrice             *decimal.Decimal
	TargetPrice           *decimal.Decimal
	Status                string
	StrategyName          string
	SignalID              string
	RiskDecisionID        string
	FailureReason         string
	Version               int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
