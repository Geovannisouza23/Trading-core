package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"trading-core/internal/application/ports/output"
	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
)

func (c *Client) ExportDataset(ctx context.Context, input output.ExportDatasetInput) (output.ExportDatasetResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.stub.ExportDataset(ctx, &quantv1.ExportDatasetRequest{
		RequestId:      uuid.NewString(),
		DatasetName:    input.DatasetName,
		DatasetVersion: input.DatasetVersion,
		Symbols:        input.Symbols,
		Timeframes:     input.Timeframes,
		StartAt:        timestamppb.New(input.StartAt),
		EndAt:          timestamppb.New(input.EndAt),
		LabelVersion:   input.LabelVersion,
	})
	if err != nil {
		return output.ExportDatasetResult{}, fmt.Errorf("quant engine ExportDataset: %w", err)
	}
	return output.ExportDatasetResult{
		ExportID: resp.GetExportId(),
		Status:   jobStatusFromProto(resp.GetStatus()),
	}, nil
}

var _ output.DatasetExportService = (*Client)(nil)
