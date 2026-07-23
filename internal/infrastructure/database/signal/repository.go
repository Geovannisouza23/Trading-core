package signal

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/shared"
	domainsignal "trading-core/internal/domain/signal"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.TradeSignalRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.TradeSignalRepository = (*Repository)(nil)

func scanRow(row pgx.Row) (*domainsignal.TradeSignal, error) {
	var m model
	if err := row.Scan(&m.ID, &m.Symbol, &m.Side, &m.EntryPrice, &m.StopPrice, &m.TargetPrice, &m.Confidence, &m.StrategyName, &m.MarketRegime, &m.CreatedAt, &m.ValidUntil); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) GetByID(ctx context.Context, id shared.SignalID) (*domainsignal.TradeSignal, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getByIDQuery, id.String()))
}

func (r *Repository) List(ctx context.Context, limit int) ([]domainsignal.TradeSignal, error) {
	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, listQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domainsignal.TradeSignal
	for rows.Next() {
		s, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *s)
	}
	return results, rows.Err()
}

func (r *Repository) Save(ctx context.Context, s *domainsignal.TradeSignal) error {
	m := toModel(s)
	_, err := transaction.Executor(ctx, r.pool).Exec(ctx, insertQuery,
		m.ID, m.Symbol, m.Side, m.EntryPrice, m.StopPrice, m.TargetPrice, m.Confidence, m.StrategyName, m.MarketRegime, m.CreatedAt, m.ValidUntil,
	)
	return err
}
