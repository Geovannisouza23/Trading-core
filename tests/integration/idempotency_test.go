//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	idempotencydb "trading-core/internal/infrastructure/database/idempotency"
)

func TestIdempotencyReserveOnlySucceedsOnce(t *testing.T) {
	pool := newTestPool(t)
	repo := idempotencydb.NewRepository(pool)
	ctx := context.Background()

	first, err := repo.Reserve(ctx, "test-scope", "key-1", time.Minute)
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if !first {
		t.Fatal("expected the first reservation to succeed")
	}

	second, err := repo.Reserve(ctx, "test-scope", "key-1", time.Minute)
	if err != nil {
		t.Fatalf("second reserve: %v", err)
	}
	if second {
		t.Fatal("expected the second reservation of the same key to fail")
	}

	// A different key in the same scope is independent.
	third, err := repo.Reserve(ctx, "test-scope", "key-2", time.Minute)
	if err != nil {
		t.Fatalf("third reserve: %v", err)
	}
	if !third {
		t.Fatal("expected a reservation of a different key to succeed")
	}
}

func TestIdempotencyReserveAllowsReclaimAfterExpiry(t *testing.T) {
	pool := newTestPool(t)
	repo := idempotencydb.NewRepository(pool)
	ctx := context.Background()

	first, err := repo.Reserve(ctx, "test-scope-ttl", "expiring-key", 500*time.Millisecond)
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if !first {
		t.Fatal("expected the first reservation to succeed")
	}

	time.Sleep(700 * time.Millisecond)

	second, err := repo.Reserve(ctx, "test-scope-ttl", "expiring-key", time.Minute)
	if err != nil {
		t.Fatalf("second reserve after expiry: %v", err)
	}
	if !second {
		t.Fatal("expected a reservation to succeed again once the previous one expired")
	}
}
