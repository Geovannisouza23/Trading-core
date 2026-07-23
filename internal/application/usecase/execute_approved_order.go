package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"trading-core/internal/application/command"
	"trading-core/internal/application/mapper"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/account"
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/order"
	"trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
)

// ExecuteApprovedOrder implements input.ExecuteApprovedOrderUseCase. It is
// the ONLY use case allowed to call output.Broker.PlaceOrder /
// output.Broker.PlaceStop.
type ExecuteApprovedOrder struct {
	decisions  output.RiskDecisionRepository
	signals    output.TradeSignalRepository
	accounts   output.AccountRepository
	orders     output.OrderRepository
	positions  output.PositionRepository
	incidents  output.SystemIncidentRepository
	idempotent output.IdempotencyRepository
	outbox     output.OutboxRepository
	broker     output.Broker
	tx         output.TransactionManager
	clock      output.Clock
}

func NewExecuteApprovedOrder(
	decisions output.RiskDecisionRepository,
	signals output.TradeSignalRepository,
	accounts output.AccountRepository,
	orders output.OrderRepository,
	positions output.PositionRepository,
	incidents output.SystemIncidentRepository,
	idempotent output.IdempotencyRepository,
	outbox output.OutboxRepository,
	broker output.Broker,
	tx output.TransactionManager,
	clock output.Clock,
) *ExecuteApprovedOrder {
	return &ExecuteApprovedOrder{
		decisions:  decisions,
		signals:    signals,
		accounts:   accounts,
		orders:     orders,
		positions:  positions,
		incidents:  incidents,
		idempotent: idempotent,
		outbox:     outbox,
		broker:     broker,
		tx:         tx,
		clock:      clock,
	}
}

var _ input.ExecuteApprovedOrderUseCase = (*ExecuteApprovedOrder)(nil)

func skip(reason string) input.ExecuteApprovedOrderResult {
	return input.ExecuteApprovedOrderResult{Skipped: true, SkipReason: reason}
}

func (uc *ExecuteApprovedOrder) Execute(ctx context.Context, cmd command.ExecuteApprovedOrderCommand) (input.ExecuteApprovedOrderResult, error) {
	now := uc.clock.Now()

	decision, err := uc.decisions.GetByID(ctx, cmd.RiskDecisionID)
	if err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("loading risk decision: %w", err)
	}
	if !decision.Allowed {
		return skip("risk decision did not allow this signal"), nil
	}

	tradeSignal, err := uc.signals.GetByID(ctx, decision.SignalID)
	if err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("loading signal: %w", err)
	}

	acct, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("loading account: %w", err)
	}
	if !acct.OperationalMode.AllowsNewEntries() {
		return skip(fmt.Sprintf("operational mode %s does not allow new entries", acct.OperationalMode)), nil
	}

	// The risk decision ID is a stable, unique, 36-character UUID string,
	// so it doubles as a deterministic ClientOrderID: retrying execution
	// for the same decision always produces the same idempotency key.
	clientOrderID, err := shared.NewClientOrderID(decision.ID.String())
	if err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("building client order id: %w", err)
	}

	if existing, err := uc.orders.GetByClientOrderID(ctx, clientOrderID); err == nil {
		dto := mapper.ToOrderDTO(existing)
		return input.ExecuteApprovedOrderResult{Order: &dto, Skipped: true, SkipReason: "order already exists for this risk decision"}, nil
	} else if !errors.Is(err, output.ErrNotFound) {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("checking existing order: %w", err)
	}

	reserved, err := uc.idempotent.Reserve(ctx, "execute_order", decision.ID.String(), 10*time.Minute)
	if err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("reserving idempotency key: %w", err)
	}
	if !reserved {
		return skip("execution for this risk decision is already in progress"), nil
	}

	stop := tradeSignal.StopPrice
	target := tradeSignal.TargetPrice
	newOrder, err := order.NewPendingOrder(
		clientOrderID,
		tradeSignal.Symbol,
		tradeSignal.Side,
		order.TypeMarket,
		decision.ApprovedPositionSize,
		nil, &stop, &target,
		tradeSignal.StrategyName,
		tradeSignal.ID,
		decision.ID,
		now,
	)
	if err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("building order: %w", err)
	}

	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.orders.Save(ctx, newOrder); err != nil {
			return err
		}
		evt, err := newOutboxEvent("OrderCreated", order.OrderCreated{Order: newOrder, OccurredAt_: now}, now)
		if err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, evt)
	}); err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("persisting pending order: %w", err)
	}

	brokerOrder, err := uc.broker.PlaceOrder(ctx, output.PlaceOrderRequest{
		ClientOrderID: clientOrderID,
		Symbol:        tradeSignal.Symbol,
		Side:          tradeSignal.Side,
		Type:          string(order.TypeMarket),
		Quantity:      decision.ApprovedPositionSize,
		// Market orders carry no limit price at real exchanges, but the
		// signal's entry price still travels along as the reference quote
		// adapters use to simulate/measure execution (e.g. PaperBroker has
		// no live feed of its own).
		Price: &tradeSignal.EntryPrice,
	})
	if err != nil {
		recovered, lookupErr := uc.broker.FindOrderByClientOrderID(ctx, clientOrderID.String())
		switch {
		case lookupErr == nil:
			brokerOrder = recovered
		case errors.Is(lookupErr, output.ErrNotFound):
			_ = newOrder.Fail(err.Error(), uc.clock.Now())
			_ = uc.orders.Save(ctx, newOrder)
			return input.ExecuteApprovedOrderResult{}, fmt.Errorf("placing order: %w", err)
		default:
			_ = newOrder.MarkUnknown(uc.clock.Now())
			_ = uc.orders.Save(ctx, newOrder)
			uc.raiseUnknownStateIncident(ctx, newOrder, err)
			return input.ExecuteApprovedOrderResult{}, fmt.Errorf("%w: %v", ErrBrokerOrderStateUnknown, err)
		}
	}

	submitNow := uc.clock.Now()
	if err := newOrder.Submit(brokerOrder.BrokerOrderID, submitNow); err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("recording submission: %w", err)
	}
	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.orders.Save(ctx, newOrder); err != nil {
			return err
		}
		evt, err := newOutboxEvent("OrderSubmitted", order.OrderSubmitted{Order: newOrder, OccurredAt_: submitNow}, submitNow)
		if err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, evt)
	}); err != nil {
		return input.ExecuteApprovedOrderResult{}, fmt.Errorf("persisting submitted order: %w", err)
	}

	if err := uc.reconcileFreshOrder(ctx, newOrder, brokerOrder, tradeSignal.StopPrice, tradeSignal.TargetPrice, acct); err != nil {
		return input.ExecuteApprovedOrderResult{}, err
	}

	dto := mapper.ToOrderDTO(newOrder)
	return input.ExecuteApprovedOrderResult{Order: &dto}, nil
}

