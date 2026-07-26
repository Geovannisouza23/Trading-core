package handler

import (
	"encoding/json"
	"net/http"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/domain/shared"
	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/request"
)

// StrategyHandler asks quant-engine whether a strategy name/version/params
// combination is valid and usable. Backed by input.QuantDiagnosticsService.
type StrategyHandler struct {
	diagnostics input.QuantDiagnosticsService
}

func NewStrategyHandler(diagnostics input.QuantDiagnosticsService) *StrategyHandler {
	return &StrategyHandler{diagnostics: diagnostics}
}

func (h *StrategyHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var body request.ValidateStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	result, err := h.diagnostics.ValidateStrategy(r.Context(), command.ValidateStrategyCommand{
		StrategyName:    body.StrategyName,
		StrategyVersion: body.StrategyVersion,
		Params:          body.Params,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}
