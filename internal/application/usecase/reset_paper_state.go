package usecase

import (
	"context"
	"errors"
	"fmt"

	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/account"
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/shared"
)

// PaperResettable is implemented by broker adapters that can wipe their own
// simulated state (only internal/infrastructure/external/broker/paper
// does). It is intentionally not part of output.Broker: no other adapter
// could honor it safely.
type PaperResettable interface {
	Reset(ctx context.Context) error
}

// ResetPaperState implements input.ResetPaperStateUseCase.
type ResetPaperState struct {
	accounts        output.AccountRepository
	positions       output.PositionRepository
	orders          output.OrderRepository
	broker          output.Broker
	tx              output.TransactionManager
	clock           output.Clock
	startingBalance shared.Money
}

func NewResetPaperState(
	accounts output.AccountRepository,
	positions output.PositionRepository,
	orders output.OrderRepository,
	broker output.Broker,
	tx output.TransactionManager,
	clock output.Clock,
	startingBalance shared.Money,
) *ResetPaperState {
	return &ResetPaperState{
		accounts: accounts, positions: positions, orders: orders,
		broker: broker, tx: tx, clock: clock, startingBalance: startingBalance,
	}
}

var _ input.ResetPaperStateUseCase = (*ResetPaperState)(nil)

func (uc *ResetPaperState) Execute(ctx context.Context) error {
	acct, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("loading account: %w", err)
	}
	if acct.OperationalMode != operation.ModePaper {
		return fmt.Errorf("%w: paper reset is only allowed in PAPER mode (current: %s)", ErrOperationalModeBlocked, acct.OperationalMode)
	}

	resettable, ok := uc.broker.(PaperResettable)
	if !ok {
		return errors.New("configured broker does not support reset")
	}

	now := uc.clock.Now()

	openPositions, err := uc.positions.ListOpen(ctx)
	if err != nil {
		return fmt.Errorf("loading open positions: %w", err)
	}
	openOrders, err := uc.orders.ListOpen(ctx)
	if err != nil {
		return fmt.Errorf("loading open orders: %w", err)
	}

	fresh, err := account.NewPaperAccount(uc.startingBalance, now)
	if err != nil {
		return err
	}
	fresh.ID = acct.ID
	fresh.Version = acct.Version + 1

	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		for i := range openPositions {
			pos := &openPositions[i]
			if err := pos.Close(pos.CurrentPrice, now); err != nil {
				return err
			}
			if err := uc.positions.Save(ctx, pos); err != nil {
				return err
			}
		}
		for i := range openOrders {
			ord := &openOrders[i]
			if err := ord.Cancel(now); err != nil {
				continue // best-effort: some states may not allow cancel; reset proceeds regardless
			}
			if err := uc.orders.Save(ctx, ord); err != nil {
				return err
			}
		}
		return uc.accounts.Save(ctx, fresh)
	}); err != nil {
		return fmt.Errorf("resetting paper state: %w", err)
	}

	return resettable.Reset(ctx)
}
