//go:build integration

package failure_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/command"
	"trading-core/internal/application/usecase"
	"trading-core/internal/domain/account"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"
	"trading-core/internal/domain/signal"

	accountdb "trading-core/internal/infrastructure/database/account"
	idempotencydb "trading-core/internal/infrastructure/database/idempotency"
	incidentdb "trading-core/internal/infrastructure/database/incident"
	orderdb "trading-core/internal/infrastructure/database/order"
	outboxdb "trading-core/internal/infrastructure/database/outbox"
	positiondb "trading-core/internal/infrastructure/database/position"
	riskdb "trading-core/internal/infrastructure/database/risk"
	signaldb "trading-core/internal/infrastructure/database/signal"
	"trading-core/internal/infrastructure/database/transaction"

	systemclock "trading-core/internal/infrastructure/clock/system"
	paperbroker "trading-core/internal/infrastructure/external/broker/paper"

	"trading-core/tests/fixtures"
)

// TestConcurrentExecuteApprovedOrderOnlyExecutesOnce reproduces spec
// section 21's "duas ordens para o mesmo sinal" / "envio duplicado"
// concern directly against the real use case, real Postgres, and the real
// PaperBroker: two goroutines racing to execute the same RiskDecisionID
// must result in exactly one order being placed.
func TestConcurrentExecuteApprovedOrderOnlyExecutesOnce(t *testing.T) {
	pool := fixtures.NewPostgresPool(t)
	ctx := context.Background()
	clock := systemclock.NewClock()

	accounts := accountdb.NewRepository(pool)
	signals := signaldb.NewRepository(pool)
	decisions := riskdb.NewRepository(pool)
	orders := orderdb.NewRepository(pool)
	positions := positiondb.NewRepository(pool)
	incidents := incidentdb.NewRepository(pool)
	idempotent := idempotencydb.NewRepository(pool)
	outbox := outboxdb.NewRepository(pool)
	tx := transaction.NewManager(pool)
	broker := paperbroker.NewBroker(decimal.NewFromInt(100000), decimal.NewFromFloat(0.0004), decimal.NewFromFloat(0))

	acc, err := account.NewPaperAccount(shared.NewMoney(decimal.NewFromInt(100000)), clock.Now())
	if err != nil {
		t.Fatalf("building account: %v", err)
	}
	if err := accounts.Save(ctx, acc); err != nil {
		t.Fatalf("saving account: %v", err)
	}

	entry := shared.MustNewPrice(decimal.NewFromInt(100))
	stop := shared.MustNewPrice(decimal.NewFromInt(95))
	target := shared.MustNewPrice(decimal.NewFromInt(115))
	sig, err := signal.New(
		shared.MustNewSymbol("BTCUSDT"), shared.SideBuy, entry, stop, target,
		shared.MustNewConfidence(decimal.NewFromFloat(0.8)), "concurrency-test", market.RegimeTrending,
		clock.Now(), clock.Now().Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("building signal: %v", err)
	}
	if err := signals.Save(ctx, sig); err != nil {
		t.Fatalf("saving signal: %v", err)
	}

	decision := risk.NewDecision(
		sig.ID,
		risk.Evaluation{Allowed: true, AppliedRules: []string{"test"}},
		shared.MustNewQuantity(decimal.NewFromInt(1)),
		risk.SizingResult{Quantity: shared.MustNewQuantity(decimal.NewFromInt(1))},
		nil,
		clock.Now(),
	)
	if err := decisions.Save(ctx, decision); err != nil {
		t.Fatalf("saving risk decision: %v", err)
	}

	execute := usecase.NewExecuteApprovedOrder(decisions, signals, accounts, orders, positions, incidents, idempotent, outbox, broker, tx, clock)

	var wg sync.WaitGroup
	results := make([]struct {
		skipped bool
		err     error
	}, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := execute.Execute(ctx, command.ExecuteApprovedOrderCommand{RiskDecisionID: decision.ID})
			results[idx].skipped = result.Skipped
			results[idx].err = err
		}(i)
	}
	wg.Wait()

	successCount := 0
	skippedCount := 0
	for _, r := range results {
		if r.err != nil {
			t.Fatalf("unexpected error from concurrent execution: %v", r.err)
		}
		if r.skipped {
			skippedCount++
		} else {
			successCount++
		}
	}
	if successCount != 1 || skippedCount != 1 {
		t.Fatalf("got successCount=%d skippedCount=%d, want exactly one success and one skip", successCount, skippedCount)
	}

	all, err := orders.List(ctx, 10)
	if err != nil {
		t.Fatalf("listing orders: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d total orders persisted, want exactly 1 (no duplicate execution)", len(all))
	}
}
