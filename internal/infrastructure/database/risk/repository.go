package risk

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	domainrisk "trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.RiskDecisionRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.RiskDecisionRepository = (*Repository)(nil)

func scanRow(row pgx.Row) (*domainrisk.Decision, error) {
	var m model
	if err := row.Scan(&m.ID, &m.SignalID, &m.Allowed, &m.OriginalPositionSize, &m.ApprovedPositionSize, &m.ReasonCodes, &m.AppliedRules, &m.EventRestrictions, &m.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) GetByID(ctx context.Context, id shared.RiskDecisionID) (*domainrisk.Decision, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getByIDQuery, id.String()))
}

func (r *Repository) List(ctx context.Context, limit int) ([]domainrisk.Decision, error) {
	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, listQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domainrisk.Decision
	for rows.Next() {
		d, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *d)
	}
	return results, rows.Err()
}

func (r *Repository) Save(ctx context.Context, d *domainrisk.Decision) error {
	m := toModel(d)
	_, err := transaction.Executor(ctx, r.pool).Exec(ctx, insertQuery,
		m.ID, m.SignalID, m.Allowed, m.OriginalPositionSize, m.ApprovedPositionSize, m.ReasonCodes, m.AppliedRules, m.EventRestrictions, m.CreatedAt,
	)
	return err
}
