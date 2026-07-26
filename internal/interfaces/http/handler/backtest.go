package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
	"trading-core/internal/domain/shared"
	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/request"
)

// BacktestHandler serves the backtest job lifecycle: submit, poll, and
// stream progress. Backed by input.BacktestService, which in turn talks
// to the real quant-engine gRPC service (see internal/app/providers.go).
type BacktestHandler struct {
	backtests input.BacktestService
}

func NewBacktestHandler(backtests input.BacktestService) *BacktestHandler {
	return &BacktestHandler{backtests: backtests}
}

func (h *BacktestHandler) Request(w http.ResponseWriter, r *http.Request) {
	var body request.BacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, r, shared.NewValidationError("body", err.Error()))
		return
	}

	config, err := backtestConfigFromRequest(body.Config)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	candles, err := candlesFromRequest(body.Config.Symbol, body.Config.Timeframe, body.Candles)
	if err != nil {
		presenter.Error(w, r, err)
		return
	}

	result, err := h.backtests.RunBacktest(r.Context(), command.RunBacktestCommand{Config: config, Candles: candles})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusAccepted, result)
}

func (h *BacktestHandler) Result(w http.ResponseWriter, r *http.Request) {
	result, err := h.backtests.GetBacktestResult(r.Context(), query.GetBacktestResultQuery{BacktestID: chi.URLParam(r, "id")})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

// Stream serves progress updates over Server-Sent Events until the
// backtest reaches a terminal status or the client disconnects — the
// same "closes after one terminal update" semantics the underlying
// StreamBacktestProgress RPC already has.
func (h *BacktestHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		presenter.Error(w, r, fmt.Errorf("streaming unsupported by this response writer"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	backtestID := chi.URLParam(r, "id")
	err := h.backtests.StreamBacktestProgress(r.Context(), query.StreamBacktestProgressQuery{BacktestID: backtestID}, func(progress output.BacktestProgress) error {
		payload, err := json.Marshal(progress)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
	if err != nil {
		// The stream is already open with a 200 status by this point —
		// an error here can only be reported as one more SSE event, not
		// a fresh HTTP status code.
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", payload)
		flusher.Flush()
	}
}
