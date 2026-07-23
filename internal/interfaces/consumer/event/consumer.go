// Package event receives raw news items (from a scheduled feed poll) and
// calls ProcessMarketEvent. It only translates the payload and calls the
// use case — sanitization, deduplication, and every classification rule
// live inside application/usecase.ProcessMarketEvent.
package event

import (
	"context"
	"log/slog"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
)

type Consumer struct {
	processEvent input.ProcessMarketEventUseCase
	logger       *slog.Logger
}

func NewConsumer(processEvent input.ProcessMarketEventUseCase, logger *slog.Logger) *Consumer {
	return &Consumer{processEvent: processEvent, logger: logger}
}

// HandleNewsItem converts a raw news item into a command and calls
// ProcessMarketEvent. Errors are logged, not propagated: one bad/duplicate
// item must never stop the poller from processing the rest of the batch.
func (c *Consumer) HandleNewsItem(ctx context.Context, cmd command.ProcessMarketEventCommand) {
	result, err := c.processEvent.Execute(ctx, cmd)
	if err != nil {
		c.logger.Warn("event consumer: processing news item failed", "url", cmd.URL, "error", err)
		return
	}
	if result.Deduplicated {
		c.logger.Debug("event consumer: duplicate news item skipped", "url", cmd.URL)
	}
}
