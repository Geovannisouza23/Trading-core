package output

import (
	"context"

	"trading-core/internal/domain/order"
	"trading-core/internal/domain/shared"
)

// OrderRepository persists the Order aggregate. Save must enforce
// optimistic locking on Order.Version and a unique constraint on
// ClientOrderID.
type OrderRepository interface {
	Save(ctx context.Context, o *order.Order) error
	GetByID(ctx context.Context, id shared.OrderID) (*order.Order, error)
	GetByClientOrderID(ctx context.Context, clientOrderID shared.ClientOrderID) (*order.Order, error)
	ListOpen(ctx context.Context) ([]order.Order, error)
	ListBySignalID(ctx context.Context, signalID shared.SignalID) ([]order.Order, error)
	List(ctx context.Context, limit int) ([]order.Order, error)
}
