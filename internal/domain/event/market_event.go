// Package event models MarketEvent: a classified, actionable news/event
// assessment that risk evaluation consults before approving new entries.
package event

import (
	"time"

	"trading-core/internal/domain/shared"
)

type Direction string

const (
	DirectionBullish Direction = "BULLISH"
	DirectionBearish Direction = "BEARISH"
	DirectionNeutral Direction = "NEUTRAL"
)

type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// Action is the deterministic instruction derived from the event
// assessment. It never authorizes an order by itself — it only restricts
// what the risk manager is allowed to approve.
type Action string

const (
	ActionNormal          Action = "NORMAL"
	ActionReduce          Action = "REDUCE"
	ActionBlockNewEntries Action = "BLOCK_NEW_ENTRIES"
	ActionReviewRequired  Action = "REVIEW_REQUIRED"
	ActionKillSwitch      Action = "KILL_SWITCH"
)

func (a Action) Valid() bool {
	switch a {
	case ActionNormal, ActionReduce, ActionBlockNewEntries, ActionReviewRequired, ActionKillSwitch:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusExpired  Status = "EXPIRED"
	StatusResolved Status = "RESOLVED"
)

// MarketEvent is a classified news/event assessment affecting one or more
// symbols.
type MarketEvent struct {
	ID                shared.MarketEventID
	EventType         string
	Direction         Direction
	Severity          Severity
	Confidence        shared.Confidence
	AffectedAssets    []shared.Symbol
	Action            Action
	SourceCount       int
	HasOfficialSource bool
	PublishedAt       time.Time
	DetectedAt        time.Time
	ExpiresAt         time.Time
	Status            Status
}

// New validates and builds a MarketEvent in StatusActive.
func New(
	eventType string,
	direction Direction,
	severity Severity,
	confidence shared.Confidence,
	affectedAssets []shared.Symbol,
	action Action,
	sourceCount int,
	hasOfficialSource bool,
	publishedAt, detectedAt, expiresAt time.Time,
) (*MarketEvent, error) {
	if eventType == "" {
		return nil, shared.NewValidationError("event_type", "must not be empty")
	}
	if !action.Valid() {
		return nil, shared.NewValidationError("action", "unknown action")
	}
	if len(affectedAssets) == 0 {
		return nil, shared.NewValidationError("affected_assets", "must not be empty")
	}
	if sourceCount < 1 {
		return nil, shared.NewValidationError("source_count", "must be at least 1")
	}
	if !expiresAt.After(detectedAt) {
		return nil, shared.NewValidationError("expires_at", "must be after detected_at")
	}
	return &MarketEvent{
		ID:                shared.NewMarketEventID(),
		EventType:         eventType,
		Direction:         direction,
		Severity:          severity,
		Confidence:        confidence,
		AffectedAssets:    affectedAssets,
		Action:            action,
		SourceCount:       sourceCount,
		HasOfficialSource: hasOfficialSource,
		PublishedAt:       publishedAt,
		DetectedAt:        detectedAt,
		ExpiresAt:         expiresAt,
		Status:            StatusActive,
	}, nil
}

// IsExpired reports whether the event should no longer restrict trading.
func (e *MarketEvent) IsExpired(now time.Time) bool {
	return now.After(e.ExpiresAt)
}

// Expire transitions the event out of ACTIVE once it ages out.
func (e *MarketEvent) Expire() {
	if e.Status == StatusActive {
		e.Status = StatusExpired
	}
}

// AffectsSymbol reports whether the event restricts the given symbol.
func (e *MarketEvent) AffectsSymbol(symbol shared.Symbol) bool {
	for _, s := range e.AffectedAssets {
		if s.Equal(symbol) {
			return true
		}
	}
	return false
}

// IsCritical reports whether the event severity/action combination should be
// treated as a hard block by risk specifications.
func (e *MarketEvent) IsCritical() bool {
	return e.Severity == SeverityCritical || e.Action == ActionBlockNewEntries || e.Action == ActionKillSwitch
}

type CriticalEventDetected struct {
	Event       *MarketEvent
	OccurredAt_ time.Time
}

func (ev CriticalEventDetected) EventName() string     { return "CriticalEventDetected" }
func (ev CriticalEventDetected) OccurredAt() time.Time { return ev.OccurredAt_ }
