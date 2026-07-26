package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/domain/shared"
	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/request"
)

// DatasetHandler triggers a Parquet dataset export on quant-engine.
// Backed by input.DatasetExportService.
type DatasetHandler struct {
	exports input.DatasetExportService
}

func NewDatasetHandler(exports input.DatasetExportService) *DatasetHandler {
	return &DatasetHandler{exports: exports}
}

func (h *DatasetHandler) Export(w http.ResponseWriter, r *http.Request) {
	var body request.DatasetExportRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	startAt, err := time.Parse(time.RFC3339, body.From)
	if err != nil {
		presenter.Error(w, r, shared.NewValidationError("from", err.Error()))
		return
	}
	endAt, err := time.Parse(time.RFC3339, body.To)
	if err != nil {
		presenter.Error(w, r, shared.NewValidationError("to", err.Error()))
		return
	}

	result, err := h.exports.ExportDataset(r.Context(), command.ExportDatasetCommand{
		DatasetName:    body.DatasetName,
		DatasetVersion: body.DatasetVersion,
		Symbols:        body.Symbols,
		Timeframes:     body.Timeframes,
		StartAt:        startAt,
		EndAt:          endAt,
		LabelVersion:   body.LabelVersion,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusAccepted, result)
}
