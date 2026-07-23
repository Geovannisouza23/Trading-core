package unit_test

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"

	"trading-core/internal/domain/shared"
)

func TestNewPrice(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"positive is valid", "100.5", false},
		{"zero is invalid", "0", true},
		{"negative is invalid", "-1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := shared.NewPrice(decimal.RequireFromString(tc.value))
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewPrice(%s) error = %v, wantErr %v", tc.value, err, tc.wantErr)
			}
		})
	}
}

func TestQuantitySub(t *testing.T) {
	ten := shared.MustNewQuantity(decimal.NewFromInt(10))
	three := shared.MustNewQuantity(decimal.NewFromInt(3))
	twenty := shared.MustNewQuantity(decimal.NewFromInt(20))

	got, err := ten.Sub(three)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(shared.MustNewQuantity(decimal.NewFromInt(7))) {
		t.Fatalf("got %s, want 7", got)
	}

	if _, err := ten.Sub(twenty); err == nil {
		t.Fatal("expected error subtracting past zero, got nil")
	}
}

func TestNewSymbolNormalizesCase(t *testing.T) {
	s, err := shared.NewSymbol("btcusdt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.String() != "BTCUSDT" {
		t.Fatalf("got %s, want BTCUSDT", s)
	}

	if _, err := shared.NewSymbol("ab"); err == nil {
		t.Fatal("expected error for too-short symbol, got nil")
	}
}

func TestDrawdownFromPeak(t *testing.T) {
	peak := shared.NewMoney(decimal.NewFromInt(1000))
	current := shared.NewMoney(decimal.NewFromInt(750))

	dd := shared.DrawdownFromPeak(peak, current)
	if dd.Decimal().String() != "0.25" {
		t.Fatalf("got drawdown %s, want 0.25", dd.Decimal())
	}

	// Current above peak must never yield a negative drawdown.
	above := shared.NewMoney(decimal.NewFromInt(1100))
	if got := shared.DrawdownFromPeak(peak, above); !got.Decimal().IsZero() {
		t.Fatalf("got drawdown %s, want 0", got.Decimal())
	}
}

func TestMoneyJSONRoundTrip(t *testing.T) {
	m := shared.NewMoney(decimal.RequireFromString("123.45000000"))
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var round shared.Money
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !round.Equal(m) {
		t.Fatalf("round-tripped %s, want %s", round, m)
	}
}

func TestClientOrderIDValidation(t *testing.T) {
	if _, err := shared.NewClientOrderID("short"); err == nil {
		t.Fatal("expected error for too-short client order id")
	}
	id, err := shared.NewClientOrderID("valid-order-id-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.IsEmpty() {
		t.Fatal("expected non-empty id")
	}
}
