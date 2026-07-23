package input

import (
	"context"

	"trading-core/internal/application/command"
)

type ChangeOperationalModeUseCase interface {
	Execute(ctx context.Context, cmd command.ChangeOperationalModeCommand) error
}
