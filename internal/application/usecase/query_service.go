package usecase

import (
	"context"
	"fmt"

	"trading-core/internal/application/dto"
	"trading-core/internal/application/mapper"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/query"
	"trading-core/internal/domain/shared"
)

// QueryService implements input.QueryService: every read-only endpoint the
// HTTP API exposes, kept together because none of them carry business logic
// beyond loading and mapping.
type QueryService struct {
	accounts  output.AccountRepository
	positions output.PositionRepository
	orders    output.OrderRepository
	signals   output.TradeSignalRepository
	decisions output.RiskDecisionRepository
	events    output.MarketEventRepository
	incidents output.SystemIncidentRepository
	broker    output.Broker
}

func NewQueryService(
	accounts output.AccountRepository,
	positions output.PositionRepository,
	orders output.OrderRepository,
	signals output.TradeSignalRepository,
	decisions output.RiskDecisionRepository,
	events output.MarketEventRepository,
	incidents output.SystemIncidentRepository,
	broker output.Broker,
) *QueryService {
	return &QueryService{
		accounts: accounts, positions: positions, orders: orders,
		signals: signals, decisions: decisions, events: events,
		incidents: incidents, broker: broker,
	}
}

var _ input.QueryService = (*QueryService)(nil)

func (q *QueryService) GetTradingDashboard(ctx context.Context, _ query.GetTradingDashboardQuery) (dto.DashboardDTO, error) {
	acct, err := q.accounts.GetActive(ctx)
	if err != nil {
		return dto.DashboardDTO{}, fmt.Errorf("loading account: %w", err)
	}
	positions, err := q.positions.ListOpen(ctx)
	if err != nil {
		return dto.DashboardDTO{}, fmt.Errorf("loading positions: %w", err)
	}
	openOrders, err := q.orders.ListOpen(ctx)
	if err != nil {
		return dto.DashboardDTO{}, fmt.Errorf("loading open orders: %w", err)
	}
	signals, err := q.signals.List(ctx, 20)
	if err != nil {
		return dto.DashboardDTO{}, fmt.Errorf("loading signals: %w", err)
	}
	decisions, err := q.decisions.List(ctx, 20)
	if err != nil {
		return dto.DashboardDTO{}, fmt.Errorf("loading risk decisions: %w", err)
	}
	activeEvents, err := q.events.ListActive(ctx)
	if err != nil {
		return dto.DashboardDTO{}, fmt.Errorf("loading active events: %w", err)
	}
	openIncidents, err := q.incidents.ListOpen(ctx)
	if err != nil {
		return dto.DashboardDTO{}, fmt.Errorf("loading open incidents: %w", err)
	}

	brokerHealthy := true
	if _, err := q.broker.GetAccount(ctx); err != nil {
		brokerHealthy = false
	}

	dashboard := dto.DashboardDTO{
		Account:             mapper.ToAccountDTO(acct),
		Positions:           mapToSlice(positions, mapper.ToPositionDTO),
		OpenOrders:          mapToSlice(openOrders, mapper.ToOrderDTO),
		RecentSignals:       mapToSlice(signals, mapper.ToSignalDTO),
		RecentRiskDecisions: mapToSlice(decisions, mapper.ToRiskDecisionDTO),
		ActiveEvents:        mapToSlice(activeEvents, mapper.ToMarketEventDTO),
		ActiveIncidents:     mapToSlice(openIncidents, mapper.ToIncidentDTO),
		Integrations: dto.IntegrationsHealthDTO{
			// Quant Engine and LLM adapters in this deployment are the
			// fake/noop implementations, which are always available by
			// construction; a future gRPC/Gemini-backed health probe would
			// replace these constants.
			BrokerHealthy:      brokerHealthy,
			QuantEngineHealthy: true,
			LLMHealthy:         true,
		},
		OperationalMode: string(acct.OperationalMode),
	}
	return dashboard, nil
}

func (q *QueryService) GetAccount(ctx context.Context) (dto.AccountDTO, error) {
	acct, err := q.accounts.GetActive(ctx)
	if err != nil {
		return dto.AccountDTO{}, err
	}
	return mapper.ToAccountDTO(acct), nil
}

func (q *QueryService) ListPositions(ctx context.Context) ([]dto.PositionDTO, error) {
	positions, err := q.positions.ListOpen(ctx)
	if err != nil {
		return nil, err
	}
	return mapToSlice(positions, mapper.ToPositionDTO), nil
}

func (q *QueryService) ListOrders(ctx context.Context, query query.ListOrdersQuery) ([]dto.OrderDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	orders, err := q.orders.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	return mapToSlice(orders, mapper.ToOrderDTO), nil
}

func (q *QueryService) GetOrderByID(ctx context.Context, query query.GetOrderByIDQuery) (dto.OrderDTO, error) {
	id, err := shared.ParseOrderID(query.OrderID)
	if err != nil {
		return dto.OrderDTO{}, err
	}
	o, err := q.orders.GetByID(ctx, id)
	if err != nil {
		return dto.OrderDTO{}, err
	}
	return mapper.ToOrderDTO(o), nil
}

func (q *QueryService) ListSignals(ctx context.Context, query query.ListSignalsQuery) ([]dto.SignalDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	signals, err := q.signals.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	return mapToSlice(signals, mapper.ToSignalDTO), nil
}

func (q *QueryService) ListRiskDecisions(ctx context.Context, query query.ListRiskDecisionsQuery) ([]dto.RiskDecisionDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	decisions, err := q.decisions.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	return mapToSlice(decisions, mapper.ToRiskDecisionDTO), nil
}

func (q *QueryService) ListEvents(ctx context.Context, query query.ListEventsQuery) ([]dto.MarketEventDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	events, err := q.events.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	return mapToSlice(events, mapper.ToMarketEventDTO), nil
}

func (q *QueryService) ListIncidents(ctx context.Context, query query.ListIncidentsQuery) ([]dto.IncidentDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	incidents, err := q.incidents.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	return mapToSlice(incidents, mapper.ToIncidentDTO), nil
}

// mapToSlice converts a slice of values using a mapper that takes a pointer.
func mapToSlice[T, R any](items []T, fn func(*T) R) []R {
	result := make([]R, 0, len(items))
	for i := range items {
		result = append(result, fn(&items[i]))
	}
	return result
}
