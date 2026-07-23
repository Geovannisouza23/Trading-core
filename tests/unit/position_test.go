package unit_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
)

func TestPositionUnrealizedPnLLong(t *testing.T) {
	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	qty := shared.MustNewQuantity(decimal.NewFromInt(2))
	pos, err := position.Open(shared.MustNewSymbol("BTCUSDT"), shared.SideBuy, qty, entry, nil, nil, time.Now())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	mark := shared.MustNewPrice(decimal.NewFromInt(110))
	pnl := pos.UnrealizedPnLAt(mark)
	if pnl.Decimal().String() != "20" {
		t.Fatalf("got pnl %s, want 20", pnl.Decimal())
	}
}

func TestPositionUnrealizedPnLShort(t *testing.T) {
	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	qty := shared.MustNewQuantity(decimal.NewFromInt(2))
	pos, err := position.Open(shared.MustNewSymbol("BTCUSDT"), shared.SideSell, qty, entry, nil, nil, time.Now())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	mark := shared.MustNewPrice(decimal.NewFromInt(90))
	pnl := pos.UnrealizedPnLAt(mark)
	if pnl.Decimal().String() != "20" {
		t.Fatalf("got pnl %s, want 20 (short profits when price falls)", pnl.Decimal())
	}
}

func TestPositionCloseRealizesAndZeroesQuantity(t *testing.T) {
	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	qty := shared.MustNewQuantity(decimal.NewFromInt(1))
	pos, err := position.Open(shared.MustNewSymbol("BTCUSDT"), shared.SideBuy, qty, entry, nil, nil, time.Now())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	exit := shared.MustNewPrice(decimal.NewFromInt(150))
	if err := pos.Close(exit, time.Now()); err != nil {
		t.Fatalf("close: %v", err)
	}
	if pos.Status != position.StatusClosed {
		t.Fatalf("got status %s, want CLOSED", pos.Status)
	}
	if pos.RealizedPnL.Decimal().String() != "50" {
		t.Fatalf("got realized pnl %s, want 50", pos.RealizedPnL.Decimal())
	}
	if !pos.Quantity.IsZero() {
		t.Fatalf("got quantity %s, want 0", pos.Quantity)
	}
}

func TestPositionReduceKeepsRemainderOpen(t *testing.T) {
	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	qty := shared.MustNewQuantity(decimal.NewFromInt(4))
	pos, err := position.Open(shared.MustNewSymbol("BTCUSDT"), shared.SideBuy, qty, entry, nil, nil, time.Now())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	exit := shared.MustNewPrice(decimal.NewFromInt(120))
	if err := pos.Reduce(shared.MustNewQuantity(decimal.NewFromInt(1)), exit, time.Now()); err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if pos.Status != position.StatusOpen {
		t.Fatalf("got status %s, want OPEN after partial reduce", pos.Status)
	}
	if pos.Quantity.Decimal().String() != "3" {
		t.Fatalf("got remaining quantity %s, want 3", pos.Quantity)
	}
	if pos.RealizedPnL.Decimal().String() != "20" {
		t.Fatalf("got realized pnl %s, want 20 (1 unit * 20 gain)", pos.RealizedPnL.Decimal())
	}
}
