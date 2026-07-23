package operation

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/output"
	domainoperation "trading-core/internal/domain/operation"
	"trading-core/internal/infrastructure/database/transaction"
)

// Repository implements output.OperationalModeRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ output.OperationalModeRepository = (*Repository)(nil)

func (r *Repository) Get(ctx context.Context) (*domainoperation.State, error) {
	row := transaction.Executor(ctx, r.pool).QueryRow(ctx, getQuery)
	var m model
	if err := row.Scan(&m.CurrentMode, &m.ChangedBy, &m.Origin, &m.Reason, &m.ChangedAt, &m.Version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, output.ErrNotFound
		}
		return nil, err
	}
	return toDomain(m), nil
}

func (r *Repository) Save(ctx context.Context, s *domainoperation.State) error {
	m := toModel(s)
	expectedPreviousVersion := m.Version - 1
	tag, err := transaction.Executor(ctx, r.pool).Exec(ctx, upsertQuery,
		m.CurrentMode, m.ChangedBy, m.Origin, m.Reason, m.ChangedAt, m.Version, expectedPreviousVersion,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: operational mode at version %d", output.ErrOptimisticLock, expectedPreviousVersion)
	}
	return nil
}
