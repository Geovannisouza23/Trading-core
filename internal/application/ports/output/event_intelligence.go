package output

import (
	"context"
	"time"

	"trading-core/internal/domain/event"
	"trading-core/internal/domain/shared"
)

// NewsAnalysisInput is a single sanitized news item submitted to the LLM for
// classification.
type NewsAnalysisInput struct {
	Source      string
	URL         string
	Title       string
	Content     string
	PublishedAt time.Time
}

// EventAssessment is the LLM's structured classification of a news item. It
// never contains a trading instruction beyond the deterministic Action
// enumeration; the LLM cannot place, size or approve orders.
type EventAssessment struct {
	EventType         string
	Direction         event.Direction
	Severity          event.Severity
	Confidence        shared.Confidence
	AffectedAssets    []shared.Symbol
	Action            event.Action
	SourceCount       int
	HasOfficialSource bool
}

// EventIntelligence is the port to the LLM-backed news/event classifier.
type EventIntelligence interface {
	Analyze(ctx context.Context, input NewsAnalysisInput) (EventAssessment, error)
}
