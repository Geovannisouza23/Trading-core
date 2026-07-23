package output

import "context"

// TransactionManager runs fn within a single database transaction. The
// transaction handle travels through ctx (set by the concrete
// implementation), so repositories never receive it as an explicit
// parameter and use cases never import pgx.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
