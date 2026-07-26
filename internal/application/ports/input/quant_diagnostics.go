package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
)

// QuantDiagnosticsService groups quant-engine's remaining read/diagnostic
// and one-off capabilities behind one port: strategy validation, direct
// feature/model access, the decision-outcome feedback loop, and
// model-registry introspection/reload.
type QuantDiagnosticsService interface {
	ValidateStrategy(ctx context.Context, cmd command.ValidateStrategyCommand) (output.ValidateStrategyResult, error)
	CalculateFeatures(ctx context.Context, cmd command.CalculateFeaturesCommand) (output.CalculateFeaturesResult, error)
	EvaluateModel(ctx context.Context, cmd command.EvaluateModelCommand) (output.ModelPrediction, error)
	RegisterDecisionOutcome(ctx context.Context, cmd command.RegisterDecisionOutcomeCommand) (output.RegisterDecisionOutcomeResult, error)
	GetFeatureSchema(ctx context.Context, q query.GetFeatureSchemaQuery) (output.GetFeatureSchemaResult, error)
	GetModelMetadata(ctx context.Context, q query.GetModelMetadataQuery) (output.GetModelMetadataResult, error)
	ReloadApprovedModel(ctx context.Context, cmd command.ReloadApprovedModelCommand) (output.ReloadApprovedModelResult, error)
}
