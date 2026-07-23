package risk

import (
	"time"

	"trading-core/internal/domain/shared"
)

// Decision is the immutable record of a single risk evaluation outcome.
type Decision struct {
	ID                   shared.RiskDecisionID
	SignalID             shared.SignalID
	Allowed              bool
	OriginalPositionSize shared.Quantity
	ApprovedPositionSize shared.Quantity
	ReasonCodes          []ReasonCode
	AppliedRules         []string
	EventRestrictions    []string
	CreatedAt            time.Time
}

// NewDecision builds a RiskDecision from a policy Evaluation and a sizing
// outcome. When either the policy rejects the signal or sizing rejects the
// resulting quantity, the decision is not allowed and the approved size is
// zero.
func NewDecision(
	signalID shared.SignalID,
	evaluation Evaluation,
	originalSize shared.Quantity,
	sizing SizingResult,
	eventRestrictions []string,
	now time.Time,
) *Decision {
	allowed := evaluation.Allowed && !sizing.Rejected
	approvedSize := shared.ZeroQuantity()
	reasonCodes := evaluation.ReasonCodes
	if sizing.Rejected {
		reasonCodes = append(reasonCodes, ReasonInsufficientBalance)
	}
	if allowed {
		approvedSize = sizing.Quantity
	}
	return &Decision{
		ID:                   shared.NewRiskDecisionID(),
		SignalID:             signalID,
		Allowed:              allowed,
		OriginalPositionSize: originalSize,
		ApprovedPositionSize: approvedSize,
		ReasonCodes:          reasonCodes,
		AppliedRules:         evaluation.AppliedRules,
		EventRestrictions:    eventRestrictions,
		CreatedAt:            now,
	}
}

type RiskDecisionCreated struct {
	Decision    *Decision
	OccurredAt_ time.Time
}

func (e RiskDecisionCreated) EventName() string     { return "RiskDecisionCreated" }
func (e RiskDecisionCreated) OccurredAt() time.Time { return e.OccurredAt_ }
