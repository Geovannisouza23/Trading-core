package unit_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/domain/order"
	"trading-core/internal/domain/shared"
)

func newTestOrder(t *testing.T) *order.Order {
	t.Helper()
	price := shared.MustNewPrice(decimal.NewFromInt(100))
	qty := shared.MustNewQuantity(decimal.NewFromInt(1))
	o, err := order.NewPendingOrder(
		shared.MustNewClientOrderID("test-order-id-0001"),
		shared.MustNewSymbol("BTCUSDT"),
		shared.SideBuy,
		order.TypeMarket,
		qty,
		&price, nil, nil,
		"test-strategy",
		shared.NewSignalID(),
		shared.NewRiskDecisionID(),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("building order: %v", err)
	}
	return o
}

func TestOrderPendingToSubmittedToFilled(t *testing.T) {
	o := newTestOrder(t)
	now := time.Now()

	if err := o.Submit("BROKER-1", now); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if o.Status != order.StatusSubmitted {
		t.Fatalf("got status %s, want SUBMITTED", o.Status)
	}

	fillPrice := shared.MustNewPrice(decimal.NewFromInt(101))
	if err := o.MarkFilled(fillPrice, now); err != nil {
		t.Fatalf("mark filled: %v", err)
	}
	if o.Status != order.StatusFilled {
		t.Fatalf("got status %s, want FILLED", o.Status)
	}
	if !o.FilledQuantity.Equal(o.Quantity) {
		t.Fatalf("filled quantity %s != order quantity %s", o.FilledQuantity, o.Quantity)
	}
}

func TestOrderCannotSkipSubmission(t *testing.T) {
	o := newTestOrder(t)
	fillPrice := shared.MustNewPrice(decimal.NewFromInt(101))
	if err := o.MarkFilled(fillPrice, time.Now()); err == nil {
		t.Fatal("expected error transitioning PENDING -> FILLED directly, got nil")
	}
}

func TestOrderTerminalStatesRejectFurtherTransitions(t *testing.T) {
	o := newTestOrder(t)
	now := time.Now()
	if err := o.Submit("BROKER-1", now); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := o.Cancel(now); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if o.Status != order.StatusCancelled {
		t.Fatalf("got %s, want CANCELLED", o.Status)
	}
	if err := o.Submit("BROKER-2", now); err == nil {
		t.Fatal("expected error re-submitting a cancelled order, got nil")
	}
}

func TestOrderUnknownStateCanRecoverToAnyTerminalOrActiveState(t *testing.T) {
	o := newTestOrder(t)
	now := time.Now()
	if err := o.Submit("BROKER-1", now); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := o.MarkUnknown(now); err != nil {
		t.Fatalf("mark unknown: %v", err)
	}
	if o.Status != order.StatusUnknown {
		t.Fatalf("got %s, want UNKNOWN", o.Status)
	}

	fillPrice := shared.MustNewPrice(decimal.NewFromInt(101))
	if err := o.MarkFilled(fillPrice, now); err != nil {
		t.Fatalf("expected UNKNOWN -> FILLED to be allowed (reconciliation outcome): %v", err)
	}
}

func TestOrderPartialFillRejectsOverfill(t *testing.T) {
	o := newTestOrder(t)
	now := time.Now()
	if err := o.Submit("BROKER-1", now); err != nil {
		t.Fatalf("submit: %v", err)
	}
	tooMuch := shared.MustNewQuantity(decimal.NewFromInt(999))
	price := shared.MustNewPrice(decimal.NewFromInt(100))
	if err := o.MarkPartiallyFilled(tooMuch, price, now); err == nil {
		t.Fatal("expected error filling more than the order quantity, got nil")
	}
}
