package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainaccount "trading-core/internal/domain/account"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.AccountRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.AccountRepository = (*Repository)(nil)

func (r *Repository) scanOne(ctx context.Context, query string, args ...any) (*domainaccount.Account, error) {
	row := transaction.Executor(ctx, r.pool).QueryRow(ctx, query, args...)
	var m model
	if err := row.Scan(&m.ID, &m.Balance, &m.Equity, &m.AvailableBalance, &m.PeakEquity, &m.DailyPnL, &m.WeeklyPnL, &m.CurrentDrawdown, &m.OperationalMode, &m.Version, &m.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) GetByID(ctx context.Context, id shared.AccountID) (*domainaccount.Account, error) {
	return r.scanOne(ctx, getByIDQuery, id.String())
}

func (r *Repository) GetActive(ctx context.Context) (*domainaccount.Account, error) {
	return r.scanOne(ctx, getActiveQuery)
}

func (r *Repository) Save(ctx context.Context, a *domainaccount.Account) error {
	m := toModel(a)
	expectedPreviousVersion := m.Version - 1
	tag, err := transaction.Executor(ctx, r.pool).Exec(ctx, upsertQuery,
		m.ID, m.Balance, m.Equity, m.AvailableBalance, m.PeakEquity, m.DailyPnL, m.WeeklyPnL,
		m.CurrentDrawdown, m.OperationalMode, m.Version, m.UpdatedAt, expectedPreviousVersion,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: account %s at version %d", output.ErrOptimisticLock, a.ID, expectedPreviousVersion)
	}
	return nil
}
