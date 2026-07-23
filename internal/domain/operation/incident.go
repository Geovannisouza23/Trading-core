package operation

import (
	"time"

	"trading-core/internal/domain/shared"
)

type IncidentType string

const (
	IncidentTypeReconciliationDivergence IncidentType = "RECONCILIATION_DIVERGENCE"
	IncidentTypeBrokerUnavailable        IncidentType = "BROKER_UNAVAILABLE"
	IncidentTypeQuantEngineUnavailable   IncidentType = "QUANT_ENGINE_UNAVAILABLE"
	IncidentTypeLLMUnavailable           IncidentType = "LLM_UNAVAILABLE"
	IncidentTypeUnknownOrderState        IncidentType = "UNKNOWN_ORDER_STATE"
	IncidentTypeKillSwitchActivated      IncidentType = "KILL_SWITCH_ACTIVATED"
	IncidentTypeCriticalMarketEvent      IncidentType = "CRITICAL_MARKET_EVENT"
)

type IncidentSeverity string

const (
	IncidentSeverityLow      IncidentSeverity = "LOW"
	IncidentSeverityMedium   IncidentSeverity = "MEDIUM"
	IncidentSeverityHigh     IncidentSeverity = "HIGH"
	IncidentSeverityCritical IncidentSeverity = "CRITICAL"
)

type IncidentStatus string

const (
	IncidentStatusOpen     IncidentStatus = "OPEN"
	IncidentStatusResolved IncidentStatus = "RESOLVED"
)

// Incident records an operational anomaly detected by the system
// (reconciliation divergence, broker/engine unavailability, etc.).
type Incident struct {
	ID              shared.IncidentID
	Type            IncidentType
	Severity        IncidentSeverity
	Description     string
	Source          string
	RelatedEntityID string
	Status          IncidentStatus
	DetectedAt      time.Time
	ResolvedAt      *time.Time
	Resolution      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewIncident opens a new incident. Incidents are always created OPEN.
func NewIncident(incidentType IncidentType, severity IncidentSeverity, description, source, relatedEntityID string, now time.Time) (*Incident, error) {
	if description == "" {
		return nil, shared.NewValidationError("description", "must not be empty")
	}
	if source == "" {
		return nil, shared.NewValidationError("source", "must not be empty")
	}
	return &Incident{
		ID:              shared.NewIncidentID(),
		Type:            incidentType,
		Severity:        severity,
		Description:     description,
		Source:          source,
		RelatedEntityID: relatedEntityID,
		Status:          IncidentStatusOpen,
		DetectedAt:      now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Resolve closes an open incident with a resolution note. Resolving an
// already-resolved incident is a no-op error to prevent double bookkeeping.
func (i *Incident) Resolve(resolution string, now time.Time) error {
	if i.Status == IncidentStatusResolved {
		return shared.NewConflictError("incident", string(IncidentStatusResolved), string(IncidentStatusResolved))
	}
	if resolution == "" {
		return shared.NewValidationError("resolution", "must not be empty")
	}
	i.Status = IncidentStatusResolved
	i.Resolution = resolution
	i.ResolvedAt = &now
	i.UpdatedAt = now
	return nil
}
