package signal

import (
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
	domainsignal "trading-core/internal/domain/signal"
)

func toDomain(m model) (*domainsignal.TradeSignal, error) {
	id, err := shared.ParseSignalID(m.ID)
	if err != nil {
		return nil, err
	}
	symbol, err := shared.NewSymbol(m.Symbol)
	if err != nil {
		return nil, err
	}
	entryPrice, err := shared.NewPrice(m.EntryPrice)
	if err != nil {
		return nil, err
	}
	stopPrice, err := shared.NewPrice(m.StopPrice)
	if err != nil {
		return nil, err
	}
	targetPrice, err := shared.NewPrice(m.TargetPrice)
	if err != nil {
		return nil, err
	}
	confidence, err := shared.NewConfidence(m.Confidence)
	if err != nil {
		return nil, err
	}
	return &domainsignal.TradeSignal{
		ID:           id,
		Symbol:       symbol,
		Side:         shared.Side(m.Side),
		EntryPrice:   entryPrice,
		StopPrice:    stopPrice,
		TargetPrice:  targetPrice,
		Confidence:   confidence,
		StrategyName: m.StrategyName,
		MarketRegime: market.Regime(m.MarketRegime),
		CreatedAt:    m.CreatedAt,
		ValidUntil:   m.ValidUntil,
	}, nil
}

func toModel(s *domainsignal.TradeSignal) model {
	return model{
		ID:           s.ID.String(),
		Symbol:       s.Symbol.String(),
		Side:         string(s.Side),
		EntryPrice:   s.EntryPrice.Decimal(),
		StopPrice:    s.StopPrice.Decimal(),
		TargetPrice:  s.TargetPrice.Decimal(),
		Confidence:   s.Confidence.Decimal(),
		StrategyName: s.StrategyName,
		MarketRegime: string(s.MarketRegime),
		CreatedAt:    s.CreatedAt,
		ValidUntil:   s.ValidUntil,
	}
}
