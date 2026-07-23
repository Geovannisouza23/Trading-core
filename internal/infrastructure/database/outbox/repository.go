package outbox

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/infrastructure/database/transaction"
)

const maxAttempts = 5

// Repository implements output.OutboxRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.OutboxRepository = (*Repository)(nil)

func (r *Repository) Insert(ctx context.Context, evt output.OutboxEvent) error {
	_, err := transaction.Executor(ctx, r.pool).Exec(ctx, insertQuery, evt.ID, evt.EventType, evt.Payload, string(output.OutboxStatusPending), time.Now().UTC())
	return err
}

func (r *Repository) FetchPendingBatch(ctx context.Context, limit int) ([]output.OutboxEvent, error) {
	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, fetchPendingBatchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []output.OutboxEvent
	for rows.Next() {
		var m model
		if err := rows.Scan(&m.ID, &m.EventType, &m.Payload, &m.Status, &m.Attempts, &m.CreatedAt, &m.ProcessedAt, &m.LastError); err != nil {
			return nil, err
		}
		events = append(events, toDomain(m))
	}
	return events, rows.Err()
}

func (r *Repository) MarkProcessed(ctx context.Context, id string) error {
	_, err := transaction.Executor(ctx, r.pool).Exec(ctx, markProcessedQuery, id, time.Now().UTC())
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id string, errMsg string) error {
	_, err := transaction.Executor(ctx, r.pool).Exec(ctx, markFailedQuery, id, errMsg, maxAttempts)
	return err
}
