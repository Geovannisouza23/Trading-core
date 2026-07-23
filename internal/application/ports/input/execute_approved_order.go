package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/dto"
)

type ExecuteApprovedOrderResult struct {
	Order      *dto.OrderDTO
	Skipped    bool
	SkipReason string
}

type ExecuteApprovedOrderUseCase interface {
	Execute(ctx context.Context, cmd command.ExecuteApprovedOrderCommand) (ExecuteApprovedOrderResult, error)
}
