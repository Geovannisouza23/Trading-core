package output

import (
	"context"
	"time"

	"trading-core/internal/domain/market"
)

type ValidateStrategyInput struct {
	StrategyName    string
	StrategyVersion string
	Params          map[string]string
}

type ValidateStrategyResult struct {
	Valid             bool     `json:"valid"`
	Errors            []string `json:"errors,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	MinimumCandles    uint32   `json:"minimum_candles"`
	CompatibleRegimes []string `json:"compatible_regimes,omitempty"`
}

// FeatureValue mirrors quant-engine's FeatureEntry oneof: exactly one of
// NumericValue/CategoricalValue/BoolValue is meaningful, selected by Kind.
type FeatureValue struct {
	Name             string `json:"name"`
	Kind             string `json:"kind"` // "numeric" | "categorical" | "bool"
	NumericValue     string `json:"numeric_value,omitempty"`
	CategoricalValue string `json:"categorical_value,omitempty"`
	BoolValue        bool   `json:"bool_value,omitempty"`
}

type CalculateFeaturesInput struct {
	Symbol               string
	Timeframe            string
	Candles              []market.Candle
	FeatureSchemaVersion string
}

type DataQualityReport struct {
	TotalCandles      uint32   `json:"total_candles"`
	ValidCandles      uint32   `json:"valid_candles"`
	DuplicateCandles  uint32   `json:"duplicate_candles"`
	GapCount          uint32   `json:"gap_count"`
	OutOfOrderCandles uint32   `json:"out_of_order_candles"`
	StaleCandles      uint32   `json:"stale_candles"`
	IncompleteCandles uint32   `json:"incomplete_candles"`
	QualityScore      string   `json:"quality_score"`
	Warnings          []string `json:"warnings,omitempty"`
}

type CalculateFeaturesResult struct {
	FeatureSetID      string             `json:"feature_set_id"`
	SchemaVersion     string             `json:"schema_version"`
	Features          []FeatureValue     `json:"features"`
	Quality           *DataQualityReport `json:"quality,omitempty"`
	DeterministicHash string             `json:"deterministic_hash"`
	CalculatedAt      time.Time          `json:"calculated_at"`
}

type ModelPrediction struct {
	ModelName            string        `json:"model_name"`
	ModelVersion         string        `json:"model_version"`
	FeatureSchemaVersion string        `json:"feature_schema_version"`
	PredictedTarget      string        `json:"predicted_target"`
	SuccessProbability   string        `json:"success_probability"`
	ExpectedReturn       string        `json:"expected_return"`
	StopProbability      string        `json:"stop_probability"`
	Confidence           string        `json:"confidence"`
	InferenceTimestamp   time.Time     `json:"inference_timestamp"`
	InferenceDuration    time.Duration `json:"inference_duration_ns"`
	Warnings             []string      `json:"warnings,omitempty"`
}

type EvaluateModelInput struct {
	ModelName            string
	ModelVersion         string
	Features             []FeatureValue
	FeatureSchemaVersion string
}

type RegisterDecisionOutcomeInput struct {
	DecisionID      string
	OrderCreated    bool
	Executed        bool
	RejectionReason string
	EntryPrice      string
	ExitPrice       string
	Quantity        string
	Fees            string
	Slippage        string
	Pnl             string
	ExitReason      string
	EntryAt         time.Time
	ExitAt          time.Time
}

type RegisterDecisionOutcomeResult struct {
	Accepted         bool   `json:"accepted"`
	TrainingRecordID string `json:"training_record_id,omitempty"`
}

type FeatureSchemaEntry struct {
	Name        string `json:"name"`
	DataType    string `json:"data_type"` // "numeric" | "categorical" | "bool"
	Description string `json:"description"`
	Nullable    bool   `json:"nullable"`
	Source      string `json:"source"`
}

type GetFeatureSchemaResult struct {
	SchemaVersion string               `json:"schema_version"`
	Features      []FeatureSchemaEntry `json:"features"`
}

type GetModelMetadataInput struct {
	ModelName    string
	ModelVersion string
}

type GetModelMetadataResult struct {
	ModelName            string     `json:"model_name"`
	ModelVersion         string     `json:"model_version"`
	State                string     `json:"state"`
	FeatureSchemaVersion string     `json:"feature_schema_version"`
	ArtifactSHA256       string     `json:"artifact_sha256"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
	Retired              bool       `json:"retired"`
}

type ReloadApprovedModelInput struct {
	ModelName string
	Stage     string // "APPROVED" | "CANARY" | "PRODUCTION"
}

type ReloadApprovedModelResult struct {
	Reloaded     bool   `json:"reloaded"`
	ModelVersion string `json:"model_version,omitempty"`
	Message      string `json:"message,omitempty"`
}

// QuantDiagnostics is the port to quant-engine's remaining read/diagnostic
// and one-off RPCs: strategy validation, direct feature/model access, the
// decision-outcome feedback loop, and model-registry introspection/reload.
// Grouped together because none of them is part of a submit/poll job
// lifecycle (see BacktestService/OptimizationService for that shape).
type QuantDiagnostics interface {
	ValidateStrategy(ctx context.Context, input ValidateStrategyInput) (ValidateStrategyResult, error)
	CalculateFeatures(ctx context.Context, input CalculateFeaturesInput) (CalculateFeaturesResult, error)
	EvaluateModel(ctx context.Context, input EvaluateModelInput) (ModelPrediction, error)
	RegisterDecisionOutcome(ctx context.Context, input RegisterDecisionOutcomeInput) (RegisterDecisionOutcomeResult, error)
	GetFeatureSchema(ctx context.Context, schemaVersion string) (GetFeatureSchemaResult, error)
	GetModelMetadata(ctx context.Context, input GetModelMetadataInput) (GetModelMetadataResult, error)
	ReloadApprovedModel(ctx context.Context, input ReloadApprovedModelInput) (ReloadApprovedModelResult, error)
}
