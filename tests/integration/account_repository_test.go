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
	"trading-core/internal/domain/shared"
	accountdb "trading-core/internal/infrastructure/database/account"
)

func TestAccountRepositorySaveAndGetActive(t *testing.T) {
	pool := newTestPool(t)
	repo := accountdb.NewRepository(pool)
	ctx := context.Background()

	acc, err := account.NewPaperAccount(shared.NewMoney(decimal.NewFromInt(10000)), time.Now())
	if err != nil {
		t.Fatalf("building account: %v", err)
	}
	if err := repo.Save(ctx, acc); err != nil {
		t.Fatalf("save: %v", err)
	}

	fetched, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if !fetched.ID.Equal(acc.ID) {
		t.Fatalf("got id %s, want %s", fetched.ID, acc.ID)
	}
	if !fetched.Balance.Equal(acc.Balance) {
		t.Fatalf("got balance %s, want %s", fetched.Balance, acc.Balance)
	}
}

func TestAccountRepositoryOptimisticLocking(t *testing.T) {
	pool := newTestPool(t)
	repo := accountdb.NewRepository(pool)
	ctx := context.Background()

	acc, err := account.NewPaperAccount(shared.NewMoney(decimal.NewFromInt(10000)), time.Now())
	if err != nil {
		t.Fatalf("building account: %v", err)
	}
	if err := repo.Save(ctx, acc); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	// Two independent loads of the same row, simulating concurrent writers.
	first, err := repo.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("get by id (first): %v", err)
	}
	second, err := repo.GetByID(ctx, acc.ID)
	if err != nil {
		t.Fatalf("get by id (second): %v", err)
	}

	first.ApplyRealizedPnL(shared.NewMoney(decimal.NewFromInt(100)), time.Now())
	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("first save should succeed: %v", err)
	}

	second.ApplyRealizedPnL(shared.NewMoney(decimal.NewFromInt(-50)), time.Now())
	err = repo.Save(ctx, second)
	if err == nil {
		t.Fatal("expected the second, stale-versioned save to fail")
	}
	if !errors.Is(err, output.ErrOptimisticLock) {
		t.Fatalf("got error %v, want wrapping output.ErrOptimisticLock", err)
	}
}

func TestAccountRepositoryGetByIDNotFound(t *testing.T) {
	pool := newTestPool(t)
	repo := accountdb.NewRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, shared.NewAccountID())
	if !errors.Is(err, output.ErrNotFound) {
		t.Fatalf("got error %v, want output.ErrNotFound", err)
	}
}
