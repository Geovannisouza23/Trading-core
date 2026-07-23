// Package position implements output.PositionRepository against the
// positions table.
package position

import (
	"time"

	"github.com/shopspring/decimal"
)

type model struct {
	ID            string
	Symbol        string
	Side          string
	Quantity      decimal.Decimal
	EntryPrice    decimal.Decimal
	CurrentPrice  decimal.Decimal
	StopPrice     *decimal.Decimal
	TargetPrice   *decimal.Decimal
	UnrealizedPnL decimal.Decimal
	RealizedPnL   decimal.Decimal
	Status        string
	Version       int
	OpenedAt      time.Time
	ClosedAt      *time.Time
}
