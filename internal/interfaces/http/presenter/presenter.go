// Package presenter formats use case results and errors into the
// standardized JSON envelope every handler uses.
package presenter

import (
	"encoding/json"
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

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
		if httpStatus, code, ok := classifyGrpcStatus(err); ok {
			return httpStatus, code
		}
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

// classifyGrpcStatus maps a wrapped gRPC status error (e.g. from the real
// quant-engine client) onto an HTTP status/code pair — quant-engine's own
// docs/grpc-contract.md error-mapping table, one hop further to HTTP.
// status.FromError unwraps through fmt.Errorf's %w chain (verified
// empirically), so this works on the "quant engine RunBacktest: %w"-style
// errors every quant client method returns, not just a bare grpc/status
// error.
func classifyGrpcStatus(err error) (int, string, bool) {
	st, ok := grpcstatus.FromError(err)
	if !ok || st.Code() == codes.OK {
		return 0, "", false
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return http.StatusBadRequest, "INVALID_ARGUMENT", true
	case codes.NotFound:
		return http.StatusNotFound, "NOT_FOUND", true
	case codes.AlreadyExists:
		return http.StatusConflict, "ALREADY_EXISTS", true
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests, "RESOURCE_EXHAUSTED", true
	case codes.Unavailable:
		return http.StatusServiceUnavailable, "QUANT_ENGINE_UNAVAILABLE", true
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout, "DEADLINE_EXCEEDED", true
	case codes.Unauthenticated:
		return http.StatusUnauthorized, "UNAUTHENTICATED", true
	case codes.PermissionDenied:
		return http.StatusForbidden, "PERMISSION_DENIED", true
	default:
		return http.StatusInternalServerError, "QUANT_ENGINE_ERROR", true
	}
}
