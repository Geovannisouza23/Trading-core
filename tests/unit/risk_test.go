package unit_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/domain/account"
	"trading-core/internal/domain/event"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"
	"trading-core/internal/domain/signal"
)

func newTestAccount(t *testing.T, balance decimal.Decimal) *account.Account {
	t.Helper()
	acc, err := account.NewPaperAccount(shared.NewMoney(balance), time.Now())
	if err != nil {
		t.Fatalf("building account: %v", err)
	}
	return acc
}

func newTestSignal(t *testing.T, now time.Time) *signal.TradeSignal {
	t.Helper()
	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	stop := shared.MustNewPrice(decimal.NewFromInt(95))
	target := shared.MustNewPrice(decimal.NewFromInt(115))
	s, err := signal.New(
		shared.MustNewSymbol("BTCUSDT"), shared.SideBuy, entry, stop, target,
		shared.MustNewConfidence(decimal.NewFromFloat(0.7)), "test-strategy", market.RegimeTrending,
		now, now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("building signal: %v", err)
	}
	return s
}

func baseRiskContext(t *testing.T, now time.Time) risk.Context {
	t.Helper()
	acc := newTestAccount(t, decimal.NewFromInt(10000))
	sig := newTestSignal(t, now)
	return risk.Context{
		Now:               now,
		Account:           acc,
		Signal:            sig,
		MarketDataAge:     time.Second,
		BrokerHealthy:     true,
		ExpectedFillPrice: sig.EntryPrice,
		RequestedLeverage: shared.MustNewLeverage(decimal.NewFromInt(1)),
		Thresholds: risk.Thresholds{
			PerTradePct:      shared.MustNewPercentage(decimal.NewFromFloat(0.03)),
			MaxDailyLossPct:  shared.MustNewPercentage(decimal.NewFromFloat(0.09)),
			MaxWeeklyLossPct: shared.MustNewPercentage(decimal.NewFromFloat(0.15)),
			MaxDrawdown:      shared.MustNewDrawdown(decimal.NewFromFloat(0.25)),
			MaxOpenPositions: 1,
			MaxLeverage:      shared.MustNewLeverage(decimal.NewFromInt(1)),
			MaxSignalAge:     30 * time.Second,
			MaxMarketDataAge: 10 * time.Second,
			MaxSlippage:      shared.MustNewSlippage(decimal.NewFromFloat(0.3)),
		},
	}
}

func TestMaxDailyLossSpecification(t *testing.T) {
	now := time.Now()
	ctx := baseRiskContext(t, now)

	if result := (risk.MaxDailyLossSpecification{}).Evaluate(ctx); !result.Satisfied {
		t.Fatal("expected satisfied with zero daily pnl")
	}

	ctx.Account.DailyPnL = shared.NewMoney(decimal.NewFromInt(-901)) // > 9% of 10000
	result := (risk.MaxDailyLossSpecification{}).Evaluate(ctx)
	if result.Satisfied {
		t.Fatal("expected violation once daily loss exceeds threshold")
	}
	if result.ReasonCode != risk.ReasonMaxDailyLossReached {
		t.Fatalf("got reason %s, want %s", result.ReasonCode, risk.ReasonMaxDailyLossReached)
	}
}

func TestSignalNotExpiredSpecification(t *testing.T) {
	now := time.Now()
	ctx := baseRiskContext(t, now)

	if result := (risk.SignalNotExpiredSpecification{}).Evaluate(ctx); !result.Satisfied {
		t.Fatal("expected fresh signal to satisfy the specification")
	}

	ctx.Now = now.Add(2 * time.Minute) // past ValidUntil and past MaxSignalAge
	result := (risk.SignalNotExpiredSpecification{}).Evaluate(ctx)
	if result.Satisfied {
		t.Fatal("expected expired signal to violate the specification")
	}
}

