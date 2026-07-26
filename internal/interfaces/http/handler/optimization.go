package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/query"
	"trading-core/internal/domain/shared"
	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/request"
)

// OptimizationHandler serves the hyperparameter search job lifecycle:
// submit and poll. Backed by input.OptimizationService.
type OptimizationHandler struct {
	optimizations input.OptimizationService
}

func NewOptimizationHandler(optimizations input.OptimizationService) *OptimizationHandler {
	return &OptimizationHandler{optimizations: optimizations}
}

func (h *OptimizationHandler) Request(w http.ResponseWriter, r *http.Request) {
	var body request.OptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	baseConfig, err := backtestConfigFromRequest(body.BaseConfig)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	candles, err := candlesFromRequest(body.BaseConfig.Symbol, body.BaseConfig.Timeframe, body.Candles)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}

	result, err := h.optimizations.RunOptimization(r.Context(), command.RunOptimizationCommand{
		BaseConfig:         baseConfig,
		Candles:            candles,
		SearchSpace:        parameterRangesFromRequest(body.SearchSpace),
		Algorithm:          body.Algorithm,
		Objective:          body.Objective,
		MaxIterations:      body.MaxIterations,
		WalkForward:        body.WalkForward,
		WalkForwardWindows: body.WalkForwardWindows,
		MonteCarlo:         body.MonteCarlo,
		MonteCarloRuns:     body.MonteCarloRuns,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusAccepted, result)
}

func (h *OptimizationHandler) Result(w http.ResponseWriter, r *http.Request) {
	result, err := h.optimizations.GetOptimizationResult(r.Context(), query.GetOptimizationResultQuery{OptimizationID: chi.URLParam(r, "id")})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}
