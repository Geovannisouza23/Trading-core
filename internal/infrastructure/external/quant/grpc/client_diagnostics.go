package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"trading-core/internal/application/ports/output"
	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
)

func featureValueToProto(v output.FeatureValue) *quantv1.FeatureEntry {
	entry := &quantv1.FeatureEntry{Name: v.Name}
	switch v.Kind {
	case "categorical":
		entry.Value = &quantv1.FeatureEntry_CategoricalValue{CategoricalValue: v.CategoricalValue}
	case "bool":
		entry.Value = &quantv1.FeatureEntry_BoolValue{BoolValue: v.BoolValue}
	default:
		entry.Value = &quantv1.FeatureEntry_NumericValue{NumericValue: v.NumericValue}
	}
	return entry
}

func featureValuesToProto(values []output.FeatureValue) []*quantv1.FeatureEntry {
	out := make([]*quantv1.FeatureEntry, len(values))
	for i, v := range values {
		out[i] = featureValueToProto(v)
	}
	return out
}

func featureValueFromProto(entry *quantv1.FeatureEntry) output.FeatureValue {
	value := output.FeatureValue{Name: entry.GetName()}
	switch entry.GetValue().(type) {
	case *quantv1.FeatureEntry_CategoricalValue:
		value.Kind = "categorical"
		value.CategoricalValue = entry.GetCategoricalValue()
	case *quantv1.FeatureEntry_BoolValue:
		value.Kind = "bool"
		value.BoolValue = entry.GetBoolValue()
	default:
		value.Kind = "numeric"
		value.NumericValue = entry.GetNumericValue()
	}
	return value
}

func featureValuesFromProto(entries []*quantv1.FeatureEntry) []output.FeatureValue {
	out := make([]output.FeatureValue, len(entries))
	for i, e := range entries {
		out[i] = featureValueFromProto(e)
	}
	return out
}

func dataQualityFromProto(q *quantv1.DataQualityReport) *output.DataQualityReport {
	if q == nil {
		return nil
	}
	return &output.DataQualityReport{
		TotalCandles:      q.GetTotalCandles(),
		ValidCandles:      q.GetValidCandles(),
		DuplicateCandles:  q.GetDuplicateCandles(),
		GapCount:          q.GetGapCount(),
		OutOfOrderCandles: q.GetOutOfOrderCandles(),
		StaleCandles:      q.GetStaleCandles(),
		IncompleteCandles: q.GetIncompleteCandles(),
		QualityScore:      q.GetQualityScore(),
		Warnings:          q.GetWarnings(),
	}
}

func modelPredictionFromProto(p *quantv1.ModelPrediction) output.ModelPrediction {
	prediction := output.ModelPrediction{
		ModelName:            p.GetModelName(),
		ModelVersion:         p.GetModelVersion(),
		FeatureSchemaVersion: p.GetFeatureSchemaVersion(),
		PredictedTarget:      p.GetPredictedTarget(),
		SuccessProbability:   p.GetSuccessProbability(),
		ExpectedReturn:       p.GetExpectedReturn(),
		StopProbability:      p.GetStopProbability(),
		Confidence:           p.GetConfidence(),
		Warnings:             p.GetWarnings(),
	}
	if p.GetInferenceTimestamp() != nil {
		prediction.InferenceTimestamp = p.GetInferenceTimestamp().AsTime()
	}
	if p.GetInferenceDuration() != nil {
		prediction.InferenceDuration = p.GetInferenceDuration().AsDuration()
	}
	return prediction
}

func (c *Client) ValidateStrategy(ctx context.Context, input output.ValidateStrategyInput) (output.ValidateStrategyResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.ValidateStrategy(ctx, &quantv1.ValidateStrategyRequest{
		StrategyName:    input.StrategyName,
		StrategyVersion: input.StrategyVersion,
		Params:          input.Params,
	})
	if err != nil {
		return output.ValidateStrategyResult{}, fmt.Errorf("quant engine ValidateStrategy: %w", err)
	}

	regimes := make([]string, len(resp.GetCompatibleRegimes()))
	for i, r := range resp.GetCompatibleRegimes() {
		regimes[i] = string(regimeFromProto(r))
	}

	return output.ValidateStrategyResult{
		Valid:             resp.GetValid(),
		Errors:            resp.GetErrors(),
		Warnings:          resp.GetWarnings(),
		MinimumCandles:    resp.GetMinimumCandles(),
		CompatibleRegimes: regimes,
	}, nil
}

func (c *Client) CalculateFeatures(ctx context.Context, input output.CalculateFeaturesInput) (output.CalculateFeaturesResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.CalculateFeatures(ctx, &quantv1.CalculateFeaturesRequest{
		RequestId:            uuid.NewString(),
		Symbol:               input.Symbol,
		Timeframe:            input.Timeframe,
		Candles:              candlesToProto(input.Candles),
		FeatureSchemaVersion: input.FeatureSchemaVersion,
	})
	if err != nil {
		return output.CalculateFeaturesResult{}, fmt.Errorf("quant engine CalculateFeatures: %w", err)
	}

	result := output.CalculateFeaturesResult{
		FeatureSetID:      resp.GetFeatureSetId(),
		SchemaVersion:     resp.GetSchemaVersion(),
		Features:          featureValuesFromProto(resp.GetFeatures()),
		Quality:           dataQualityFromProto(resp.GetQuality()),
		DeterministicHash: resp.GetDeterministicHash(),
	}
	if resp.GetCalculatedAt() != nil {
		result.CalculatedAt = resp.GetCalculatedAt().AsTime()
	}
	return result, nil
}

