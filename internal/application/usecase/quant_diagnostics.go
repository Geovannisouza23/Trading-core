package usecase

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
	"trading-core/internal/domain/shared"
)

// QuantDiagnosticsService implements input.QuantDiagnosticsService by
// delegating straight to output.QuantDiagnostics.
type QuantDiagnosticsService struct {
	diagnostics output.QuantDiagnostics
}

func NewQuantDiagnosticsService(diagnostics output.QuantDiagnostics) *QuantDiagnosticsService {
	return &QuantDiagnosticsService{diagnostics: diagnostics}
}

var _ input.QuantDiagnosticsService = (*QuantDiagnosticsService)(nil)

func (s *QuantDiagnosticsService) ValidateStrategy(ctx context.Context, cmd command.ValidateStrategyCommand) (output.ValidateStrategyResult, error) {
	if cmd.StrategyName == "" {
		return output.ValidateStrategyResult{}, shared.NewValidationError("strategy_name", "must not be empty")
	}
	return s.diagnostics.ValidateStrategy(ctx, output.ValidateStrategyInput{
		StrategyName:    cmd.StrategyName,
		StrategyVersion: cmd.StrategyVersion,
		Params:          cmd.Params,
	})
}

func (s *QuantDiagnosticsService) CalculateFeatures(ctx context.Context, cmd command.CalculateFeaturesCommand) (output.CalculateFeaturesResult, error) {
	if cmd.Symbol == "" {
		return output.CalculateFeaturesResult{}, shared.NewValidationError("symbol", "must not be empty")
	}
	return s.diagnostics.CalculateFeatures(ctx, output.CalculateFeaturesInput{
		Symbol:               cmd.Symbol,
		Timeframe:            cmd.Timeframe,
		Candles:              cmd.Candles,
		FeatureSchemaVersion: cmd.FeatureSchemaVersion,
	})
}

func (s *QuantDiagnosticsService) EvaluateModel(ctx context.Context, cmd command.EvaluateModelCommand) (output.ModelPrediction, error) {
	if cmd.ModelName == "" {
		return output.ModelPrediction{}, shared.NewValidationError("model_name", "must not be empty")
	}
	return s.diagnostics.EvaluateModel(ctx, output.EvaluateModelInput{
		ModelName:            cmd.ModelName,
		ModelVersion:         cmd.ModelVersion,
		Features:             cmd.Features,
		FeatureSchemaVersion: cmd.FeatureSchemaVersion,
	})
}

func (s *QuantDiagnosticsService) RegisterDecisionOutcome(ctx context.Context, cmd command.RegisterDecisionOutcomeCommand) (output.RegisterDecisionOutcomeResult, error) {
	if cmd.DecisionID == "" {
		return output.RegisterDecisionOutcomeResult{}, shared.NewValidationError("decision_id", "must not be empty")
	}
	return s.diagnostics.RegisterDecisionOutcome(ctx, output.RegisterDecisionOutcomeInput{
		DecisionID:      cmd.DecisionID,
		OrderCreated:    cmd.OrderCreated,
		Executed:        cmd.Executed,
		RejectionReason: cmd.RejectionReason,
		EntryPrice:      cmd.EntryPrice,
		ExitPrice:       cmd.ExitPrice,
		Quantity:        cmd.Quantity,
		Fees:            cmd.Fees,
		Slippage:        cmd.Slippage,
		Pnl:             cmd.Pnl,
		ExitReason:      cmd.ExitReason,
		EntryAt:         cmd.EntryAt,
		ExitAt:          cmd.ExitAt,
	})
}

func (s *QuantDiagnosticsService) GetFeatureSchema(ctx context.Context, q query.GetFeatureSchemaQuery) (output.GetFeatureSchemaResult, error) {
	return s.diagnostics.GetFeatureSchema(ctx, q.SchemaVersion)
}

func (s *QuantDiagnosticsService) GetModelMetadata(ctx context.Context, q query.GetModelMetadataQuery) (output.GetModelMetadataResult, error) {
	if q.ModelName == "" {
		return output.GetModelMetadataResult{}, shared.NewValidationError("model_name", "must not be empty")
	}
	return s.diagnostics.GetModelMetadata(ctx, output.GetModelMetadataInput{
		ModelName:    q.ModelName,
		ModelVersion: q.ModelVersion,
	})
}

func (s *QuantDiagnosticsService) ReloadApprovedModel(ctx context.Context, cmd command.ReloadApprovedModelCommand) (output.ReloadApprovedModelResult, error) {
	if cmd.ModelName == "" {
		return output.ReloadApprovedModelResult{}, shared.NewValidationError("model_name", "must not be empty")
	}
	if cmd.Stage == "" {
		return output.ReloadApprovedModelResult{}, shared.NewValidationError("stage", "must not be empty")
	}
	return s.diagnostics.ReloadApprovedModel(ctx, output.ReloadApprovedModelInput{
		ModelName: cmd.ModelName,
		Stage:     cmd.Stage,
	})
}
