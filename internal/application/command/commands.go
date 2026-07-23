// Package command holds the write-intent inputs accepted by use cases.
package command

import (
	"time"

	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

// EvaluateMarketSignalCommand carries a freshly closed candle (plus recent
// history for context) into the signal evaluation pipeline.
type EvaluateMarketSignalCommand struct {
	Candle        market.Candle
	RecentCandles []market.Candle
}

// EvaluateRiskCommand triggers risk evaluation for an already-persisted
// signal. CurrentPrice is the latest known market price at evaluation time
// (typically the close of the candle that produced the signal), used to
// measure slippage against the signal's proposed entry price.
type EvaluateRiskCommand struct {
	SignalID     shared.SignalID
	CurrentPrice shared.Price
}

// ExecuteApprovedOrderCommand triggers order execution for an
// already-persisted, allowed risk decision.
type ExecuteApprovedOrderCommand struct {
	RiskDecisionID shared.RiskDecisionID
}

// ReconcileBrokerStateCommand triggers a reconciliation pass. Reason is
// informational (e.g. "scheduled", "manual", "post-kill-switch").
type ReconcileBrokerStateCommand struct {
	Reason string
}

// ProcessMarketEventCommand carries a raw news item into the event
// intelligence pipeline.
type ProcessMarketEventCommand struct {
	Source      string
	URL         string
	Title       string
	Content     string
	PublishedAt time.Time
}

// ActivateKillSwitchCommand forces the system into ModeKillSwitch.
type ActivateKillSwitchCommand struct {
	ActivatedBy string
	Origin      string
	Reason      string
	AllowClose  bool
}

// ChangeOperationalModeCommand requests a controlled operational mode
// transition. RealConfirmationToken is only inspected when TargetMode is
// REAL.
type ChangeOperationalModeCommand struct {
	TargetMode            string
	ActedBy               string
	Origin                string
	Reason                string
	RealConfirmationToken string
}
