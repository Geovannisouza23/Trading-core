package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/query"
	"trading-core/internal/interfaces/http/presenter"
)

// QueryHandler serves every read-only /v1 endpoint through the single
// input.QueryService port.
type QueryHandler struct {
	queries input.QueryService
}

func NewQueryHandler(queries input.QueryService) *QueryHandler {
	return &QueryHandler{queries: queries}
}

func (h *QueryHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.GetTradingDashboard(r.Context(), query.GetTradingDashboardQuery{})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) Account(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.GetAccount(r.Context())
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) Positions(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.ListPositions(r.Context())
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) Orders(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.ListOrders(r.Context(), query.ListOrdersQuery{Limit: parseLimit(r, 50)})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) OrderByID(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.GetOrderByID(r.Context(), query.GetOrderByIDQuery{OrderID: chi.URLParam(r, "id")})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) Signals(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.ListSignals(r.Context(), query.ListSignalsQuery{Limit: parseLimit(r, 50)})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) RiskDecisions(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.ListRiskDecisions(r.Context(), query.ListRiskDecisionsQuery{Limit: parseLimit(r, 50)})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) Events(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.ListEvents(r.Context(), query.ListEventsQuery{Limit: parseLimit(r, 50)})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) Incidents(w http.ResponseWriter, r *http.Request) {
	result, err := h.queries.ListIncidents(r.Context(), query.ListIncidentsQuery{Limit: parseLimit(r, 50)})
	if err != nil {
		presenter.Error(w, r, err)
		return
	}
	presenter.JSON(w, http.StatusOK, result)
}

func parseLimit(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
