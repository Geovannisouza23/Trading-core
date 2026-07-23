package input

import (
	"context"

	"trading-core/internal/application/command"
)

type ReconciliationReport struct {
	DivergencesFound   int
	Details            []string
	CriticalDivergence bool
}

type ReconcileBrokerStateUseCase interface {
	Execute(ctx context.Context, cmd command.ReconcileBrokerStateCommand) (ReconciliationReport, error)
}
