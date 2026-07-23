package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"trading-core/internal/config"
)

// Module is the only file in this package allowed to import Uber Fx (spec
// section 25: Fx is confined to cmd, internal/app, and composition-only
// module files). It wires the pgx pool's lifecycle to the application's.
func Module() fx.Option {
	return fx.Options(
		fx.Provide(func(lc fx.Lifecycle, cfg *config.Config) (*pgxpool.Pool, error) {
			pool, err := NewPool(context.Background(), cfg.Database)
			if err != nil {
				return nil, err
			}
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					pool.Close()
					return nil
				},
			})
			return pool, nil
		}),
	)
}
