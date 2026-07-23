package risk

import (
	"github.com/shopspring/decimal"

	"trading-core/internal/domain/shared"
)

// SizingInput carries every fact position sizing needs, expressed purely in
// domain value objects (no broker- or config-specific types).
type SizingInput struct {
	AvailableBalance shared.Money
	RiskPerTrade     shared.Percentage
	EntryPrice       shared.Price
	StopPrice        shared.Price
	MinQuantity      shared.Quantity
	StepSize         decimal.Decimal
	MinNotional      shared.Money
	MaxLeverage      shared.Leverage
	ExistingExposure shared.Money
	ATR              *decimal.Decimal
}

// SizingResult is either an approved quantity or a rejection reason (never
// both). A rejection is not an error: it is a valid, deterministic outcome
// that EvaluateRisk turns into a blocked RiskDecision.
type SizingResult struct {
	Quantity shared.Quantity
	Rejected bool
	Reason   string
}

// PositionSizer is the Strategy pattern port for turning risk inputs into a
// concrete order quantity.
type PositionSizer interface {
	Name() string
	Size(input SizingInput) SizingResult
}

func roundDownToStep(raw, step decimal.Decimal) decimal.Decimal {
	if step.IsZero() {
		return raw
	}
	steps := raw.Div(step).Floor()
	return steps.Mul(step)
}

func capByLeverage(rawQuantity decimal.Decimal, input SizingInput) decimal.Decimal {
	maxNotional := input.AvailableBalance.Decimal().Mul(input.MaxLeverage.Decimal())
	if input.EntryPrice.Decimal().IsZero() {
		return rawQuantity
	}
	maxQuantityByLeverage := maxNotional.Div(input.EntryPrice.Decimal())
	if rawQuantity.GreaterThan(maxQuantityByLeverage) {
		return maxQuantityByLeverage
	}
	return rawQuantity
}

func finalizeQuantity(rawQuantity decimal.Decimal, input SizingInput) SizingResult {
	stepped := roundDownToStep(rawQuantity, input.StepSize)
	quantity, err := shared.NewQuantity(stepped)
	if err != nil || quantity.IsZero() {
		return SizingResult{Rejected: true, Reason: "quantity rounds down to zero"}
	}
	if quantity.LessThan(input.MinQuantity) {
		return SizingResult{Rejected: true, Reason: "below broker minimum quantity"}
	}
	notional := shared.NewMoney(quantity.Decimal().Mul(input.EntryPrice.Decimal()))
	if notional.LessThan(input.MinNotional) {
		return SizingResult{Rejected: true, Reason: "below broker minimum notional"}
	}
	return SizingResult{Quantity: quantity}
}

// FixedRiskPositionSizing sizes the position so that a full stop-out loses
// exactly RiskPerTrade of available balance.
type FixedRiskPositionSizing struct{}

func (FixedRiskPositionSizing) Name() string { return "FixedRiskPositionSizing" }

func (FixedRiskPositionSizing) Size(input SizingInput) SizingResult {
	if !input.AvailableBalance.IsPositive() {
		return SizingResult{Rejected: true, Reason: "insufficient balance"}
	}
	stopDistance := input.EntryPrice.DistanceTo(input.StopPrice)
	if stopDistance.IsZero() {
		return SizingResult{Rejected: true, Reason: "zero stop distance"}
	}
	riskAmount := input.RiskPerTrade.Of(input.AvailableBalance)
	rawQuantity := riskAmount.Decimal().Div(stopDistance)
	rawQuantity = capByLeverage(rawQuantity, input)
	return finalizeQuantity(rawQuantity, input)
}

// ATRPositionSizing behaves like FixedRiskPositionSizing but derives the
// effective stop distance from ATR * multiplier when an ATR value is
// supplied, falling back to the signal's own stop distance otherwise.
type ATRPositionSizing struct {
	Multiplier decimal.Decimal
}

func (ATRPositionSizing) Name() string { return "ATRPositionSizing" }

func (s ATRPositionSizing) Size(input SizingInput) SizingResult {
	if !input.AvailableBalance.IsPositive() {
		return SizingResult{Rejected: true, Reason: "insufficient balance"}
	}
	stopDistance := input.EntryPrice.DistanceTo(input.StopPrice)
	if input.ATR != nil && !input.ATR.IsZero() {
		multiplier := s.Multiplier
		if multiplier.IsZero() {
			multiplier = decimal.NewFromFloat(1.5)
		}
		stopDistance = input.ATR.Mul(multiplier)
	}
	if stopDistance.IsZero() {
		return SizingResult{Rejected: true, Reason: "zero stop distance"}
	}
	riskAmount := input.RiskPerTrade.Of(input.AvailableBalance)
	rawQuantity := riskAmount.Decimal().Div(stopDistance)
	rawQuantity = capByLeverage(rawQuantity, input)
	return finalizeQuantity(rawQuantity, input)
}
