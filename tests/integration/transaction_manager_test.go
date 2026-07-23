//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/account"
	"trading-core/internal/domain/order"
	"trading-core/internal/domain/shared"
	accountdb "trading-core/internal/infrastructure/database/account"
	orderdb "trading-core/internal/infrastructure/database/order"
	outboxdb "trading-core/internal/infrastructure/database/outbox"
	"trading-core/internal/infrastructure/database/transaction"
)

func TestTransactionManagerCommitsBothWritesTogether(t *testing.T) {
	pool := newTestPool(t)
	tx := transaction.NewManager(pool)
	accounts := accountdb.NewRepository(pool)
	outbox := outboxdb.NewRepository(pool)
	ctx := context.Background()

	acc, err := account.NewPaperAccount(shared.NewMoney(decimal.NewFromInt(10000)), time.Now())
	if err != nil {
		t.Fatalf("building account: %v", err)
	}
	if err := accounts.Save(ctx, acc); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	acc.ApplyRealizedPnL(shared.NewMoney(decimal.NewFromInt(100)), time.Now())

	err = tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := accounts.Save(ctx, acc); err != nil {
			return err
		}
		return outbox.Insert(ctx, output.OutboxEvent{
			ID:        "tx-test-event-1",
			EventType: "TestEvent",
			Payload:   []byte(`{"ok":true}`),
			Status:    output.OutboxStatusPending,
			CreatedAt: time.Now(),
		})
	})
	if err != nil {
		t.Fatalf("WithinTransaction: %v", err)
	}

	fetched, err := accounts.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if fetched.Version != 2 {
		t.Fatalf("got version %d, want 2 (transaction committed)", fetched.Version)
	}

	batch, err := outbox.FetchPendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("fetch pending batch: %v", err)
	}
	found := false
	for _, evt := range batch {
		if evt.ID == "tx-test-event-1" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the outbox event inserted in the same transaction to be visible")
	}
}

func TestTransactionManagerRollsBackOnError(t *testing.T) {
	pool := newTestPool(t)
	tx := transaction.NewManager(pool)
	orders := orderdb.NewRepository(pool)
	ctx := context.Background()

	price := shared.MustNewPrice(decimal.NewFromInt(100))
	qty := shared.MustNewQuantity(decimal.NewFromInt(1))
	o, err := order.NewPendingOrder(
		shared.MustNewClientOrderID("tx-rollback-order-01"),
		shared.MustNewSymbol("BTCUSDT"), shared.SideBuy, order.TypeMarket, qty,
		&price, nil, nil, "test", shared.NewSignalID(), shared.NewRiskDecisionID(), time.Now(),
	)
	if err != nil {
		t.Fatalf("building order: %v", err)
	}

	sentinel := errors.New("deliberate failure after the write")
	err = tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := orders.Save(ctx, o); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want the sentinel to propagate", err)
	}

	if _, err := orders.GetByID(ctx, o.ID); !errors.Is(err, output.ErrNotFound) {
		t.Fatalf("got error %v, want output.ErrNotFound (the write must have rolled back)", err)
	}
}
