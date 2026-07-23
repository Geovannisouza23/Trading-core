package order

import (
	"github.com/shopspring/decimal"

	domainorder "trading-core/internal/domain/order"
	"trading-core/internal/domain/shared"
)

func toDomain(m model) (*domainorder.Order, error) {
	id, err := shared.ParseOrderID(m.ID)
	if err != nil {
		return nil, err
	}
	clientOrderID, err := shared.NewClientOrderID(m.ClientOrderID)
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
	filledQuantity, err := shared.NewQuantity(m.FilledQuantity)
	if err != nil {
		return nil, err
	}
	signalID, err := shared.ParseSignalID(m.SignalID)
	if err != nil {
		return nil, err
	}
	riskDecisionID, err := shared.ParseRiskDecisionID(m.RiskDecisionID)
	if err != nil {
		return nil, err
	}

	requestedPrice, err := optionalPrice(m.RequestedPrice)
	if err != nil {
		return nil, err
	}
	averagePrice, err := optionalPrice(m.AverageExecutionPrice)
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

	return &domainorder.Order{
		ID:                    id,
		ClientOrderID:         clientOrderID,
		BrokerOrderID:         m.BrokerOrderID,
		Symbol:                symbol,
		Side:                  shared.Side(m.Side),
		Type:                  domainorder.Type(m.Type),
		Quantity:              quantity,
		FilledQuantity:        filledQuantity,
		RequestedPrice:        requestedPrice,
		AverageExecutionPrice: averagePrice,
		StopPrice:             stopPrice,
		TargetPrice:           targetPrice,
		Status:                domainorder.Status(m.Status),
		StrategyName:          m.StrategyName,
		SignalID:              signalID,
		RiskDecisionID:        riskDecisionID,
		FailureReason:         m.FailureReason,
		Version:               m.Version,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}, nil
}

func toModel(o *domainorder.Order) model {
	return model{
		ID:                    o.ID.String(),
		ClientOrderID:         o.ClientOrderID.String(),
		BrokerOrderID:         o.BrokerOrderID,
		Symbol:                o.Symbol.String(),
		Side:                  string(o.Side),
		Type:                  string(o.Type),
		Quantity:              o.Quantity.Decimal(),
		FilledQuantity:        o.FilledQuantity.Decimal(),
		RequestedPrice:        priceDecimal(o.RequestedPrice),
		AverageExecutionPrice: priceDecimal(o.AverageExecutionPrice),
		StopPrice:             priceDecimal(o.StopPrice),
		TargetPrice:           priceDecimal(o.TargetPrice),
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
