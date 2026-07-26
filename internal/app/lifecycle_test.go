package app

import (
	"context"
	"testing"

	"trading-core/internal/domain/order"
	"trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
)

func TestActivityTransitionTrackerPublishesOnlyOnTheEdge(t *testing.T) {
	var tracker activityTransitionTracker

	if !tracker.observe(true) {
		t.Error("first observation must always count as a transition (unknown -> known)")
	}
	if tracker.observe(true) {
		t.Error("repeating the same idle value must not count as a transition")
	}
	if !tracker.observe(false) {
		t.Error("idle -> active must count as a transition")
	}
	if tracker.observe(false) {
		t.Error("repeating the same active value must not count as a transition")
	}
	if !tracker.observe(true) {
		t.Error("active -> idle must count as a transition")
	}
}

type fakePositionRepository struct {
	open []position.Position
}

func (f *fakePositionRepository) Save(context.Context, *position.Position) error { return nil }
func (f *fakePositionRepository) GetByID(context.Context, shared.PositionID) (*position.Position, error) {
	return nil, nil
}
func (f *fakePositionRepository) ListOpen(context.Context) ([]position.Position, error) {
	return f.open, nil
}
func (f *fakePositionRepository) GetOpenBySymbol(context.Context, shared.Symbol) (*position.Position, error) {
	return nil, nil
}

type fakeOrderRepository struct {
	open []order.Order
}

func (f *fakeOrderRepository) Save(context.Context, *order.Order) error { return nil }
func (f *fakeOrderRepository) GetByID(context.Context, shared.OrderID) (*order.Order, error) {
	return nil, nil
}
func (f *fakeOrderRepository) GetByClientOrderID(context.Context, shared.ClientOrderID) (*order.Order, error) {
	return nil, nil
}
func (f *fakeOrderRepository) ListOpen(context.Context) ([]order.Order, error) {
	return f.open, nil
}
func (f *fakeOrderRepository) ListBySignalID(context.Context, shared.SignalID) ([]order.Order, error) {
	return nil, nil
}
func (f *fakeOrderRepository) List(context.Context, int) ([]order.Order, error) {
	return nil, nil
}

func TestIsSystemIdleRequiresBothZeroPositionsAndZeroOrders(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name          string
		openPositions []position.Position
		openOrders    []order.Order
		wantIdle      bool
	}{
		{"nothing open", nil, nil, true},
		{"open position only", []position.Position{{}}, nil, false},
		{"open order only", nil, []order.Order{{}}, false},
		{"both open", []position.Position{{}}, []order.Order{{}}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			positions := &fakePositionRepository{open: tc.openPositions}
			orders := &fakeOrderRepository{open: tc.openOrders}
			idle, err := isSystemIdle(ctx, positions, orders)
			if err != nil {
				t.Fatalf("isSystemIdle returned an error: %v", err)
			}
			if idle != tc.wantIdle {
				t.Errorf("isSystemIdle() = %v, want %v", idle, tc.wantIdle)
			}
		})
	}
}
