// Package snapshot implements output.AccountSnapshotRepository against the
// account_snapshots table.
package snapshot

import (
	"time"

	"github.com/shopspring/decimal"
)

type model struct {
	ID         string
	Balance    decimal.Decimal
	Equity     decimal.Decimal
	Positions  []byte
	OpenOrders []byte
	DailyPnL   decimal.Decimal
	Drawdown   decimal.Decimal
	Timestamp  time.Time
}
