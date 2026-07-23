package order

import (
	"time"

	"trading-core/internal/domain/shared"
)

// Order is the only aggregate the execution use case is allowed to send to
// a broker. It owns its own state transitions; no other package may mutate
// Status directly.
type Order struct {
	ID                    shared.OrderID
	ClientOrderID         shared.ClientOrderID
	BrokerOrderID         string
	Symbol                shared.Symbol
	Side                  shared.Side
	Type                  Type
	Quantity              shared.Quantity
	FilledQuantity        shared.Quantity
	RequestedPrice        *shared.Price
	AverageExecutionPrice *shared.Price
	StopPrice             *shared.Price
	TargetPrice           *shared.Price
	Status                Status
	StrategyName          string
	SignalID              shared.SignalID
	RiskDecisionID        shared.RiskDecisionID
	FailureReason         string
	Version               int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// NewPendingOrder builds a new Order in StatusPending. It never touches a
// broker; ExecuteApprovedOrder is responsible for submission after this
// constructor succeeds.
func NewPendingOrder(
	clientOrderID shared.ClientOrderID,
	symbol shared.Symbol,
	side shared.Side,
	orderType Type,
	quantity shared.Quantity,
	requestedPrice, stopPrice, targetPrice *shared.Price,
	strategyName string,
	signalID shared.SignalID,
	riskDecisionID shared.RiskDecisionID,
	now time.Time,
) (*Order, error) {
	if !side.Valid() {
		return nil, shared.NewValidationError("side", "must be BUY or SELL")
	}
	if !orderType.Valid() {
		return nil, shared.NewValidationError("type", "unsupported order type")
	}
	if quantity.IsZero() {
		return nil, shared.NewValidationError("quantity", "must be greater than zero")
	}
	if strategyName == "" {
		return nil, shared.NewValidationError("strategy_name", "must not be empty")
	}
	return &Order{
		ID:             shared.NewOrderID(),
		ClientOrderID:  clientOrderID,
		Symbol:         symbol,
		Side:           side,
		Type:           orderType,
		Quantity:       quantity,
		FilledQuantity: shared.ZeroQuantity(),
		RequestedPrice: requestedPrice,
		StopPrice:      stopPrice,
		TargetPrice:    targetPrice,
		Status:         StatusPending,
		StrategyName:   strategyName,
		SignalID:       signalID,
		RiskDecisionID: riskDecisionID,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (o *Order) transition(target Status, now time.Time) error {
	if !o.Status.CanTransition(target) {
		return shared.NewConflictError("order", string(o.Status), string(target))
	}
	o.Status = target
	o.UpdatedAt = now
	o.Version++
	return nil
}

// Submit records the broker's acknowledgement of the order.
func (o *Order) Submit(brokerOrderID string, now time.Time) error {
	if brokerOrderID == "" {
		return shared.NewValidationError("broker_order_id", "must not be empty")
	}
	if err := o.transition(StatusSubmitted, now); err != nil {
		return err
	}
	o.BrokerOrderID = brokerOrderID
	return nil
}

// Fail records a local failure that happened before the broker ever saw the
// order (e.g. the HTTP call itself errored with no ambiguity).
func (o *Order) Fail(reason string, now time.Time) error {
	if err := o.transition(StatusFailed, now); err != nil {
		return err
	}
	o.FailureReason = reason
	return nil
}

// MarkPartiallyFilled records a partial execution report from the broker.
func (o *Order) MarkPartiallyFilled(filledQuantity shared.Quantity, averagePrice shared.Price, now time.Time) error {
	if filledQuantity.IsZero() {
		return shared.NewValidationError("filled_quantity", "must be greater than zero")
	}
	if filledQuantity.GreaterThan(o.Quantity) {
		return shared.NewValidationError("filled_quantity", "must not exceed order quantity")
	}
	if err := o.transition(StatusPartiallyFilled, now); err != nil {
		return err
	}
	o.FilledQuantity = filledQuantity
	o.AverageExecutionPrice = &averagePrice
	return nil
}

// MarkFilled records full execution of the order.
func (o *Order) MarkFilled(averagePrice shared.Price, now time.Time) error {
	if err := o.transition(StatusFilled, now); err != nil {
		return err
	}
	o.FilledQuantity = o.Quantity
	o.AverageExecutionPrice = &averagePrice
	return nil
}

// Cancel records that the order was cancelled, at the broker or locally.
func (o *Order) Cancel(now time.Time) error {
	return o.transition(StatusCancelled, now)
}

// Reject records a broker rejection.
func (o *Order) Reject(reason string, now time.Time) error {
	if err := o.transition(StatusRejected, now); err != nil {
		return err
	}
	o.FailureReason = reason
	return nil
}

// MarkUnknown records that the outcome of a broker call could not be
// determined (e.g. a request timeout with no confirmation either way). The
// order must be reconciled before any further action is taken on it.
func (o *Order) MarkUnknown(now time.Time) error {
	return o.transition(StatusUnknown, now)
}

// IsPending reports whether the order has not yet been submitted.
func (o *Order) IsPending() bool { return o.Status == StatusPending }

// RemainingQuantity returns the quantity not yet filled.
func (o *Order) RemainingQuantity() (shared.Quantity, error) {
	return o.Quantity.Sub(o.FilledQuantity)
}
