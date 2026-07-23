package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/dto"
)

type ProcessMarketEventResult struct {
	Event        *dto.MarketEventDTO
	Deduplicated bool
}

type ProcessMarketEventUseCase interface {
	Execute(ctx context.Context, cmd command.ProcessMarketEventCommand) (ProcessMarketEventResult, error)
}
