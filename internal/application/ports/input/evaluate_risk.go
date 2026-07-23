package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/dto"
)

type EvaluateRiskResult struct {
	Decision dto.RiskDecisionDTO
}

type EvaluateRiskUseCase interface {
	Execute(ctx context.Context, cmd command.EvaluateRiskCommand) (EvaluateRiskResult, error)
}
