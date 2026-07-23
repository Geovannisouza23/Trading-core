package output

import (
	"context"

	"trading-core/internal/domain/event"
	"trading-core/internal/domain/shared"
)

// MarketEventRepository persists the MarketEvent aggregate.
type MarketEventRepository interface {
	Save(ctx context.Context, e *event.MarketEvent) error
	GetByID(ctx context.Context, id shared.MarketEventID) (*event.MarketEvent, error)
	ListActive(ctx context.Context) ([]event.MarketEvent, error)
	List(ctx context.Context, limit int) ([]event.MarketEvent, error)
}
