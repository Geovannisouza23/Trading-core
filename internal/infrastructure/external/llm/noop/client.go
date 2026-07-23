// Package noop implements output.EventIntelligence without calling any
// external service, so the system can run with zero LLM credentials. It is
// the default provider (see internal/config LLMConfig.Provider == "noop").
package noop

import (
	"context"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/event"
	"trading-core/internal/domain/shared"
)

// placeholderSymbol is used because the domain requires MarketEvent to name
// at least one affected asset (section 9.6), but a no-op classifier has no
// real answer. Combined with SeverityLow/ActionNormal, this placeholder
// never triggers NoCriticalEventSpecification for any real symbol.
var placeholderSymbol = shared.MustNewSymbol("GLOBAL")

// Client always classifies input as NORMAL/low severity.
type Client struct{}

func NewClient() *Client { return &Client{} }

var _ output.EventIntelligence = (*Client)(nil)

func (c *Client) Analyze(ctx context.Context, input output.NewsAnalysisInput) (output.EventAssessment, error) {
	return output.EventAssessment{
		EventType:         "UNCLASSIFIED",
		Direction:         event.DirectionNeutral,
		Severity:          event.SeverityLow,
		Confidence:        shared.MustNewConfidence(decimal.Zero),
		AffectedAssets:    []shared.Symbol{placeholderSymbol},
		Action:            event.ActionNormal,
		SourceCount:       1,
		HasOfficialSource: false,
	}, nil
}
