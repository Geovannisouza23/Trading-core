package app

import (
	"log/slog"
	"time"

	"trading-core/internal/application/ports/output"
	outboxdb "trading-core/internal/infrastructure/database/outbox"
)

func provideOutboxPublisher(bus output.EventBus) *outboxdb.Publisher {
	return outboxdb.NewPublisher(bus)
}

func provideOutboxWorker(repo output.OutboxRepository, publisher *outboxdb.Publisher, logger *slog.Logger) *outboxdb.Worker {
	const (
		drainInterval = 2 * time.Second
		batchSize     = 20
	)
	return outboxdb.NewWorker(repo, publisher, drainInterval, batchSize, logger)
}
