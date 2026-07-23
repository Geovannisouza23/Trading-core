package output

import (
	"context"

	"trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
)

// PositionRepository persists the Position aggregate.
type PositionRepository interface {
	Save(ctx context.Context, p *position.Position) error
	GetByID(ctx context.Context, id shared.PositionID) (*position.Position, error)
	ListOpen(ctx context.Context) ([]position.Position, error)
	GetOpenBySymbol(ctx context.Context, symbol shared.Symbol) (*position.Position, error)
}
