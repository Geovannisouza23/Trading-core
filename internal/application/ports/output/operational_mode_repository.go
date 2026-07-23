package output

import (
	"context"

	"trading-core/internal/domain/operation"
)

// OperationalModeRepository persists the single, auditable
// operation.State row.
type OperationalModeRepository interface {
	Get(ctx context.Context) (*operation.State, error)
	Save(ctx context.Context, s *operation.State) error
}
