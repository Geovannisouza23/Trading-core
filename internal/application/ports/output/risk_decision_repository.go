package output

import (
	"context"

	"trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"
)

// RiskDecisionRepository persists the risk.Decision aggregate.
type RiskDecisionRepository interface {
	Save(ctx context.Context, d *risk.Decision) error
	GetByID(ctx context.Context, id shared.RiskDecisionID) (*risk.Decision, error)
	List(ctx context.Context, limit int) ([]risk.Decision, error)
}
