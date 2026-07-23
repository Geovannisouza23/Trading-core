// Package transaction implements output.TransactionManager against pgx: it
// threads an active pgx.Tx through context.Context so every repository call
// made inside WithinTransaction automatically joins the same transaction,
// without repositories ever receiving a *pgx.Tx parameter directly.
package transaction

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey struct{}

// Manager implements output.TransactionManager.
type Manager struct {
	pool *pgxpool.Pool
}

func NewManager(pool *pgxpool.Pool) *Manager {
	return &Manager{pool: pool}
}

// WithinTransaction runs fn inside a single pgx transaction. Nested calls
// (fn itself calling WithinTransaction again, directly or through another
// use case) reuse the already-open transaction instead of nesting a new
// one, since pgx does not support true nested transactions.
func (m *Manager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, alreadyInTx := ctx.Value(contextKey{}).(pgx.Tx); alreadyInTx {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txCtx := context.WithValue(ctx, contextKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Executor returns the transaction active on ctx, if any, otherwise the
// pool itself. Every repository's query methods call this first.
func Executor(ctx context.Context, pool *pgxpool.Pool) Queryer {
	if tx, ok := ctx.Value(contextKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// Queryer is the minimal pgx surface repositories need; both *pgxpool.Pool
// and pgx.Tx satisfy it.
type Queryer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
