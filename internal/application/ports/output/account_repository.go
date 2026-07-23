package output

import (
	"context"

	"trading-core/internal/domain/account"
	"trading-core/internal/domain/shared"
)

// AccountRepository persists the Account aggregate. Save must enforce
// optimistic locking on Account.Version.
type AccountRepository interface {
	GetByID(ctx context.Context, id shared.AccountID) (*account.Account, error)
	// GetActive returns the single account currently in use by the running
	// mode. The MVP operates a single account per deployment.
	GetActive(ctx context.Context) (*account.Account, error)
	Save(ctx context.Context, acc *account.Account) error
}