// reconcileFreshOrder interprets the broker's immediate response to a
// just-placed order (paper and most exchanges fill market orders
// synchronously) and books the resulting fill/position/stop.
func (uc *ExecuteApprovedOrder) reconcileFreshOrder(
	ctx context.Context,
	localOrder *order.Order,
	brokerOrder output.BrokerOrder,
	stopPrice, targetPrice shared.Price,
	acct *account.Account,
) error {
	if brokerOrder.RawStatus != "FILLED" || brokerOrder.AveragePrice == nil {
		// Not immediately filled (e.g. a resting limit order): leave the
		// order Submitted. ReconcileBrokerState will pick up the eventual
		// fill on its next pass.
		return nil
	}

	fillNow := uc.clock.Now()
	if err := localOrder.MarkFilled(*brokerOrder.AveragePrice, fillNow); err != nil {
		return fmt.Errorf("recording fill: %w", err)
	}

	pos, err := position.Open(localOrder.Symbol, localOrder.Side, localOrder.Quantity, *brokerOrder.AveragePrice, &stopPrice, &targetPrice, fillNow)
	if err != nil {
		return fmt.Errorf("opening position: %w", err)
	}

	notional := shared.NewMoney(pos.Quantity.Decimal().Mul(pos.EntryPrice.Decimal()))
	if err := acct.ReserveMargin(notional, fillNow); err != nil {
		uc.raiseUnknownStateIncident(ctx, localOrder, err)
	}

	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.orders.Save(ctx, localOrder); err != nil {
			return err
		}
		if err := uc.positions.Save(ctx, pos); err != nil {
			return err
		}
		if err := uc.accounts.Save(ctx, acct); err != nil {
			return err
		}
		filledEvt, err := newOutboxEvent("OrderFilled", order.OrderFilled{Order: localOrder, OccurredAt_: fillNow}, fillNow)
		if err != nil {
			return err
		}
		if err := uc.outbox.Insert(ctx, filledEvt); err != nil {
			return err
		}
		openedEvt, err := newOutboxEvent("PositionOpened", position.PositionOpened{Position: pos, OccurredAt_: fillNow}, fillNow)
		if err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, openedEvt)
	}); err != nil {
		return fmt.Errorf("persisting fill and position: %w", err)
	}

	if _, err := uc.broker.PlaceStop(ctx, output.PlaceStopRequest{
		ClientOrderID: mustStopClientOrderID(localOrder.ClientOrderID),
		Symbol:        localOrder.Symbol,
		Side:          localOrder.Side.Opposite(),
		Quantity:      localOrder.Quantity,
		StopPrice:     stopPrice,
	}); err != nil {
		uc.raiseUnknownStateIncident(ctx, localOrder, fmt.Errorf("stop placement failed: %w", err))
	}

	return nil
}

// mustStopClientOrderID derives a distinct, still-idempotent client order id
// for the protective stop tied to `orderID`.
func mustStopClientOrderID(orderID shared.ClientOrderID) shared.ClientOrderID {
	id, err := shared.NewClientOrderID("s-" + orderID.String()[:34])
	if err != nil {
		return orderID
	}
	return id
}

func (uc *ExecuteApprovedOrder) raiseUnknownStateIncident(ctx context.Context, o *order.Order, cause error) {
	incident, err := operation.NewIncident(
		operation.IncidentTypeUnknownOrderState,
		operation.IncidentSeverityHigh,
		cause.Error(),
		"ExecuteApprovedOrder",
		o.ID.String(),
		uc.clock.Now(),
	)
	if err != nil {
		return
	}
	_ = uc.incidents.Save(ctx, incident)
}
