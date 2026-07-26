package input

import (
	"context"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/output"
)

// DatasetExportService is the port for triggering a Parquet dataset
// export job on quant-engine.
type DatasetExportService interface {
	ExportDataset(ctx context.Context, cmd command.ExportDatasetCommand) (output.ExportDatasetResult, error)
}
