package position

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	domainposition "trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.PositionRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.PositionRepository = (*Repository)(nil)

func scanRow(row pgx.Row) (*domainposition.Position, error) {
	var m model
	if err := row.Scan(
		&m.ID, &m.Symbol, &m.Side, &m.Quantity, &m.EntryPrice, &m.CurrentPrice, &m.StopPrice, &m.TargetPrice,
		&m.UnrealizedPnL, &m.RealizedPnL, &m.Status, &m.Version, &m.OpenedAt, &m.ClosedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) GetByID(ctx context.Context, id shared.PositionID) (*domainposition.Position, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getByIDQuery, id.String()))
}

func (r *Repository) GetOpenBySymbol(ctx context.Context, symbol shared.Symbol) (*domainposition.Position, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getOpenBySymbolQuery, symbol.String()))
}

func (r *Repository) ListOpen(ctx context.Context) ([]domainposition.Position, error) {
	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, listOpenQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domainposition.Position
	for rows.Next() {
		p, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *p)
	}
	return results, rows.Err()
}

func (r *Repository) Save(ctx context.Context, p *domainposition.Position) error {
	m := toModel(p)
	expectedPreviousVersion := m.Version - 1
	tag, err := transaction.Executor(ctx, r.pool).Exec(ctx, upsertQuery,
		m.ID, m.Symbol, m.Side, m.Quantity, m.EntryPrice, m.CurrentPrice, m.StopPrice, m.TargetPrice,
		m.UnrealizedPnL, m.RealizedPnL, m.Status, m.Version, m.OpenedAt, m.ClosedAt,
		expectedPreviousVersion,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: position %s at version %d", output.ErrOptimisticLock, p.ID, expectedPreviousVersion)
	}
	return nil
}
