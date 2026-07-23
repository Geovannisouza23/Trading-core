package position

import (
	"github.com/shopspring/decimal"

	domainposition "trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
)

func toDomain(m model) (*domainposition.Position, error) {
	id, err := shared.ParsePositionID(m.ID)
	if err != nil {
		return nil, err
	}
	symbol, err := shared.NewSymbol(m.Symbol)
	if err != nil {
		return nil, err
	}
	quantity, err := shared.NewQuantity(m.Quantity)
	if err != nil {
		return nil, err
	}
	entryPrice, err := shared.NewPrice(m.EntryPrice)
	if err != nil {
		return nil, err
	}
	currentPrice, err := shared.NewPrice(m.CurrentPrice)
	if err != nil {
		return nil, err
	}
	stopPrice, err := optionalPrice(m.StopPrice)
	if err != nil {
		return nil, err
	}
	targetPrice, err := optionalPrice(m.TargetPrice)
	if err != nil {
		return nil, err
	}

	return &domainposition.Position{
		ID:            id,
		Symbol:        symbol,
		Side:          shared.Side(m.Side),
		Quantity:      quantity,
		EntryPrice:    entryPrice,
		CurrentPrice:  currentPrice,
		StopPrice:     stopPrice,
		TargetPrice:   targetPrice,
		UnrealizedPnL: shared.NewMoney(m.UnrealizedPnL),
		RealizedPnL:   shared.NewMoney(m.RealizedPnL),
		Status:        domainposition.Status(m.Status),
		Version:       m.Version,
		OpenedAt:      m.OpenedAt,
		ClosedAt:      m.ClosedAt,
	}, nil
}

func toModel(p *domainposition.Position) model {
	return model{
		ID:            p.ID.String(),
		Symbol:        p.Symbol.String(),
		Side:          string(p.Side),
		Quantity:      p.Quantity.Decimal(),
		EntryPrice:    p.EntryPrice.Decimal(),
		CurrentPrice:  p.CurrentPrice.Decimal(),
		StopPrice:     priceDecimal(p.StopPrice),
		TargetPrice:   priceDecimal(p.TargetPrice),
		UnrealizedPnL: p.UnrealizedPnL.Decimal(),
		RealizedPnL:   p.RealizedPnL.Decimal(),
		Status:        string(p.Status),
		Version:       p.Version,
		OpenedAt:      p.OpenedAt,
		ClosedAt:      p.ClosedAt,
	}
}

func optionalPrice(d *decimal.Decimal) (*shared.Price, error) {
	if d == nil {
		return nil, nil
	}
	p, err := shared.NewPrice(*d)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func priceDecimal(p *shared.Price) *decimal.Decimal {
	if p == nil {
		return nil
	}
	d := p.Decimal()
	return &d
}
