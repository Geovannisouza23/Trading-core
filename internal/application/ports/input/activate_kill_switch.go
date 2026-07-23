package input

import (
	"context"

	"trading-core/internal/application/command"
)

type ActivateKillSwitchUseCase interface {
	Execute(ctx context.Context, cmd command.ActivateKillSwitchCommand) error
}
