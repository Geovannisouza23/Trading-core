// Package risk implements output.RiskDecisionRepository against the
// risk_decisions table.
package risk

import (
	"time"

	"github.com/shopspring/decimal"
)

type model struct {
	ID                   string
	SignalID             string
	Allowed              bool
	OriginalPositionSize decimal.Decimal
	ApprovedPositionSize decimal.Decimal
	ReasonCodes          []string
	AppliedRules         []string
	EventRestrictions    []string
	CreatedAt            time.Time
}
