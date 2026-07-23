package usecase

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"

	"trading-core/internal/application/command"
	"trading-core/internal/application/mapper"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/event"
)

// ProcessMarketEvent implements input.ProcessMarketEventUseCase. It is
// deliberately conservative: the LLM only classifies the event. Nothing in
// this use case places, sizes or approves an order, and a KILL_SWITCH
// classification only raises a critical incident and notification — it
// never calls ActivateKillSwitch automatically.
type ProcessMarketEvent struct {
	events          output.MarketEventRepository
	intelligence    output.EventIntelligence
	idempotent      output.IdempotencyRepository
	outbox          output.OutboxRepository
	tx              output.TransactionManager
	notifier        output.Notifier
	clock           output.Clock
	allowedSources  []string
	maxContentBytes int
	maxNewsAge      time.Duration
	dedupTTL        time.Duration
}

func NewProcessMarketEvent(
	events output.MarketEventRepository,
	intelligence output.EventIntelligence,
	idempotent output.IdempotencyRepository,
	outbox output.OutboxRepository,
	tx output.TransactionManager,
	notifier output.Notifier,
	clock output.Clock,
	allowedSources []string,
	maxContentBytes int,
	maxNewsAge time.Duration,
) *ProcessMarketEvent {
	return &ProcessMarketEvent{
		events:          events,
		intelligence:    intelligence,
		idempotent:      idempotent,
		outbox:          outbox,
		tx:              tx,
		notifier:        notifier,
		clock:           clock,
		allowedSources:  allowedSources,
		maxContentBytes: maxContentBytes,
		maxNewsAge:      maxNewsAge,
		dedupTTL:        48 * time.Hour,
	}
}

var _ input.ProcessMarketEventUseCase = (*ProcessMarketEvent)(nil)

func (uc *ProcessMarketEvent) Execute(ctx context.Context, cmd command.ProcessMarketEventCommand) (input.ProcessMarketEventResult, error) {
	now := uc.clock.Now()

	if !uc.sourceAllowed(cmd.Source) {
		return input.ProcessMarketEventResult{}, fmt.Errorf("source %q is not in the allowlist", cmd.Source)
	}
	parsedURL, err := url.Parse(cmd.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return input.ProcessMarketEventResult{}, fmt.Errorf("invalid source URL %q", cmd.URL)
	}
	if len(cmd.Content) > uc.maxContentBytes {
		return input.ProcessMarketEventResult{}, fmt.Errorf("content exceeds maximum allowed size of %d bytes", uc.maxContentBytes)
	}

	sanitizedTitle := sanitizeText(cmd.Title)
	sanitizedContent := sanitizeText(cmd.Content)

	reserved, err := uc.idempotent.Reserve(ctx, "market_event", cmd.URL, uc.dedupTTL)
	if err != nil {
		return input.ProcessMarketEventResult{}, fmt.Errorf("checking deduplication: %w", err)
	}
	if !reserved {
		return input.ProcessMarketEventResult{Deduplicated: true}, nil
	}

	if now.Sub(cmd.PublishedAt) > uc.maxNewsAge {
		return input.ProcessMarketEventResult{}, fmt.Errorf("news item is older than the maximum allowed age of %s", uc.maxNewsAge)
	}

	assessment, err := uc.intelligence.Analyze(ctx, output.NewsAnalysisInput{
		Source:      cmd.Source,
		URL:         cmd.URL,
		Title:       sanitizedTitle,
		Content:     sanitizedContent,
		PublishedAt: cmd.PublishedAt,
	})
	if err != nil {
		return input.ProcessMarketEventResult{}, fmt.Errorf("event intelligence analysis failed: %w", err)
	}

	marketEvent, err := event.New(
		assessment.EventType,
		assessment.Direction,
		assessment.Severity,
		assessment.Confidence,
		assessment.AffectedAssets,
		assessment.Action,
		assessment.SourceCount,
		assessment.HasOfficialSource,
		cmd.PublishedAt,
		now,
		now.Add(uc.maxNewsAge),
	)
	if err != nil {
		return input.ProcessMarketEventResult{}, fmt.Errorf("event intelligence returned an invalid assessment: %w", err)
	}

	if marketEvent.IsCritical() {
		if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
			if err := uc.events.Save(ctx, marketEvent); err != nil {
				return err
			}
			evt, err := newOutboxEvent("CriticalEventDetected", event.CriticalEventDetected{Event: marketEvent, OccurredAt_: now}, now)
			if err != nil {
				return err
			}
			return uc.outbox.Insert(ctx, evt)
		}); err != nil {
			return input.ProcessMarketEventResult{}, fmt.Errorf("persisting critical market event: %w", err)
		}
		_ = uc.notifier.Notify(ctx, output.Notification{
			Title:    "Critical market event detected",
			Message:  fmt.Sprintf("%s affecting %v: action=%s severity=%s", marketEvent.EventType, marketEvent.AffectedAssets, marketEvent.Action, marketEvent.Severity),
			Severity: output.NotificationCritical,
		})
	} else if err := uc.events.Save(ctx, marketEvent); err != nil {
		return input.ProcessMarketEventResult{}, fmt.Errorf("persisting market event: %w", err)
	}

	dto := mapper.ToMarketEventDTO(marketEvent)
	return input.ProcessMarketEventResult{Event: &dto}, nil
}

func (uc *ProcessMarketEvent) sourceAllowed(source string) bool {
	normalized := strings.ToLower(strings.TrimSpace(source))
	for _, allowed := range uc.allowedSources {
		if strings.ToLower(allowed) == normalized {
			return true
		}
	}
	return false
}

// sanitizeText strips control/non-printable characters and collapses
// surrounding whitespace, providing basic protection against prompt
// injection payloads riding along in scraped news content.
func sanitizeText(input string) string {
	var b strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
