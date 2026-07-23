package input

import (
	"context"

	"trading-core/internal/application/dto"
	"trading-core/internal/application/query"
)

// QueryService groups every read-only use case behind one port so HTTP
// handlers depend on a single, cohesive read-model interface instead of one
// interface per GET endpoint.
type QueryService interface {
	GetTradingDashboard(ctx context.Context, q query.GetTradingDashboardQuery) (dto.DashboardDTO, error)
	GetAccount(ctx context.Context) (dto.AccountDTO, error)
	ListPositions(ctx context.Context) ([]dto.PositionDTO, error)
	ListOrders(ctx context.Context, q query.ListOrdersQuery) ([]dto.OrderDTO, error)
	GetOrderByID(ctx context.Context, q query.GetOrderByIDQuery) (dto.OrderDTO, error)
	ListSignals(ctx context.Context, q query.ListSignalsQuery) ([]dto.SignalDTO, error)
	ListRiskDecisions(ctx context.Context, q query.ListRiskDecisionsQuery) ([]dto.RiskDecisionDTO, error)
	ListEvents(ctx context.Context, q query.ListEventsQuery) ([]dto.MarketEventDTO, error)
	ListIncidents(ctx context.Context, q query.ListIncidentsQuery) ([]dto.IncidentDTO, error)
}
