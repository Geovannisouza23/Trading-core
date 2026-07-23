// Package handler holds thin HTTP handlers: validate the request, call an
// input port, map the result through a presenter. No handler here imports
// PostgreSQL, a broker, an LLM client, or opens a transaction.
package handler

import (
	"context"
	"net/http"

	"trading-core/internal/interfaces/http/presenter"
	"trading-core/internal/interfaces/http/response"
)

// Health always returns 200 once the process is up; it does not check any
// dependency (that's /ready's job).
func Health(w http.ResponseWriter, r *http.Request) {
	presenter.JSON(w, http.StatusOK, response.StatusResponse{Status: "ok"})
}

// ReadinessChecker verifies that a single essential dependency (e.g. the
// database) is reachable. Handlers never import pgx/database directly;
// internal/app wires a concrete checker in.
type ReadinessChecker func(ctx context.Context) error

// Ready reports 200 only when every checker succeeds.
func Ready(checkers ...ReadinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, check := range checkers {
			if err := check(r.Context()); err != nil {
				presenter.JSON(w, http.StatusServiceUnavailable, response.ReadyResponse{Status: "not_ready", Error: err.Error()})
				return
			}
		}
		presenter.JSON(w, http.StatusOK, response.ReadyResponse{Status: "ready"})
	}
}
