package risk

import (
	domainrisk "trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"
)

func toDomain(m model) (*domainrisk.Decision, error) {
	id, err := shared.ParseRiskDecisionID(m.ID)
	if err != nil {
		return nil, err
	}
	signalID, err := shared.ParseSignalID(m.SignalID)
	if err != nil {
		return nil, err
	}
	originalSize, err := shared.NewQuantity(m.OriginalPositionSize)
	if err != nil {
		return nil, err
	}
	approvedSize, err := shared.NewQuantity(m.ApprovedPositionSize)
	if err != nil {
		return nil, err
	}

	reasonCodes := make([]domainrisk.ReasonCode, 0, len(m.ReasonCodes))
	for _, rc := range m.ReasonCodes {
		reasonCodes = append(reasonCodes, domainrisk.ReasonCode(rc))
	}

	return &domainrisk.Decision{
		ID:                   id,
		SignalID:             signalID,
		Allowed:              m.Allowed,
		OriginalPositionSize: originalSize,
		ApprovedPositionSize: approvedSize,
		ReasonCodes:          reasonCodes,
		AppliedRules:         m.AppliedRules,
		EventRestrictions:    m.EventRestrictions,
		CreatedAt:            m.CreatedAt,
	}, nil
}

func toModel(d *domainrisk.Decision) model {
	reasonCodes := make([]string, 0, len(d.ReasonCodes))
	for _, rc := range d.ReasonCodes {
		reasonCodes = append(reasonCodes, string(rc))
	}
	return model{
		ID:                   d.ID.String(),
		SignalID:             d.SignalID.String(),
		Allowed:              d.Allowed,
		OriginalPositionSize: d.OriginalPositionSize.Decimal(),
		ApprovedPositionSize: d.ApprovedPositionSize.Decimal(),
		ReasonCodes:          reasonCodes,
		AppliedRules:         d.AppliedRules,
		EventRestrictions:    d.EventRestrictions,
		CreatedAt:            d.CreatedAt,
	}
}
