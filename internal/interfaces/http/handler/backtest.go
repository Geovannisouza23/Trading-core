package handler

import (
	"net/http"

	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/response"
)

// BacktestRequest is intentionally not implemented: no backtest engine or
// use case is defined anywhere else in the spec beyond this single
// endpoint, and faking a "queued" response would be misleading. It returns
// a clear, structured 501 instead of silently accepting work it can't do.
func BacktestRequest(w http.ResponseWriter, r *http.Request) {
	presenter.JSON(w, http.StatusNotImplemented, response.NotImplementedResponse{
		Status:  "not_implemented",
		Message: "backtesting is not implemented in this delivery; see docs/architecture.md",
	})
}
