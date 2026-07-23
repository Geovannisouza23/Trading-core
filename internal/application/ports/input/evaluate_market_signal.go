// Package input defines the inbound ports (one per use case) that the
// interfaces layer is allowed to call. Concrete use case implementations
// live in internal/application/usecase.
package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/dto"
)

type EvaluateMarketSignalResult struct {
	SignalCreated bool
	Signal        *dto.SignalDTO
}

type EvaluateMarketSignalUseCase interface {
	Execute(ctx context.Context, cmd command.EvaluateMarketSignalCommand) (EvaluateMarketSignalResult, error)
}
