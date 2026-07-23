package output

import (
	"context"

	"trading-core/internal/domain/shared"
	"trading-core/internal/domain/signal"
)

// TradeSignalRepository persists the TradeSignal aggregate.
type TradeSignalRepository interface {
	Save(ctx context.Context, s *signal.TradeSignal) error
	GetByID(ctx context.Context, id shared.SignalID) (*signal.TradeSignal, error)
	List(ctx context.Context, limit int) ([]signal.TradeSignal, error)
}
