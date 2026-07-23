//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"trading-core/internal/application/ports/output"
	outboxdb "trading-core/internal/infrastructure/database/outbox"
)

func TestOutboxFetchPendingBatchClaimsRows(t *testing.T) {
	pool := newTestPool(t)
	repo := outboxdb.NewRepository(pool)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.Insert(ctx, output.OutboxEvent{
			ID:        fmt.Sprintf("claim-test-%d", i),
			EventType: "TestEvent",
			Payload:   []byte(`{}`),
			Status:    output.OutboxStatusPending,
			CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	batch, err := repo.FetchPendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("fetch pending batch: %v", err)
	}
	if len(batch) != 3 {
		t.Fatalf("got %d events, want 3", len(batch))
	}

	// A second fetch must return nothing: the first fetch already moved
	// every row from PENDING to PROCESSING.
	second, err := repo.FetchPendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("second fetch pending batch: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("got %d events on second fetch, want 0 (rows should already be claimed)", len(second))
	}
}

func TestOutboxMarkProcessedAndMarkFailed(t *testing.T) {
	pool := newTestPool(t)
	repo := outboxdb.NewRepository(pool)
	ctx := context.Background()

	if err := repo.Insert(ctx, output.OutboxEvent{
		ID: "mark-test-1", EventType: "TestEvent", Payload: []byte(`{}`),
		Status: output.OutboxStatusPending, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := repo.Insert(ctx, output.OutboxEvent{
		ID: "mark-test-2", EventType: "TestEvent", Payload: []byte(`{}`),
		Status: output.OutboxStatusPending, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	batch, err := repo.FetchPendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(batch) != 2 {
		t.Fatalf("got %d events, want 2", len(batch))
	}

	if err := repo.MarkProcessed(ctx, "mark-test-1"); err != nil {
		t.Fatalf("mark processed: %v", err)
	}
	if err := repo.MarkFailed(ctx, "mark-test-2", "boom"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	// Neither a PROCESSED nor a (still-retryable) once-failed-but-under-the-
	// attempt-limit PENDING row should double-claim on the next batch fetch
	// for the processed one; the failed one goes back to PENDING and CAN be
	// claimed again (that's the retry contract).
	nextBatch, err := repo.FetchPendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("fetch after marking: %v", err)
	}
	if len(nextBatch) != 1 || nextBatch[0].ID != "mark-test-2" {
		t.Fatalf("got batch %+v, want exactly the retried mark-test-2 event", nextBatch)
	}
	if nextBatch[0].Attempts != 1 {
		t.Fatalf("got attempts %d, want 1", nextBatch[0].Attempts)
	}
}

// TestOutboxConcurrentWorkersNeverClaimTheSameRow is the direct proof of
// spec section 21's "múltiplos workers processando o mesmo evento" concern:
// two goroutines racing FetchPendingBatch against the same pending rows
// must partition them, never double-claim.
func TestOutboxConcurrentWorkersNeverClaimTheSameRow(t *testing.T) {
	pool := newTestPool(t)
	repo := outboxdb.NewRepository(pool)
	ctx := context.Background()

	const total = 20
	for i := 0; i < total; i++ {
		if err := repo.Insert(ctx, output.OutboxEvent{
			ID: fmt.Sprintf("concurrent-%02d", i), EventType: "TestEvent", Payload: []byte(`{}`),
			Status: output.OutboxStatusPending, CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	var (
		mu   sync.Mutex
		seen = make(map[string]int)
		wg   sync.WaitGroup
	)

	claim := func() {
		defer wg.Done()
		batch, err := repo.FetchPendingBatch(ctx, total)
		if err != nil {
			t.Errorf("fetch pending batch: %v", err)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for _, evt := range batch {
			seen[evt.ID]++
		}
	}

	wg.Add(2)
	go claim()
	go claim()
	wg.Wait()

	if len(seen) != total {
		t.Fatalf("got %d unique claimed events across both workers, want %d", len(seen), total)
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("event %s was claimed %d times, want exactly 1", id, count)
		}
	}
}
