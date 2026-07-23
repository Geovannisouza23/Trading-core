// Package presenter formats use case results and errors into the
// standardized JSON envelope every handler uses.
package presenter

import (
	"encoding/json"
	"errors"
	"net/http"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/usecase"
	"trading-core/internal/domain/shared"
	"trading-core/internal/interfaces/http/middleware"
)

type ErrorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

type errorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// JSON writes body as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Error maps err to the standardized error envelope and an appropriate
// HTTP status code.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	status, code := classify(err)
	JSON(w, status, errorEnvelope{Error: ErrorBody{
		Code:      code,
		Message:   err.Error(),
		RequestID: middleware.RequestIDFromContext(r.Context()),
	}})
}

func classify(err error) (int, string) {
	var validationErr *shared.ValidationError
	var conflictErr *shared.ConflictError

	switch {
	case errors.Is(err, output.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, output.ErrOptimisticLock):
		return http.StatusConflict, "VERSION_CONFLICT"
	case errors.Is(err, usecase.ErrOperationalModeBlocked):
		return http.StatusBadRequest, "INVALID_OPERATION_MODE"
	case errors.Is(err, usecase.ErrDuplicateExecution):
		return http.StatusConflict, "DUPLICATE_EXECUTION"
	case errors.Is(err, usecase.ErrStaleMarketData):
		return http.StatusBadRequest, "STALE_MARKET_DATA"
	case errors.Is(err, usecase.ErrBrokerOrderStateUnknown):
		return http.StatusConflict, "BROKER_ORDER_STATE_UNKNOWN"
	case errors.As(err, &validationErr):
		return http.StatusBadRequest, "VALIDATION_ERROR"
	case errors.As(err, &conflictErr):
		return http.StatusConflict, "INVALID_STATE_TRANSITION"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}
