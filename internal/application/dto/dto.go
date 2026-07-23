// Package dto holds plain, JSON-friendly read models returned by use cases.
// Interfaces-layer presenters serialize these directly; they never see a
// domain aggregate.
package dto

import "time"

type AccountDTO struct {
	ID               string    `json:"id"`
	Balance          string    `json:"balance"`
	Equity           string    `json:"equity"`
	AvailableBalance string    `json:"available_balance"`
	PeakEquity       string    `json:"peak_equity"`
	DailyPnL         string    `json:"daily_pnl"`
	WeeklyPnL        string    `json:"weekly_pnl"`
	CurrentDrawdown  string    `json:"current_drawdown"`
	OperationalMode  string    `json:"operational_mode"`
	Version          int       `json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PositionDTO struct {
	ID            string     `json:"id"`
	Symbol        string     `json:"symbol"`
	Side          string     `json:"side"`
	Quantity      string     `json:"quantity"`
	EntryPrice    string     `json:"entry_price"`
	CurrentPrice  string     `json:"current_price"`
	StopPrice     *string    `json:"stop_price,omitempty"`
	TargetPrice   *string    `json:"target_price,omitempty"`
	UnrealizedPnL string     `json:"unrealized_pnl"`
	RealizedPnL   string     `json:"realized_pnl"`
	Status        string     `json:"status"`
	Version       int        `json:"version"`
	OpenedAt      time.Time  `json:"opened_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
}

type OrderDTO struct {
	ID                    string    `json:"id"`
	ClientOrderID         string    `json:"client_order_id"`
	BrokerOrderID         string    `json:"broker_order_id,omitempty"`
	Symbol                string    `json:"symbol"`
	Side                  string    `json:"side"`
	Type                  string    `json:"type"`
	Quantity              string    `json:"quantity"`
	FilledQuantity        string    `json:"filled_quantity"`
	RequestedPrice        *string   `json:"requested_price,omitempty"`
	AverageExecutionPrice *string   `json:"average_execution_price,omitempty"`
	StopPrice             *string   `json:"stop_price,omitempty"`
	TargetPrice           *string   `json:"target_price,omitempty"`
	Status                string    `json:"status"`
	StrategyName          string    `json:"strategy_name"`
	SignalID              string    `json:"signal_id"`
	RiskDecisionID        string    `json:"risk_decision_id"`
	FailureReason         string    `json:"failure_reason,omitempty"`
	Version               int       `json:"version"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type SignalDTO struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	EntryPrice   string    `json:"entry_price"`
	StopPrice    string    `json:"stop_price"`
	TargetPrice  string    `json:"target_price"`
	Confidence   string    `json:"confidence"`
	StrategyName string    `json:"strategy_name"`
	MarketRegime string    `json:"market_regime"`
	CreatedAt    time.Time `json:"created_at"`
	ValidUntil   time.Time `json:"valid_until"`
}

type RiskDecisionDTO struct {
	ID                   string    `json:"id"`
	SignalID             string    `json:"signal_id"`
	Allowed              bool      `json:"allowed"`
	OriginalPositionSize string    `json:"original_position_size"`
	ApprovedPositionSize string    `json:"approved_position_size"`
	ReasonCodes          []string  `json:"reason_codes"`
	AppliedRules         []string  `json:"applied_rules"`
	EventRestrictions    []string  `json:"event_restrictions"`
	CreatedAt            time.Time `json:"created_at"`
}

type MarketEventDTO struct {
	ID                string    `json:"id"`
	EventType         string    `json:"event_type"`
	Direction         string    `json:"direction"`
	Severity          string    `json:"severity"`
	Confidence        string    `json:"confidence"`
	AffectedAssets    []string  `json:"affected_assets"`
	Action            string    `json:"action"`
	SourceCount       int       `json:"source_count"`
	HasOfficialSource bool      `json:"has_official_source"`
	PublishedAt       time.Time `json:"published_at"`
	DetectedAt        time.Time `json:"detected_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	Status            string    `json:"status"`
}

type IncidentDTO struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"`
	Severity        string     `json:"severity"`
	Description     string     `json:"description"`
	Source          string     `json:"source"`
	RelatedEntityID string     `json:"related_entity_id,omitempty"`
	Status          string     `json:"status"`
	DetectedAt      time.Time  `json:"detected_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	Resolution      string     `json:"resolution,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type IntegrationsHealthDTO struct {
	BrokerHealthy      bool `json:"broker_healthy"`
	QuantEngineHealthy bool `json:"quant_engine_healthy"`
	LLMHealthy         bool `json:"llm_healthy"`
}

type DashboardDTO struct {
	Account              AccountDTO            `json:"account"`
	Positions            []PositionDTO         `json:"positions"`
	OpenOrders           []OrderDTO            `json:"open_orders"`
	RecentSignals        []SignalDTO           `json:"recent_signals"`
	RecentRiskDecisions  []RiskDecisionDTO     `json:"recent_risk_decisions"`
	ActiveEvents         []MarketEventDTO      `json:"active_events"`
	ActiveIncidents      []IncidentDTO         `json:"active_incidents"`
	Integrations         IntegrationsHealthDTO `json:"integrations"`
	OperationalMode      string                `json:"operational_mode"`
	LastReconciliationAt *time.Time            `json:"last_reconciliation_at,omitempty"`
}
