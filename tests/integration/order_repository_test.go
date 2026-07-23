//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/order"
	"trading-core/internal/domain/shared"
	orderdb "trading-core/internal/infrastructure/database/order"
	signaldb "trading-core/internal/infrastructure/database/signal"

	"trading-core/internal/domain/market"
	"trading-core/internal/domain/signal"
)

func mustBuildOrder(t *testing.T, clientOrderID string) *order.Order {
	t.Helper()
	price := shared.MustNewPrice(decimal.NewFromInt(100))
	qty := shared.MustNewQuantity(decimal.NewFromInt(1))
	o, err := order.NewPendingOrder(
		shared.MustNewClientOrderID(clientOrderID),
		shared.MustNewSymbol("BTCUSDT"),
		shared.SideBuy,
		order.TypeMarket,
		qty,
		&price, nil, nil,
		"integration-test",
		shared.NewSignalID(),
		shared.NewRiskDecisionID(),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("building order: %v", err)
	}
	return o
}

func TestOrderRepositorySaveAndGetByClientOrderID(t *testing.T) {
	pool := newTestPool(t)
	repo := orderdb.NewRepository(pool)
	ctx := context.Background()

	o := mustBuildOrder(t, "integration-order-0001")
	if err := repo.Save(ctx, o); err != nil {
		t.Fatalf("save: %v", err)
	}

	fetched, err := repo.GetByClientOrderID(ctx, o.ClientOrderID)
	if err != nil {
		t.Fatalf("get by client order id: %v", err)
	}
	if !fetched.ID.Equal(o.ID) {
		t.Fatalf("got id %s, want %s", fetched.ID, o.ID)
	}
	if fetched.Status != order.StatusPending {
		t.Fatalf("got status %s, want PENDING", fetched.Status)
	}
}

func TestOrderRepositoryEnforcesUniqueClientOrderID(t *testing.T) {
	pool := newTestPool(t)
	repo := orderdb.NewRepository(pool)
	ctx := context.Background()

	clientOrderID := "integration-order-duplicate-01"
	first := mustBuildOrder(t, clientOrderID)
	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("first save: %v", err)
	}

	second := mustBuildOrder(t, clientOrderID) // different order id, same client_order_id
	err := repo.Save(ctx, second)
	if err == nil {
		t.Fatal("expected a unique constraint violation for a duplicate client_order_id, got nil")
	}
}

func TestOrderRepositoryStatusTransitionPersists(t *testing.T) {
	pool := newTestPool(t)
	repo := orderdb.NewRepository(pool)
	ctx := context.Background()

	o := mustBuildOrder(t, "integration-order-0002")
	if err := repo.Save(ctx, o); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	now := time.Now()
	if err := o.Submit("BROKER-XYZ", now); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := repo.Save(ctx, o); err != nil {
		t.Fatalf("save after submit: %v", err)
	}

	fillPrice := shared.MustNewPrice(decimal.NewFromInt(101))
	if err := o.MarkFilled(fillPrice, now); err != nil {
		t.Fatalf("mark filled: %v", err)
	}
	if err := repo.Save(ctx, o); err != nil {
		t.Fatalf("save after fill: %v", err)
	}

	fetched, err := repo.GetByID(ctx, o.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if fetched.Status != order.StatusFilled {
		t.Fatalf("got status %s, want FILLED", fetched.Status)
	}
	if fetched.BrokerOrderID != "BROKER-XYZ" {
		t.Fatalf("got broker order id %q, want BROKER-XYZ", fetched.BrokerOrderID)
	}
	if fetched.Version != 3 { // 1 (pending) -> 2 (submitted) -> 3 (filled)
		t.Fatalf("got version %d, want 3", fetched.Version)
	}
}

func TestOrderRepositoryListOpenExcludesTerminalStates(t *testing.T) {
	pool := newTestPool(t)
	repo := orderdb.NewRepository(pool)
	ctx := context.Background()

	open := mustBuildOrder(t, "integration-order-open-01")
	if err := repo.Save(ctx, open); err != nil {
		t.Fatalf("save open: %v", err)
	}

	terminal := mustBuildOrder(t, "integration-order-terminal-01")
	if err := terminal.Fail("test failure", time.Now()); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if err := repo.Save(ctx, terminal); err != nil {
		t.Fatalf("save terminal: %v", err)
	}

	openOrders, err := repo.ListOpen(ctx)
	if err != nil {
		t.Fatalf("list open: %v", err)
	}
	for _, o := range openOrders {
		if o.ID.Equal(terminal.ID) {
			t.Fatal("ListOpen returned a FAILED order")
		}
	}
	found := false
	for _, o := range openOrders {
		if o.ID.Equal(open.ID) {
			found = true
		}
	}
	if !found {
		t.Fatal("ListOpen did not return the still-pending order")
	}
}

// signalRepositoryRoundTrip is exercised here (rather than its own file)
// because ExecuteApprovedOrder's real flow always persists the signal
// before the order that references it — this keeps the fixture realistic.
func TestSignalRepositorySaveAndGetByID(t *testing.T) {
	pool := newTestPool(t)
	repo := signaldb.NewRepository(pool)
	ctx := context.Background()

	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	stop := shared.MustNewPrice(decimal.NewFromInt(95))
	target := shared.MustNewPrice(decimal.NewFromInt(115))
	now := time.Now()
	s, err := signal.New(
		shared.MustNewSymbol("BTCUSDT"), shared.SideBuy, entry, stop, target,
		shared.MustNewConfidence(decimal.NewFromFloat(0.7)), "integration-test", market.RegimeTrending,
		now, now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("building signal: %v", err)
	}
	if err := repo.Save(ctx, s); err != nil {
		t.Fatalf("save: %v", err)
	}

	fetched, err := repo.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if !fetched.EntryPrice.Equal(s.EntryPrice) {
		t.Fatalf("got entry price %s, want %s", fetched.EntryPrice, s.EntryPrice)
	}

	if _, err := repo.GetByID(ctx, shared.NewSignalID()); !errors.Is(err, output.ErrNotFound) {
		t.Fatalf("got error %v, want output.ErrNotFound for unknown id", err)
	}
}
