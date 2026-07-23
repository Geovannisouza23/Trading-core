package incident

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.SystemIncidentRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.SystemIncidentRepository = (*Repository)(nil)

func scanRow(row pgx.Row) (*operation.Incident, error) {
	var m model
	if err := row.Scan(&m.ID, &m.Type, &m.Severity, &m.Description, &m.Source, &m.RelatedEntityID, &m.Status, &m.DetectedAt, &m.ResolvedAt, &m.Resolution, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) GetByID(ctx context.Context, id shared.IncidentID) (*operation.Incident, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getByIDQuery, id.String()))
}

func (r *Repository) listQuery(ctx context.Context, query string, args ...any) ([]operation.Incident, error) {
	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []operation.Incident
	for rows.Next() {
		i, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *i)
	}
	return results, rows.Err()
}

func (r *Repository) ListOpen(ctx context.Context) ([]operation.Incident, error) {
	return r.listQuery(ctx, listOpenQuery)
}

func (r *Repository) List(ctx context.Context, limit int) ([]operation.Incident, error) {
	return r.listQuery(ctx, listQuery, limit)
}

func (r *Repository) Save(ctx context.Context, i *operation.Incident) error {
	m := toModel(i)
	_, err := transaction.Executor(ctx, r.pool).Exec(ctx, upsertQuery,
		m.ID, m.Type, m.Severity, m.Description, m.Source, m.RelatedEntityID, m.Status, m.DetectedAt, m.ResolvedAt, m.Resolution, m.CreatedAt, m.UpdatedAt,
	)
	return err
}
