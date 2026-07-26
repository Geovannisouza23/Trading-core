package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/query"
	"trading-core/internal/domain/shared"
	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/request"
)

// QuantDiagnosticsHandler serves quant-engine's remaining read/diagnostic
// and one-off capabilities: direct feature/model access, the
// decision-outcome feedback loop, and model-registry introspection/reload.
// Backed by input.QuantDiagnosticsService (synchronous, gRPC) and
// input.QuantEventPublisherService (asynchronous, NATS fan-out — see
// Publish* below).
type QuantDiagnosticsHandler struct {
	diagnostics input.QuantDiagnosticsService
	publisher   input.QuantEventPublisherService
}

func NewQuantDiagnosticsHandler(diagnostics input.QuantDiagnosticsService, publisher input.QuantEventPublisherService) *QuantDiagnosticsHandler {
	return &QuantDiagnosticsHandler{diagnostics: diagnostics, publisher: publisher}
}

func (h *QuantDiagnosticsHandler) CalculateFeatures(w http.ResponseWriter, r *http.Request) {
	var body request.CalculateFeaturesRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	candles, err := candlesFromRequest(body.Symbol, body.Timeframe, body.Candles)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}

	result, err := h.diagnostics.CalculateFeatures(r.Context(), command.CalculateFeaturesCommand{
		Symbol:               body.Symbol,
		Timeframe:            body.Timeframe,
		Candles:              candles,
		FeatureSchemaVersion: body.FeatureSchemaVersion,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QuantDiagnosticsHandler) EvaluateModel(w http.ResponseWriter, r *http.Request) {
	var body request.EvaluateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	result, err := h.diagnostics.EvaluateModel(r.Context(), command.EvaluateModelCommand{
		ModelName:            body.ModelName,
		ModelVersion:         body.ModelVersion,
		Features:             featureValuesFromRequest(body.Features),
		FeatureSchemaVersion: body.FeatureSchemaVersion,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QuantDiagnosticsHandler) RegisterDecisionOutcome(w http.ResponseWriter, r *http.Request) {
	var body request.RegisterDecisionOutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	entryAt, err := time.Parse(time.RFC3339, body.EntryAt)
	if err != nil {
		presenter.Error(w, r, shared.NewValidationError("entry_at", err.Error()))
		return
	}
	exitAt, err := time.Parse(time.RFC3339, body.ExitAt)
	if err != nil {
		presenter.Error(w, r, shared.NewValidationError("exit_at", err.Error()))
		return
	}

	result, err := h.diagnostics.RegisterDecisionOutcome(r.Context(), command.RegisterDecisionOutcomeCommand{
		DecisionID:      chi.URLParam(r, "decision_id"),
		OrderCreated:    body.OrderCreated,
		Executed:        body.Executed,
		RejectionReason: body.RejectionReason,
		EntryPrice:      body.EntryPrice,
		ExitPrice:       body.ExitPrice,
		Quantity:        body.Quantity,
		Fees:            body.Fees,
		Slippage:        body.Slippage,
		Pnl:             body.Pnl,
		ExitReason:      body.ExitReason,
		EntryAt:         entryAt,
		ExitAt:          exitAt,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

// PublishDecisionOutcome is the async counterpart to RegisterDecisionOutcome:
// fire-and-forget over NATS (quant.decision.outcome.received) instead of a
// synchronous gRPC call, reaching every quant-engine process/replica
// instead of just the one RegisterDecisionOutcome happens to dial.
func (h *QuantDiagnosticsHandler) PublishDecisionOutcome(w http.ResponseWriter, r *http.Request) {
	var body request.RegisterDecisionOutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	entryAt, err := time.Parse(time.RFC3339, body.EntryAt)
	if err != nil {
		presenter.Error(w, r, shared.NewValidationError("entry_at", err.Error()))
		return
	}
	exitAt, err := time.Parse(time.RFC3339, body.ExitAt)
	if err != nil {
		presenter.Error(w, r, shared.NewValidationError("exit_at", err.Error()))
		return
	}

	err = h.publisher.PublishDecisionOutcome(r.Context(), command.RegisterDecisionOutcomeCommand{
		DecisionID:      chi.URLParam(r, "decision_id"),
		OrderCreated:    body.OrderCreated,
		Executed:        body.Executed,
		RejectionReason: body.RejectionReason,
		EntryPrice:      body.EntryPrice,
		ExitPrice:       body.ExitPrice,
		Quantity:        body.Quantity,
		Fees:            body.Fees,
		Slippage:        body.Slippage,
		Pnl:             body.Pnl,
		ExitReason:      body.ExitReason,
		EntryAt:         entryAt,
		ExitAt:          exitAt,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusAccepted, map[string]string{"status": "published"})
}

// PublishModelApproved is the async counterpart to ReloadModel:
// fire-and-forget over NATS (quant.model.approved) instead of a
// synchronous gRPC call.
func (h *QuantDiagnosticsHandler) PublishModelApproved(w http.ResponseWriter, r *http.Request) {
	var body request.ReloadApprovedModelRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	err := h.publisher.PublishModelApproved(r.Context(), command.ReloadApprovedModelCommand{
		ModelName: body.ModelName,
		Stage:     body.Stage,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusAccepted, map[string]string{"status": "published"})
}

func (h *QuantDiagnosticsHandler) FeatureSchema(w http.ResponseWriter, r *http.Request) {
	result, err := h.diagnostics.GetFeatureSchema(r.Context(), query.GetFeatureSchemaQuery{
		SchemaVersion: r.URL.Query().Get("version"),
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QuantDiagnosticsHandler) ModelMetadata(w http.ResponseWriter, r *http.Request) {
	result, err := h.diagnostics.GetModelMetadata(r.Context(), query.GetModelMetadataQuery{
		ModelName:    r.URL.Query().Get("model_name"),
		ModelVersion: r.URL.Query().Get("model_version"),
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QuantDiagnosticsHandler) ReloadModel(w http.ResponseWriter, r *http.Request) {
	var body request.ReloadApprovedModelRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	result, err := h.diagnostics.ReloadApprovedModel(r.Context(), command.ReloadApprovedModelCommand{
		ModelName: body.ModelName,
		Stage:     body.Stage,
	})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}
