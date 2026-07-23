// Package mapper converts domain aggregates into application/dto read
// models. This is the only place that translates internal domain types into
// the shape exposed outward.
package mapper

import (
	"trading-core/internal/application/dto"
	"trading-core/internal/domain/account"
	"trading-core/internal/domain/event"
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/order"
	"trading-core/internal/domain/position"
	"trading-core/internal/domain/risk"
	"trading-core/internal/domain/signal"
)

func ToAccountDTO(a *account.Account) dto.AccountDTO {
	return dto.AccountDTO{
		ID:               a.ID.String(),
		Balance:          a.Balance.String(),
		Equity:           a.Equity.String(),
		AvailableBalance: a.AvailableBalance.String(),
		PeakEquity:       a.PeakEquity.String(),
		DailyPnL:         a.DailyPnL.String(),
		WeeklyPnL:        a.WeeklyPnL.String(),
		CurrentDrawdown:  a.CurrentDrawdown.String(),
		OperationalMode:  string(a.OperationalMode),
		Version:          a.Version,
		UpdatedAt:        a.UpdatedAt,
	}
}

func ToPositionDTO(p *position.Position) dto.PositionDTO {
	var stop, target *string
	if p.StopPrice != nil {
		s := p.StopPrice.String()
		stop = &s
	}
	if p.TargetPrice != nil {
		t := p.TargetPrice.String()
		target = &t
	}
	return dto.PositionDTO{
		ID:            p.ID.String(),
		Symbol:        p.Symbol.String(),
		Side:          p.Side.String(),
		Quantity:      p.Quantity.String(),
		EntryPrice:    p.EntryPrice.String(),
		CurrentPrice:  p.CurrentPrice.String(),
		StopPrice:     stop,
		TargetPrice:   target,
		UnrealizedPnL: p.UnrealizedPnL.String(),
		RealizedPnL:   p.RealizedPnL.String(),
		Status:        string(p.Status),
		Version:       p.Version,
		OpenedAt:      p.OpenedAt,
		ClosedAt:      p.ClosedAt,
	}
}

func ToOrderDTO(o *order.Order) dto.OrderDTO {
	var requested, avg, stop, target *string
	if o.RequestedPrice != nil {
		v := o.RequestedPrice.String()
		requested = &v
	}
	if o.AverageExecutionPrice != nil {
		v := o.AverageExecutionPrice.String()
		avg = &v
	}
	if o.StopPrice != nil {
		v := o.StopPrice.String()
		stop = &v
	}
	if o.TargetPrice != nil {
		v := o.TargetPrice.String()
		target = &v
	}
	return dto.OrderDTO{
		ID:                    o.ID.String(),
		ClientOrderID:         o.ClientOrderID.String(),
		BrokerOrderID:         o.BrokerOrderID,
		Symbol:                o.Symbol.String(),
		Side:                  o.Side.String(),
		Type:                  string(o.Type),
		Quantity:              o.Quantity.String(),
		FilledQuantity:        o.FilledQuantity.String(),
		RequestedPrice:        requested,
		AverageExecutionPrice: avg,
		StopPrice:             stop,
		TargetPrice:           target,
		Status:                string(o.Status),
		StrategyName:          o.StrategyName,
		SignalID:              o.SignalID.String(),
		RiskDecisionID:        o.RiskDecisionID.String(),
		FailureReason:         o.FailureReason,
		Version:               o.Version,
		CreatedAt:             o.CreatedAt,
		UpdatedAt:             o.UpdatedAt,
	}
}

func ToSignalDTO(s *signal.TradeSignal) dto.SignalDTO {
	return dto.SignalDTO{
		ID:           s.ID.String(),
		Symbol:       s.Symbol.String(),
		Side:         s.Side.String(),
		EntryPrice:   s.EntryPrice.String(),
		StopPrice:    s.StopPrice.String(),
		TargetPrice:  s.TargetPrice.String(),
		Confidence:   s.Confidence.String(),
		StrategyName: s.StrategyName,
		MarketRegime: string(s.MarketRegime),
		CreatedAt:    s.CreatedAt,
		ValidUntil:   s.ValidUntil,
	}
}

func ToRiskDecisionDTO(d *risk.Decision) dto.RiskDecisionDTO {
	reasonCodes := make([]string, 0, len(d.ReasonCodes))
	for _, rc := range d.ReasonCodes {
		reasonCodes = append(reasonCodes, string(rc))
	}
	return dto.RiskDecisionDTO{
		ID:                   d.ID.String(),
		SignalID:             d.SignalID.String(),
		Allowed:              d.Allowed,
		OriginalPositionSize: d.OriginalPositionSize.String(),
		ApprovedPositionSize: d.ApprovedPositionSize.String(),
		ReasonCodes:          reasonCodes,
		AppliedRules:         d.AppliedRules,
		EventRestrictions:    d.EventRestrictions,
		CreatedAt:            d.CreatedAt,
	}
}

func ToMarketEventDTO(e *event.MarketEvent) dto.MarketEventDTO {
	assets := make([]string, 0, len(e.AffectedAssets))
	for _, a := range e.AffectedAssets {
		assets = append(assets, a.String())
	}
	return dto.MarketEventDTO{
		ID:                e.ID.String(),
		EventType:         e.EventType,
		Direction:         string(e.Direction),
		Severity:          string(e.Severity),
		Confidence:        e.Confidence.String(),
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

func ToIncidentDTO(i *operation.Incident) dto.IncidentDTO {
	return dto.IncidentDTO{
		ID:              i.ID.String(),
		Type:            string(i.Type),
		Severity:        string(i.Severity),
		Description:     i.Description,
		Source:          i.Source,
		RelatedEntityID: i.RelatedEntityID,
		Status:          string(i.Status),
		DetectedAt:      i.DetectedAt,
		ResolvedAt:      i.ResolvedAt,
		Resolution:      i.Resolution,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       i.UpdatedAt,
	}
}
