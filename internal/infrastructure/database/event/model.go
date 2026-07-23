// Package event implements output.MarketEventRepository against the
// market_events table.
package event

import (
	"time"

	"github.com/shopspring/decimal"
)

type model struct {
	ID                string
	EventType         string
	Direction         string
	Severity          string
	Confidence        decimal.Decimal
	AffectedAssets    []string
	Action            string
	SourceCount       int
	HasOfficialSource bool
	PublishedAt       time.Time
	DetectedAt        time.Time
	ExpiresAt         time.Time
	Status            string
}
