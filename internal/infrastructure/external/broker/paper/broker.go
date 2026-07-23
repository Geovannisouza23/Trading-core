// Package paper implements output.Broker as a fully in-process simulator:
// no network calls, no external dependencies. It is the only broker
// adapter that is functional out of the box, matching the requirement that
// the system runs end-to-end in PAPER mode with zero external credentials.
package paper

import (
	"context"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/shared"
)

// Broker simulates a broker: it keeps its own balance/positions/orders
// ledger, applies configurable fees and slippage, and never touches the
// network. State is held in memory and is intentionally lost on restart —
// call Reset (also used by POST /v1/paper/reset) to start over on demand.
type Broker struct {
	mu sync.Mutex

	startingBalance decimal.Decimal
	feePct          decimal.Decimal
	slippagePct     decimal.Decimal

	balance      decimal.Decimal
	positions    map[string]output.BrokerPosition
	orders       map[string]output.BrokerOrder
	byClientID   map[string]string
	nextOrderSeq int
}

// NewBroker builds a PaperBroker seeded with startingBalance.
func NewBroker(startingBalance, feePct, slippagePct decimal.Decimal) *Broker {
	b := &Broker{
		startingBalance: startingBalance,
		feePct:          feePct,
		slippagePct:     slippagePct,
	}
	b.resetLocked()
	return b
}

var _ output.Broker = (*Broker)(nil)

func (b *Broker) resetLocked() {
	b.balance = b.startingBalance
	b.positions = make(map[string]output.BrokerPosition)
	b.orders = make(map[string]output.BrokerOrder)
	b.byClientID = make(map[string]string)
	b.nextOrderSeq = 0
}

// Reset wipes all simulated state back to the starting balance. Implements
// usecase.PaperResettable.
func (b *Broker) Reset(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resetLocked()
	return nil
}

func (b *Broker) GetAccount(ctx context.Context) (output.BrokerAccount, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	equity := b.balance
	for _, pos := range b.positions {
		diff := pos.MarkPrice.Decimal().Sub(pos.EntryPrice.Decimal())
		if pos.Side == shared.SideSell {
			diff = diff.Neg()
		}
		equity = equity.Add(diff.Mul(pos.Quantity.Decimal()))
	}

	return output.BrokerAccount{
		Balance:          shared.NewMoney(b.balance),
		Equity:           shared.NewMoney(equity),
		AvailableBalance: shared.NewMoney(b.balance),
	}, nil
}

func (b *Broker) GetPositions(ctx context.Context) ([]output.BrokerPosition, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	positions := make([]output.BrokerPosition, 0, len(b.positions))
	for _, p := range b.positions {
		positions = append(positions, p)
	}
	return positions, nil
}

func (b *Broker) GetOpenOrders(ctx context.Context) ([]output.BrokerOrder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var open []output.BrokerOrder
	for _, o := range b.orders {
		if o.RawStatus == "NEW" || o.RawStatus == "PARTIALLY_FILLED" {
			open = append(open, o)
		}
	}
	return open, nil
}

// PlaceOrder simulates immediate execution of a market order: slippage is
// applied to the reference price, a fee is deducted, and the resulting fill
// is folded into the simulated position book. Calling PlaceOrder twice with
// the same ClientOrderID is idempotent: the second call returns the
// already-recorded order without side effects.
func (b *Broker) PlaceOrder(ctx context.Context, req output.PlaceOrderRequest) (output.BrokerOrder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if existingID, ok := b.byClientID[req.ClientOrderID.String()]; ok {
		return b.orders[existingID], nil
	}
	if req.Price == nil {
		return output.BrokerOrder{}, fmt.Errorf("paper broker: order %s has no reference price", req.ClientOrderID)
	}

	execPrice, err := applySlippage(*req.Price, req.Side, b.slippagePct)
	if err != nil {
		return output.BrokerOrder{}, err
	}
	notional := req.Quantity.Decimal().Mul(execPrice.Decimal())
	fee := notional.Mul(b.feePct)
	if b.balance.LessThan(notional.Add(fee)) {
		return output.BrokerOrder{}, fmt.Errorf("paper broker: insufficient simulated balance for order %s", req.ClientOrderID)
	}

	b.nextOrderSeq++
	brokerOrderID := fmt.Sprintf("PAPER-%08d", b.nextOrderSeq)

	filled := output.BrokerOrder{
		BrokerOrderID:  brokerOrderID,
		ClientOrderID:  req.ClientOrderID.String(),
		Symbol:         req.Symbol,
		Side:           req.Side,
		Type:           req.Type,
		Quantity:       req.Quantity,
		FilledQuantity: req.Quantity,
		AveragePrice:   &execPrice,
		RawStatus:      "FILLED",
	}

	b.balance = b.balance.Sub(fee)
	b.applyFillLocked(req.Symbol, req.Side, req.Quantity, execPrice)

	b.orders[brokerOrderID] = filled
	b.byClientID[req.ClientOrderID.String()] = brokerOrderID
	return filled, nil
}