func TestNoCriticalEventSpecificationBlocksAffectedSymbol(t *testing.T) {
	now := time.Now()
	ctx := baseRiskContext(t, now)

	critical, err := event.New(
		"REGULATORY", event.DirectionBearish, event.SeverityCritical,
		shared.MustNewConfidence(decimal.NewFromFloat(0.9)),
		[]shared.Symbol{ctx.Signal.Symbol},
		event.ActionBlockNewEntries, 3, true,
		now, now, now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("building event: %v", err)
	}
	ctx.ActiveEvents = []event.MarketEvent{*critical}

	result := (risk.NoCriticalEventSpecification{}).Evaluate(ctx)
	if result.Satisfied {
		t.Fatal("expected critical event affecting the signal's symbol to block it")
	}
}

func TestCompositeRiskPolicyAggregatesViolations(t *testing.T) {
	now := time.Now()
	ctx := baseRiskContext(t, now)
	ctx.Account.DailyPnL = shared.NewMoney(decimal.NewFromInt(-901))
	ctx.BrokerHealthy = false

	policy := risk.NewDefaultRiskPolicy()
	evaluation := policy.Evaluate(ctx)

	if evaluation.Allowed {
		t.Fatal("expected policy to block when broker unhealthy and daily loss exceeded")
	}
	if len(evaluation.ReasonCodes) < 2 {
		t.Fatalf("expected at least 2 reason codes, got %v", evaluation.ReasonCodes)
	}
	if len(evaluation.AppliedRules) != len(risk.DefaultSpecifications()) {
		t.Fatalf("expected every specification to be recorded as applied, got %d", len(evaluation.AppliedRules))
	}
}

func TestFixedRiskPositionSizing(t *testing.T) {
	sizer := risk.FixedRiskPositionSizing{}
	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	stop := shared.MustNewPrice(decimal.NewFromInt(95)) // 5 distance

	result := sizer.Size(risk.SizingInput{
		AvailableBalance: shared.NewMoney(decimal.NewFromInt(10000)),
		RiskPerTrade:     shared.MustNewPercentage(decimal.NewFromFloat(0.03)), // risk $300
		EntryPrice:       entry,
		StopPrice:        stop,
		MinQuantity:      shared.MustNewQuantity(decimal.NewFromFloat(0.001)),
		StepSize:         decimal.NewFromFloat(0.001),
		MinNotional:      shared.NewMoney(decimal.NewFromInt(5)),
		MaxLeverage:      shared.MustNewLeverage(decimal.NewFromInt(1)),
	})

	if result.Rejected {
		t.Fatalf("expected sizing to succeed, got rejected: %s", result.Reason)
	}
	// riskAmount 300 / stopDistance 5 = 60, capped by leverage (10000*1/100=100) -> 60 stays.
	if result.Quantity.Decimal().String() != "60" {
		t.Fatalf("got quantity %s, want 60", result.Quantity.Decimal())
	}
}

func TestFixedRiskPositionSizingRejectsBelowMinNotional(t *testing.T) {
	sizer := risk.FixedRiskPositionSizing{}
	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	stop := shared.MustNewPrice(decimal.NewFromInt(99))

	result := sizer.Size(risk.SizingInput{
		AvailableBalance: shared.NewMoney(decimal.NewFromInt(100)),
		RiskPerTrade:     shared.MustNewPercentage(decimal.NewFromFloat(0.001)), // tiny risk budget
		EntryPrice:       entry,
		StopPrice:        stop,
		MinQuantity:      shared.MustNewQuantity(decimal.NewFromFloat(0.001)),
		StepSize:         decimal.NewFromFloat(0.001),
		MinNotional:      shared.NewMoney(decimal.NewFromInt(1000)), // unreachable given the tiny risk budget
		MaxLeverage:      shared.MustNewLeverage(decimal.NewFromInt(1)),
	})

	if !result.Rejected {
		t.Fatalf("expected rejection below minimum notional, got quantity %s", result.Quantity)
	}
}
