package event

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	domainevent "trading-core/internal/domain/event"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.MarketEventRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.MarketEventRepository = (*Repository)(nil)

func scanRow(row pgx.Row) (*domainevent.MarketEvent, error) {
	var m model
	if err := row.Scan(&m.ID, &m.EventType, &m.Direction, &m.Severity, &m.Confidence, &m.AffectedAssets, &m.Action, &m.SourceCount, &m.HasOfficialSource, &m.PublishedAt, &m.DetectedAt, &m.ExpiresAt, &m.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) GetByID(ctx context.Context, id shared.MarketEventID) (*domainevent.MarketEvent, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getByIDQuery, id.String()))
}

func (r *Repository) listQuery(ctx context.Context, query string, args ...any) ([]domainevent.MarketEvent, error) {
	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domainevent.MarketEvent
	for rows.Next() {
		e, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *e)
	}
	return results, rows.Err()
}

func (r *Repository) ListActive(ctx context.Context) ([]domainevent.MarketEvent, error) {
	return r.listQuery(ctx, listActiveQuery)
}

func (r *Repository) List(ctx context.Context, limit int) ([]domainevent.MarketEvent, error) {
	return r.listQuery(ctx, listQuery, limit)
}

func (r *Repository) Save(ctx context.Context, e *domainevent.MarketEvent) error {
	m := toModel(e)
	_, err := transaction.Executor(ctx, r.pool).Exec(ctx, insertQuery,
		m.ID, m.EventType, m.Direction, m.Severity, m.Confidence, m.AffectedAssets, m.Action, m.SourceCount, m.HasOfficialSource, m.PublishedAt, m.DetectedAt, m.ExpiresAt, m.Status,
	)
	return err
}
