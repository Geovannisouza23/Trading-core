package account

import (
	"time"

	"trading-core/internal/domain/order"
	"trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
)

// Snapshot is a point-in-time read model of the account, its open positions
// and open orders. It is persisted periodically so the dashboard and
// incident investigations have a consistent historical view.
type Snapshot struct {
	ID         shared.SnapshotID
	Balance    shared.Money
	Equity     shared.Money
	Positions  []position.Position
	OpenOrders []order.Order
	DailyPnL   shared.Money
	Drawdown   shared.Drawdown
	Timestamp  time.Time
}

// NewSnapshot captures the current account state.
func NewSnapshot(acc *Account, positions []position.Position, openOrders []order.Order, timestamp time.Time) *Snapshot {
	return &Snapshot{
		ID:         shared.NewSnapshotID(),
		Balance:    acc.Balance,
		Equity:     acc.Equity,
		Positions:  positions,
		OpenOrders: openOrders,
		DailyPnL:   acc.DailyPnL,
		Drawdown:   acc.CurrentDrawdown,
		Timestamp:  timestamp,
	}
}
