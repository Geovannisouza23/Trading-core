package output

import (
	"context"

	"trading-core/internal/domain/account"
)

// AccountSnapshotRepository persists periodic Account.Snapshot read models.
type AccountSnapshotRepository interface {
	Save(ctx context.Context, s *account.Snapshot) error
	Latest(ctx context.Context) (*account.Snapshot, error)
}
