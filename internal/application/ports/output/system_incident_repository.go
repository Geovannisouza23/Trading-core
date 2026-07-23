package output

import (
	"context"

	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/shared"
)

// SystemIncidentRepository persists the operation.Incident aggregate.
type SystemIncidentRepository interface {
	Save(ctx context.Context, i *operation.Incident) error
	GetByID(ctx context.Context, id shared.IncidentID) (*operation.Incident, error)
	ListOpen(ctx context.Context) ([]operation.Incident, error)
	List(ctx context.Context, limit int) ([]operation.Incident, error)
}