// PlaceStop records a resting stop order. The simulator has no live price
// feed of its own, so stops never auto-trigger; they exist so
// GetOpenOrders/GetOrder/CancelOrder behave consistently with a real venue.
func (b *Broker) PlaceStop(ctx context.Context, req output.PlaceStopRequest) (output.BrokerOrder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if existingID, ok := b.byClientID[req.ClientOrderID.String()]; ok {
		return b.orders[existingID], nil
	}

	b.nextOrderSeq++
	brokerOrderID := fmt.Sprintf("PAPER-STOP-%08d", b.nextOrderSeq)
	price := req.StopPrice

	resting := output.BrokerOrder{
		BrokerOrderID:  brokerOrderID,
		ClientOrderID:  req.ClientOrderID.String(),
		Symbol:         req.Symbol,
		Side:           req.Side,
		Type:           "STOP_MARKET",
		Quantity:       req.Quantity,
		FilledQuantity: shared.ZeroQuantity(),
		AveragePrice:   &price,
		RawStatus:      "NEW",
	}
	b.orders[brokerOrderID] = resting
	b.byClientID[req.ClientOrderID.String()] = brokerOrderID
	return resting, nil
}

func (b *Broker) CancelOrder(ctx context.Context, brokerOrderID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	o, ok := b.orders[brokerOrderID]
	if !ok {
		return output.ErrNotFound
	}
	o.RawStatus = "CANCELLED"
	b.orders[brokerOrderID] = o
	return nil
}

func (b *Broker) GetOrder(ctx context.Context, brokerOrderID string) (output.BrokerOrder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	o, ok := b.orders[brokerOrderID]
	if !ok {
		return output.BrokerOrder{}, output.ErrNotFound
	}
	return o, nil
}

func (b *Broker) FindOrderByClientOrderID(ctx context.Context, clientOrderID string) (output.BrokerOrder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	brokerOrderID, ok := b.byClientID[clientOrderID]
	if !ok {
		return output.BrokerOrder{}, output.ErrNotFound
	}
	return b.orders[brokerOrderID], nil
}

// applyFillLocked folds a fill into the simulated position book: opening a
// new position, adding to an existing same-side position, or reducing/
// flipping an opposite-side position and crystallizing realized P&L into
// balance. Caller must hold b.mu.
func (b *Broker) applyFillLocked(symbol shared.Symbol, side shared.Side, quantity shared.Quantity, price shared.Price) {
	key := symbol.String()
	existing, ok := b.positions[key]
	if !ok {
		b.positions[key] = output.BrokerPosition{
			Symbol: symbol, Side: side, Quantity: quantity, EntryPrice: price, MarkPrice: price,
		}
		return
	}

	if existing.Side == side {
		totalNotional := existing.Quantity.Decimal().Mul(existing.EntryPrice.Decimal()).
			Add(quantity.Decimal().Mul(price.Decimal()))
		totalQuantity := existing.Quantity.Decimal().Add(quantity.Decimal())
		avgPrice, err := shared.NewPrice(totalNotional.Div(totalQuantity))
		if err != nil {
			avgPrice = existing.EntryPrice
		}
		newQuantity, err := shared.NewQuantity(totalQuantity)
		if err != nil {
			newQuantity = existing.Quantity
		}
		b.positions[key] = output.BrokerPosition{
			Symbol: symbol, Side: side, Quantity: newQuantity, EntryPrice: avgPrice, MarkPrice: price,
		}
		return
	}

	// Opposite side: reduce (or flip) the existing position and realize P&L.
	diff := price.Decimal().Sub(existing.EntryPrice.Decimal())
	if existing.Side == shared.SideSell {
		diff = diff.Neg()
	}
	closedQuantity := quantity
	if quantity.GreaterThan(existing.Quantity) {
		closedQuantity = existing.Quantity
	}
	realized := diff.Mul(closedQuantity.Decimal())
	b.balance = b.balance.Add(realized)

	remaining, err := existing.Quantity.Sub(closedQuantity)
	if err != nil {
		remaining = shared.ZeroQuantity()
	}
	if remaining.IsZero() {
		flipQuantity, err := quantity.Sub(closedQuantity)
		if err == nil && flipQuantity.GreaterThan(shared.ZeroQuantity()) {
			b.positions[key] = output.BrokerPosition{
				Symbol: symbol, Side: side, Quantity: flipQuantity, EntryPrice: price, MarkPrice: price,
			}
			return
		}
		delete(b.positions, key)
		return
	}
	b.positions[key] = output.BrokerPosition{
		Symbol: symbol, Side: existing.Side, Quantity: remaining, EntryPrice: existing.EntryPrice, MarkPrice: price,
	}
}

// applySlippage nudges the reference price against the position: buyers pay
// slightly more, sellers receive slightly less, matching real market impact.
func applySlippage(reference shared.Price, side shared.Side, slippagePct decimal.Decimal) (shared.Price, error) {
	adjustment := reference.Decimal().Mul(slippagePct)
	adjusted := reference.Decimal().Add(adjustment)
	if side == shared.SideSell {
		adjusted = reference.Decimal().Sub(adjustment)
	}
	return shared.NewPrice(adjusted)
}
