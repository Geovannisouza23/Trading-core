package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	domainorder "trading-core/internal/domain/order"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.OrderRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.OrderRepository = (*Repository)(nil)

func scanRow(row pgx.Row) (*domainorder.Order, error) {
	var m model
	if err := row.Scan(
		&m.ID, &m.ClientOrderID, &m.BrokerOrderID, &m.Symbol, &m.Side, &m.Type, &m.Quantity, &m.FilledQuantity,
		&m.RequestedPrice, &m.AverageExecutionPrice, &m.StopPrice, &m.TargetPrice, &m.Status, &m.StrategyName,
		&m.SignalID, &m.RiskDecisionID, &m.FailureReason, &m.Version, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m)
}

func (r *Repository) GetByID(ctx context.Context, id shared.OrderID) (*domainorder.Order, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getByIDQuery, id.String()))
}

func (r *Repository) GetByClientOrderID(ctx context.Context, clientOrderID shared.ClientOrderID) (*domainorder.Order, error) {
	return scanRow(transaction.Executor(ctx, r.pool).QueryRow(ctx, getByClientOrderIDQuery, clientOrderID.String()))
}

func (r *Repository) listQuery(ctx context.Context, query string, args ...any) ([]domainorder.Order, error) {
	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domainorder.Order
	for rows.Next() {
		o, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *o)
	}
	return results, rows.Err()
}

func (r *Repository) ListOpen(ctx context.Context) ([]domainorder.Order, error) {
	return r.listQuery(ctx, listOpenQuery)
}

func (r *Repository) ListBySignalID(ctx context.Context, signalID shared.SignalID) ([]domainorder.Order, error) {
	return r.listQuery(ctx, listBySignalIDQuery, signalID.String())
}

func (r *Repository) List(ctx context.Context, limit int) ([]domainorder.Order, error) {
	return r.listQuery(ctx, listQuery, limit)
}

func (r *Repository) Save(ctx context.Context, o *domainorder.Order) error {
	m := toModel(o)
	expectedPreviousVersion := m.Version - 1
	executor := transaction.Executor(ctx, r.pool)

	tag, err := executor.Exec(ctx, upsertQuery,
		m.ID, m.ClientOrderID, m.BrokerOrderID, m.Symbol, m.Side, m.Type, m.Quantity, m.FilledQuantity,
		m.RequestedPrice, m.AverageExecutionPrice, m.StopPrice, m.TargetPrice, m.Status, m.StrategyName,
		m.SignalID, m.RiskDecisionID, m.FailureReason, m.Version, m.CreatedAt, m.UpdatedAt,
		expectedPreviousVersion,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: order %s at version %d", output.ErrOptimisticLock, o.ID, expectedPreviousVersion)
	}

	if (o.Status == domainorder.StatusFilled || o.Status == domainorder.StatusPartiallyFilled) && o.AverageExecutionPrice != nil {
		if _, err := executor.Exec(ctx, insertExecutionQuery,
			uuid.NewString(), o.ID.String(), o.FilledQuantity.Decimal(), o.AverageExecutionPrice.Decimal(), time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("recording execution audit row: %w", err)
		}
	}

	return nil
}
