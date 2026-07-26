package grpc

import (
	"testing"

	"trading-core/internal/application/ports/output"
	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
)

func TestFeatureValueRoundTripsEachOneofVariant(t *testing.T) {
	cases := []output.FeatureValue{
		{Name: "rsi_14", Kind: "numeric", NumericValue: "55.2"},
		{Name: "market_regime", Kind: "categorical", CategoricalValue: "trending_up"},
		{Name: "is_weekend", Kind: "bool", BoolValue: true},
	}

	for _, want := range cases {
		proto := featureValueToProto(want)
		got := featureValueFromProto(proto)
		if got != want {
			t.Errorf("round-trip mismatch: got %+v, want %+v", got, want)
		}
	}
}

func TestFeatureValueFromProtoDefaultsToNumericWhenOneofIsUnset(t *testing.T) {
	entry := &quantv1.FeatureEntry{Name: "unset_feature"}
	got := featureValueFromProto(entry)
	if got.Kind != "numeric" {
		t.Errorf("kind = %q, want numeric (the zero-value oneof case)", got.Kind)
	}
}

func TestJobStatusFromProtoCoversAllSixWireValues(t *testing.T) {
	cases := map[quantv1.JobStatus]string{
		quantv1.JobStatus_JOB_STATUS_UNSPECIFIED: output.JobStatusUnspecified,
		quantv1.JobStatus_JOB_STATUS_QUEUED:      output.JobStatusQueued,
		quantv1.JobStatus_JOB_STATUS_RUNNING:     output.JobStatusRunning,
		quantv1.JobStatus_JOB_STATUS_COMPLETED:   output.JobStatusCompleted,
		quantv1.JobStatus_JOB_STATUS_FAILED:      output.JobStatusFailed,
		quantv1.JobStatus_JOB_STATUS_CANCELLED:   output.JobStatusCancelled,
	}
	for wire, want := range cases {
		if got := jobStatusFromProto(wire); got != want {
			t.Errorf("jobStatusFromProto(%s) = %q, want %q", wire, got, want)
		}
	}
}

func TestTradeRecordFromProtoMapsActionToSide(t *testing.T) {
	buy := tradeRecordFromProto(&quantv1.TradeRecord{Side: quantv1.Action_ACTION_BUY})
	if buy.Side != "BUY" {
		t.Errorf("side = %q, want BUY", buy.Side)
	}
	sell := tradeRecordFromProto(&quantv1.TradeRecord{Side: quantv1.Action_ACTION_SELL})
	if sell.Side != "SELL" {
		t.Errorf("side = %q, want SELL", sell.Side)
	}
}
