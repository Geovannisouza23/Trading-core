package output

import (
	"context"
	"time"
)

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "PENDING"
	OutboxStatusProcessing OutboxStatus = "PROCESSING"
	OutboxStatusProcessed  OutboxStatus = "PROCESSED"
	OutboxStatusFailed     OutboxStatus = "FAILED"
)

// OutboxEvent is a row in the transactional outbox table.
type OutboxEvent struct {
	ID          string
	EventType   string
	Payload     []byte
	Status      OutboxStatus
	Attempts    int
	CreatedAt   time.Time
	ProcessedAt *time.Time
	LastError   string
}

// OutboxRepository persists domain events transactionally alongside the
// aggregate that produced them, and lets the outbox worker drain them.
type OutboxRepository interface {
	// Insert must be called from within the same transaction as the
	// aggregate write it accompanies (via TransactionManager.WithinTransaction).
	Insert(ctx context.Context, evt OutboxEvent) error
	// FetchPendingBatch locks up to `limit` pending rows (SELECT ... FOR
	// UPDATE SKIP LOCKED) so concurrent workers never process the same row.
	FetchPendingBatch(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkProcessed(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, errMsg string) error
}
