package usecase

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/shared"
)

// DatasetExportService implements input.DatasetExportService by delegating
// straight to output.DatasetExportService.
type DatasetExportService struct {
	exports output.DatasetExportService
}

func NewDatasetExportService(exports output.DatasetExportService) *DatasetExportService {
	return &DatasetExportService{exports: exports}
}

var _ input.DatasetExportService = (*DatasetExportService)(nil)

func (s *DatasetExportService) ExportDataset(ctx context.Context, cmd command.ExportDatasetCommand) (output.ExportDatasetResult, error) {
	if cmd.DatasetName == "" {
		return output.ExportDatasetResult{}, shared.NewValidationError("dataset_name", "must not be empty")
	}
	if cmd.DatasetVersion == "" {
		return output.ExportDatasetResult{}, shared.NewValidationError("dataset_version", "must not be empty")
	}
	return s.exports.ExportDataset(ctx, output.ExportDatasetInput{
		DatasetName:    cmd.DatasetName,
		DatasetVersion: cmd.DatasetVersion,
		Symbols:        cmd.Symbols,
		Timeframes:     cmd.Timeframes,
		StartAt:        cmd.StartAt,
		EndAt:          cmd.EndAt,
		LabelVersion:   cmd.LabelVersion,
	})
}
