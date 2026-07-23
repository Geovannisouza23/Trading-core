// Package signal implements output.TradeSignalRepository against the
// trade_signals table.
package signal

import (
	"time"

	"github.com/shopspring/decimal"
)

type model struct {
	ID           string
	Symbol       string
	Side         string
	EntryPrice   decimal.Decimal
	StopPrice    decimal.Decimal
	TargetPrice  decimal.Decimal
	Confidence   decimal.Decimal
	StrategyName string
	MarketRegime string
	CreatedAt    time.Time
	ValidUntil   time.Time
}