func (c *Client) EvaluateModel(ctx context.Context, input output.EvaluateModelInput) (output.ModelPrediction, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.EvaluateModel(ctx, &quantv1.EvaluateModelRequest{
		RequestId:            uuid.NewString(),
		ModelName:            input.ModelName,
		ModelVersion:         input.ModelVersion,
		Features:             featureValuesToProto(input.Features),
		FeatureSchemaVersion: input.FeatureSchemaVersion,
	})
	if err != nil {
		return output.ModelPrediction{}, fmt.Errorf("quant engine EvaluateModel: %w", err)
	}
	return modelPredictionFromProto(resp.GetPrediction()), nil
}

func (c *Client) RegisterDecisionOutcome(ctx context.Context, input output.RegisterDecisionOutcomeInput) (output.RegisterDecisionOutcomeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.RegisterDecisionOutcome(ctx, &quantv1.RegisterDecisionOutcomeRequest{
		DecisionId:      input.DecisionID,
		OrderCreated:    input.OrderCreated,
		Executed:        input.Executed,
		RejectionReason: input.RejectionReason,
		EntryPrice:      input.EntryPrice,
		ExitPrice:       input.ExitPrice,
		Quantity:        input.Quantity,
		Fees:            input.Fees,
		Slippage:        input.Slippage,
		Pnl:             input.Pnl,
		ExitReason:      input.ExitReason,
		EntryAt:         timestamppb.New(input.EntryAt),
		ExitAt:          timestamppb.New(input.ExitAt),
	})
	if err != nil {
		return output.RegisterDecisionOutcomeResult{}, fmt.Errorf("quant engine RegisterDecisionOutcome: %w", err)
	}
	return output.RegisterDecisionOutcomeResult{
		Accepted:         resp.GetAccepted(),
		TrainingRecordID: resp.GetTrainingRecordId(),
	}, nil
}

func (c *Client) GetFeatureSchema(ctx context.Context, schemaVersion string) (output.GetFeatureSchemaResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.GetFeatureSchema(ctx, &quantv1.GetFeatureSchemaRequest{SchemaVersion: schemaVersion})
	if err != nil {
		return output.GetFeatureSchemaResult{}, fmt.Errorf("quant engine GetFeatureSchema: %w", err)
	}

	features := make([]output.FeatureSchemaEntry, len(resp.GetFeatures()))
	for i, f := range resp.GetFeatures() {
		features[i] = output.FeatureSchemaEntry{
			Name:        f.GetName(),
			DataType:    f.GetDataType(),
			Description: f.GetDescription(),
			Nullable:    f.GetNullable(),
			Source:      f.GetSource(),
		}
	}

	return output.GetFeatureSchemaResult{
		SchemaVersion: resp.GetSchemaVersion(),
		Features:      features,
	}, nil
}

func (c *Client) GetModelMetadata(ctx context.Context, input output.GetModelMetadataInput) (output.GetModelMetadataResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.GetModelMetadata(ctx, &quantv1.GetModelMetadataRequest{
		ModelName:    input.ModelName,
		ModelVersion: input.ModelVersion,
	})
	if err != nil {
		return output.GetModelMetadataResult{}, fmt.Errorf("quant engine GetModelMetadata: %w", err)
	}

	result := output.GetModelMetadataResult{
		ModelName:            resp.GetModelName(),
		ModelVersion:         resp.GetModelVersion(),
		State:                resp.GetState(),
		FeatureSchemaVersion: resp.GetFeatureSchemaVersion(),
		ArtifactSHA256:       resp.GetArtifactSha256(),
		Retired:              resp.GetRetired(),
	}
	if resp.GetApprovedAt() != nil {
		approvedAt := resp.GetApprovedAt().AsTime()
		result.ApprovedAt = &approvedAt
	}
	return result, nil
}

func (c *Client) ReloadApprovedModel(ctx context.Context, input output.ReloadApprovedModelInput) (output.ReloadApprovedModelResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.ReloadApprovedModel(ctx, &quantv1.ReloadApprovedModelRequest{
		ModelName: input.ModelName,
		Stage:     input.Stage,
	})
	if err != nil {
		return output.ReloadApprovedModelResult{}, fmt.Errorf("quant engine ReloadApprovedModel: %w", err)
	}
	return output.ReloadApprovedModelResult{
		Reloaded:     resp.GetReloaded(),
		ModelVersion: resp.GetModelVersion(),
		Message:      resp.GetMessage(),
	}, nil
}

var _ output.QuantDiagnostics = (*Client)(nil)
