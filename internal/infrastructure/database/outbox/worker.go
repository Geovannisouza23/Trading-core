package outbox

import (
	"context"
	"log/slog"
	"time"

	"trading-core/internal/application/ports/output"
)

// Worker periodically drains claimed outbox rows and publishes them to the
// EventBus. Failures are retried (via MarkFailed's attempts counter) up to
// a fixed limit before the row is parked as FAILED.
type Worker struct {
	repo      output.OutboxRepository
	publisher *Publisher
	interval  time.Duration
	batchSize int
	logger    *slog.Logger
}

func NewWorker(repo output.OutboxRepository, publisher *Publisher, interval time.Duration, batchSize int, logger *slog.Logger) *Worker {
	return &Worker{repo: repo, publisher: publisher, interval: interval, batchSize: batchSize, logger: logger}
}

// Run blocks, draining the outbox on every tick, until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.DrainOnce(ctx)
		}
	}
}

// DrainOnce processes a single batch; exported so tests and a manual admin
// trigger can invoke it directly without waiting for the ticker.
func (w *Worker) DrainOnce(ctx context.Context) {
	events, err := w.repo.FetchPendingBatch(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("outbox: fetching pending batch failed", "error", err)
		return
	}
	for _, evt := range events {
		if err := w.publisher.Publish(ctx, evt); err != nil {
			w.logger.Error("outbox: publishing event failed", "event_id", evt.ID, "event_type", evt.EventType, "error", err)
			if markErr := w.repo.MarkFailed(ctx, evt.ID, err.Error()); markErr != nil {
				w.logger.Error("outbox: marking event failed", "event_id", evt.ID, "error", markErr)
			}
			continue
		}
		if err := w.repo.MarkProcessed(ctx, evt.ID); err != nil {
			w.logger.Error("outbox: marking event processed failed", "event_id", evt.ID, "error", err)
		}
	}
}
