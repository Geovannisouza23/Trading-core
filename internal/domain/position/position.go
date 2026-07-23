// Package position models the Position aggregate: the net exposure the
// system currently holds (or held) on a symbol.
package position

import (
	"time"

	"trading-core/internal/domain/shared"
)

type Status string

const (
	StatusOpen   Status = "OPEN"
	StatusClosed Status = "CLOSED"
)

// Position tracks net exposure on a single symbol, opened by one or more
// fills and closed when quantity returns to zero.
type Position struct {
	ID            shared.PositionID
	Symbol        shared.Symbol
	Side          shared.Side
	Quantity      shared.Quantity
	EntryPrice    shared.Price
	CurrentPrice  shared.Price
	StopPrice     *shared.Price
	TargetPrice   *shared.Price
	UnrealizedPnL shared.Money
	RealizedPnL   shared.Money
	Status        Status
	Version       int
	OpenedAt      time.Time
	ClosedAt      *time.Time
}

// Open creates a new open Position from an execution fill.
func Open(
	symbol shared.Symbol,
	side shared.Side,
	quantity shared.Quantity,
	entryPrice shared.Price,
	stopPrice, targetPrice *shared.Price,
	now time.Time,
) (*Position, error) {
	if !side.Valid() {
		return nil, shared.NewValidationError("side", "must be BUY or SELL")
	}
	if quantity.IsZero() {
		return nil, shared.NewValidationError("quantity", "must be greater than zero")
	}
	return &Position{
		ID:            shared.NewPositionID(),
		Symbol:        symbol,
		Side:          side,
		Quantity:      quantity,
		EntryPrice:    entryPrice,
		CurrentPrice:  entryPrice,
		StopPrice:     stopPrice,
		TargetPrice:   targetPrice,
		UnrealizedPnL: shared.ZeroMoney(),
		RealizedPnL:   shared.ZeroMoney(),
		Status:        StatusOpen,
		Version:       1,
		OpenedAt:      now,
	}, nil
}

// UnrealizedPnLAt computes unrealized P&L for a given mark price without
// mutating the position, given side*quantity*(mark-entry) semantics.
func (p *Position) UnrealizedPnLAt(mark shared.Price) shared.Money {
	diff := mark.Decimal().Sub(p.EntryPrice.Decimal())
	if p.Side == shared.SideSell {
		diff = diff.Neg()
	}
	return shared.NewMoney(diff.Mul(p.Quantity.Decimal()))
}

// MarkPrice updates CurrentPrice and recomputes UnrealizedPnL. It is a
// no-op on closed positions.
func (p *Position) MarkPrice(mark shared.Price, now time.Time) error {
	if p.Status == StatusClosed {
		return shared.NewConflictError("position", string(StatusClosed), string(StatusOpen))
	}
	p.CurrentPrice = mark
	p.UnrealizedPnL = p.UnrealizedPnLAt(mark)
	p.Version++
	_ = now // reserved for future audit trail of price marks
	return nil
}

// Reduce partially closes the position by `quantity` at `exitPrice`,
// crystallizing realized P&L proportionally. Reducing to zero closes the
// position.
func (p *Position) Reduce(quantity shared.Quantity, exitPrice shared.Price, now time.Time) error {
	if p.Status == StatusClosed {
		return shared.NewConflictError("position", string(StatusClosed), string(StatusOpen))
	}
	remaining, err := p.Quantity.Sub(quantity)
	if err != nil {
		return shared.NewValidationError("quantity", "must not exceed open position quantity")
	}
	realized := p.UnrealizedPnLAt(exitPrice).Decimal().Mul(quantity.Decimal()).Div(p.Quantity.Decimal())
	p.RealizedPnL = p.RealizedPnL.Add(shared.NewMoney(realized))
	p.Quantity = remaining
	p.Version++
	if remaining.IsZero() {
		p.Status = StatusClosed
		p.ClosedAt = &now
		p.UnrealizedPnL = shared.ZeroMoney()
	} else {
		p.CurrentPrice = exitPrice
		p.UnrealizedPnL = p.UnrealizedPnLAt(exitPrice)
	}
	return nil
}

// Close fully closes the position at exitPrice.
func (p *Position) Close(exitPrice shared.Price, now time.Time) error {
	return p.Reduce(p.Quantity, exitPrice, now)
}
