package grpc

import (
	"context"
	"fmt"

	"trading-core/internal/application/ports/output"
)

// Unavailable stands in for the backtest/optimization/dataset/diagnostics
// capabilities when QUANT_MODE is not "grpc" — unlike the realtime
// EvaluateSignal/AnalyzeMarketRegime path (Engine, fake.go), there is no
// meaningful deterministic fake to simulate a backtest or a model
// registry reload, so every method here fails clearly and immediately
// instead of pretending. Mirrors this codebase's own precedent: the
// pre-integration backtest HTTP handler returned a clean 501 rather than
// fabricating a response.
type Unavailable struct {
	mode string
}

func NewUnavailable(mode string) *Unavailable {
	return &Unavailable{mode: mode}
}

func (u *Unavailable) err(capability string) error {
	return fmt.Errorf("quant engine %s is unavailable in quant.mode=%q; set QUANT_MODE=grpc against a running quant-engine", capability, u.mode)
}

var (
	_ output.BacktestService      = (*Unavailable)(nil)
	_ output.OptimizationService  = (*Unavailable)(nil)
	_ output.DatasetExportService = (*Unavailable)(nil)
	_ output.QuantDiagnostics     = (*Unavailable)(nil)
)

func (u *Unavailable) RunBacktest(context.Context, output.RunBacktestInput) (output.RunBacktestResult, error) {
	return output.RunBacktestResult{}, u.err("RunBacktest")
}

func (u *Unavailable) GetBacktestResult(context.Context, string) (output.GetBacktestResultResult, error) {
	return output.GetBacktestResultResult{}, u.err("GetBacktestResult")
}

func (u *Unavailable) StreamBacktestProgress(context.Context, string, func(output.BacktestProgress) error) error {
	return u.err("StreamBacktestProgress")
}

func (u *Unavailable) RunOptimization(context.Context, output.RunOptimizationInput) (output.RunOptimizationResult, error) {
	return output.RunOptimizationResult{}, u.err("RunOptimization")
}

func (u *Unavailable) GetOptimizationResult(context.Context, string) (output.GetOptimizationResultResult, error) {
	return output.GetOptimizationResultResult{}, u.err("GetOptimizationResult")
}

func (u *Unavailable) ExportDataset(context.Context, output.ExportDatasetInput) (output.ExportDatasetResult, error) {
	return output.ExportDatasetResult{}, u.err("ExportDataset")
}

func (u *Unavailable) ValidateStrategy(context.Context, output.ValidateStrategyInput) (output.ValidateStrategyResult, error) {
	return output.ValidateStrategyResult{}, u.err("ValidateStrategy")
}

func (u *Unavailable) CalculateFeatures(context.Context, output.CalculateFeaturesInput) (output.CalculateFeaturesResult, error) {
	return output.CalculateFeaturesResult{}, u.err("CalculateFeatures")
}

func (u *Unavailable) EvaluateModel(context.Context, output.EvaluateModelInput) (output.ModelPrediction, error) {
	return output.ModelPrediction{}, u.err("EvaluateModel")
}

func (u *Unavailable) RegisterDecisionOutcome(context.Context, output.RegisterDecisionOutcomeInput) (output.RegisterDecisionOutcomeResult, error) {
	return output.RegisterDecisionOutcomeResult{}, u.err("RegisterDecisionOutcome")
}

func (u *Unavailable) GetFeatureSchema(context.Context, string) (output.GetFeatureSchemaResult, error) {
	return output.GetFeatureSchemaResult{}, u.err("GetFeatureSchema")
}

func (u *Unavailable) GetModelMetadata(context.Context, output.GetModelMetadataInput) (output.GetModelMetadataResult, error) {
	return output.GetModelMetadataResult{}, u.err("GetModelMetadata")
}

func (u *Unavailable) ReloadApprovedModel(context.Context, output.ReloadApprovedModelInput) (output.ReloadApprovedModelResult, error) {
	return output.ReloadApprovedModelResult{}, u.err("ReloadApprovedModel")
}
