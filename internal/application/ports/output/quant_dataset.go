package output

import (
	"context"
	"time"
)

type ExportDatasetInput struct {
	DatasetName    string
	DatasetVersion string
	Symbols        []string
	Timeframes     []string
	StartAt        time.Time
	EndAt          time.Time
	LabelVersion   string
}

type ExportDatasetResult struct {
	ExportID string `json:"export_id"`
	Status   string `json:"status"`
}

// DatasetExportService is the port to quant-engine's Parquet dataset
// export. There is no dedicated poll RPC on the wire contract for this
// job kind (unlike backtest/optimization) — the only status available is
// the one returned by ExportDataset itself.
type DatasetExportService interface {
	ExportDataset(ctx context.Context, input ExportDatasetInput) (ExportDatasetResult, error)
}
