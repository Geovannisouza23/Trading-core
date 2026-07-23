package unit_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/shared"
	paperbroker "trading-core/internal/infrastructure/external/broker/paper"
)

func TestPaperBrokerPlaceOrderIsIdempotentByClientOrderID(t *testing.T) {
	broker := paperbroker.NewBroker(decimal.NewFromInt(10000), decimal.NewFromFloat(0.0004), decimal.NewFromFloat(0))
	ctx := context.Background()

	price := shared.MustNewPrice(decimal.NewFromInt(100))
	req := output.PlaceOrderRequest{
		ClientOrderID: shared.MustNewClientOrderID("idempotent-order-1"),
		Symbol:        shared.MustNewSymbol("BTCUSDT"),
		Side:          shared.SideBuy,
		Type:          "MARKET",
		Quantity:      shared.MustNewQuantity(decimal.NewFromInt(1)),
		Price:         &price,
	}

	first, err := broker.PlaceOrder(ctx, req)
	if err != nil {
		t.Fatalf("first PlaceOrder: %v", err)
	}
	second, err := broker.PlaceOrder(ctx, req)
	if err != nil {
		t.Fatalf("second PlaceOrder: %v", err)
	}
	if first.BrokerOrderID != second.BrokerOrderID {
		t.Fatalf("expected same broker order id on retry, got %s and %s", first.BrokerOrderID, second.BrokerOrderID)
	}

	account, err := broker.GetAccount(ctx)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	// Balance should reflect exactly one fill's fee, not two.
	expectedFee := decimal.NewFromInt(100).Mul(decimal.NewFromFloat(0.0004))
	expectedBalance := decimal.NewFromInt(10000).Sub(expectedFee)
	if !account.Balance.Decimal().Equal(expectedBalance) {
		t.Fatalf("got balance %s, want %s (only one fill should have been applied)", account.Balance.Decimal(), expectedBalance)
	}
}

func TestPaperBrokerFillOpensPositionWithSlippage(t *testing.T) {
	broker := paperbroker.NewBroker(decimal.NewFromInt(10000), decimal.NewFromFloat(0), decimal.NewFromFloat(0.01))
	ctx := context.Background()

	price := shared.MustNewPrice(decimal.NewFromInt(100))
	order, err := broker.PlaceOrder(ctx, output.PlaceOrderRequest{
		ClientOrderID: shared.MustNewClientOrderID("slippage-order-1"),
		Symbol:        shared.MustNewSymbol("BTCUSDT"),
		Side:          shared.SideBuy,
		Type:          "MARKET",
		Quantity:      shared.MustNewQuantity(decimal.NewFromInt(1)),
		Price:         &price,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if order.AveragePrice == nil {
		t.Fatal("expected an average execution price")
	}
	// A buy should fill slightly above the reference price (1% slippage).
	if !order.AveragePrice.Decimal().Equal(decimal.NewFromInt(101)) {
		t.Fatalf("got fill price %s, want 101 (100 + 1%% slippage)", order.AveragePrice.Decimal())
	}

	positions, err := broker.GetPositions(ctx)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("got %d positions, want 1", len(positions))
	}
	if positions[0].Quantity.Decimal().String() != "1" {
		t.Fatalf("got position quantity %s, want 1", positions[0].Quantity)
	}
}

func TestPaperBrokerRejectsOrderWithoutReferencePrice(t *testing.T) {
	broker := paperbroker.NewBroker(decimal.NewFromInt(10000), decimal.NewFromFloat(0), decimal.NewFromFloat(0))
	ctx := context.Background()

	_, err := broker.PlaceOrder(ctx, output.PlaceOrderRequest{
		ClientOrderID: shared.MustNewClientOrderID("no-price-order-1"),
		Symbol:        shared.MustNewSymbol("BTCUSDT"),
		Side:          shared.SideBuy,
		Type:          "MARKET",
		Quantity:      shared.MustNewQuantity(decimal.NewFromInt(1)),
	})
	if err == nil {
		t.Fatal("expected error placing a market order with no reference price")
	}
}

func TestPaperBrokerResetClearsState(t *testing.T) {
	broker := paperbroker.NewBroker(decimal.NewFromInt(10000), decimal.NewFromFloat(0), decimal.NewFromFloat(0))
	ctx := context.Background()

	price := shared.MustNewPrice(decimal.NewFromInt(100))
	if _, err := broker.PlaceOrder(ctx, output.PlaceOrderRequest{
		ClientOrderID: shared.MustNewClientOrderID("reset-order-1"),
		Symbol:        shared.MustNewSymbol("BTCUSDT"),
		Side:          shared.SideBuy,
		Type:          "MARKET",
		Quantity:      shared.MustNewQuantity(decimal.NewFromInt(1)),
		Price:         &price,
	}); err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	if err := broker.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	positions, err := broker.GetPositions(ctx)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("got %d positions after reset, want 0", len(positions))
	}
	account, err := broker.GetAccount(ctx)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if !account.Balance.Decimal().Equal(decimal.NewFromInt(10000)) {
		t.Fatalf("got balance %s after reset, want 10000", account.Balance.Decimal())
	}
}
