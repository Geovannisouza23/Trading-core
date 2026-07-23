package event

import (
	domainevent "trading-core/internal/domain/event"
	"trading-core/internal/domain/shared"
)

func toDomain(m model) (*domainevent.MarketEvent, error) {
	id, err := shared.ParseMarketEventID(m.ID)
	if err != nil {
		return nil, err
	}
	confidence, err := shared.NewConfidence(m.Confidence)
	if err != nil {
		return nil, err
	}
	assets := make([]shared.Symbol, 0, len(m.AffectedAssets))
	for _, raw := range m.AffectedAssets {
		symbol, err := shared.NewSymbol(raw)
		if err != nil {
			return nil, err
		}
		assets = append(assets, symbol)
	}

	return &domainevent.MarketEvent{
		ID:                id,
		EventType:         m.EventType,
		Direction:         domainevent.Direction(m.Direction),
		Severity:          domainevent.Severity(m.Severity),
		Confidence:        confidence,
		AffectedAssets:    assets,
		Action:            domainevent.Action(m.Action),
		SourceCount:       m.SourceCount,
		HasOfficialSource: m.HasOfficialSource,
		PublishedAt:       m.PublishedAt,
		DetectedAt:        m.DetectedAt,
		ExpiresAt:         m.ExpiresAt,
		Status:            domainevent.Status(m.Status),
	}, nil
}

func toModel(e *domainevent.MarketEvent) model {
	assets := make([]string, 0, len(e.AffectedAssets))
	for _, a := range e.AffectedAssets {
		assets = append(assets, a.String())
	}
	return model{
		ID:                e.ID.String(),
		EventType:         e.EventType,
		Direction:         string(e.Direction),
		Severity:          string(e.Severity),
		Confidence:        e.Confidence.Decimal(),
		AffectedAssets:    assets,
		Action:            string(e.Action),
		SourceCount:       e.SourceCount,
		HasOfficialSource: e.HasOfficialSource,
		PublishedAt:       e.PublishedAt,
		DetectedAt:        e.DetectedAt,
		ExpiresAt:         e.ExpiresAt,
		Status:            string(e.Status),
	}
}
