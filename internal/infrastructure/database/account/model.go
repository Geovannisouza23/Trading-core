// Package account implements output.AccountRepository against the accounts
// table.
package account

import (
	"time"

	"github.com/shopspring/decimal"
)

type model struct {
	ID               string
	Balance          decimal.Decimal
	Equity           decimal.Decimal
	AvailableBalance decimal.Decimal
	PeakEquity       decimal.Decimal
	DailyPnL         decimal.Decimal
	WeeklyPnL        decimal.Decimal
	CurrentDrawdown  decimal.Decimal
	OperationalMode  string
	Version          int
	UpdatedAt        time.Time
}
