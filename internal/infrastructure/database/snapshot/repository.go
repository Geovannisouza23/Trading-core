package snapshot

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/account"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.AccountSnapshotRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.AccountSnapshotRepository = (*Repository)(nil)

func (r *Repository) Latest(ctx context.Context) (*account.Snapshot, error) {
	row := transaction.Executor(ctx, r.pool).QueryRow(ctx, latestQuery)
	var m model
	if err := row.Scan(&m.ID, &m.Balance, &m.Equity, &m.Positions, &m.OpenOrders, &m.DailyPnL, &m.Drawdown, &m.Timestamp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) Save(ctx context.Context, s *account.Snapshot) error {
	m, err := toModel(s)
	if err != nil {
		return err
	}
	_, err = transaction.Executor(ctx, r.pool).Exec(ctx, insertQuery,
		m.ID, m.Balance, m.Equity, m.Positions, m.OpenOrders, m.DailyPnL, m.Drawdown, m.Timestamp,
	)
	return err
}
