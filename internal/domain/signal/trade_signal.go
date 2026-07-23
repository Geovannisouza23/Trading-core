// Package signal models the TradeSignal aggregate: a candidate trade
// produced by the Quant Engine evaluation pipeline, prior to any risk
// review.
package signal

import (
	"time"

	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
)

// TradeSignal is a candidate trade idea awaiting risk evaluation.
type TradeSignal struct {
	ID           shared.SignalID
	Symbol       shared.Symbol
	Side         shared.Side
	EntryPrice   shared.Price
	StopPrice    shared.Price
	TargetPrice  shared.Price
	Confidence   shared.Confidence
	StrategyName string
	MarketRegime market.Regime
	CreatedAt    time.Time
	ValidUntil   time.Time
}

// New validates invariants and builds a TradeSignal.
//
// Stop/target placement must be coherent with the side: a BUY signal must
// have its stop below entry and its target above entry; a SELL signal is
// the mirror image.
func New(
	symbol shared.Symbol,
	side shared.Side,
	entryPrice, stopPrice, targetPrice shared.Price,
	confidence shared.Confidence,
	strategyName string,
	regime market.Regime,
	createdAt, validUntil time.Time,
) (*TradeSignal, error) {
	if !side.Valid() {
		return nil, shared.NewValidationError("side", "must be BUY or SELL")
	}
	if strategyName == "" {
		return nil, shared.NewValidationError("strategy_name", "must not be empty")
	}
	if !regime.Valid() {
		return nil, shared.NewValidationError("market_regime", "unknown regime")
	}
	if !validUntil.After(createdAt) {
		return nil, shared.NewValidationError("valid_until", "must be after created_at")
	}

	switch side {
	case shared.SideBuy:
		if stopPrice.GreaterThanOrEqual(entryPrice) {
			return nil, shared.NewValidationError("stop_price", "must be below entry price for a BUY signal")
		}
		if targetPrice.LessThan(entryPrice) || targetPrice.Equal(entryPrice) {
			return nil, shared.NewValidationError("target_price", "must be above entry price for a BUY signal")
		}
	case shared.SideSell:
		if stopPrice.LessThan(entryPrice) || stopPrice.Equal(entryPrice) {
			return nil, shared.NewValidationError("stop_price", "must be above entry price for a SELL signal")
		}
		if targetPrice.GreaterThanOrEqual(entryPrice) {
			return nil, shared.NewValidationError("target_price", "must be below entry price for a SELL signal")
		}
	}

	return &TradeSignal{
		ID:           shared.NewSignalID(),
		Symbol:       symbol,
		Side:         side,
		EntryPrice:   entryPrice,
		StopPrice:    stopPrice,
		TargetPrice:  targetPrice,
		Confidence:   confidence,
		StrategyName: strategyName,
		MarketRegime: regime,
		CreatedAt:    createdAt,
		ValidUntil:   validUntil,
	}, nil
}

// IsExpired reports whether the signal is no longer actionable at `now`.
func (s *TradeSignal) IsExpired(now time.Time) bool {
	return now.After(s.ValidUntil)
}

// StopDistance returns the absolute price distance between entry and stop,
// used by position sizing.
func (s *TradeSignal) StopDistance() shared.Price {
	return shared.MustNewPrice(s.EntryPrice.DistanceTo(s.StopPrice))
}

// SignalCreated is published once a TradeSignal has been persisted.
type SignalCreated struct {
	Signal      *TradeSignal
	OccurredAt_ time.Time
}

func (e SignalCreated) EventName() string     { return "SignalCreated" }
func (e SignalCreated) OccurredAt() time.Time { return e.OccurredAt_ }
