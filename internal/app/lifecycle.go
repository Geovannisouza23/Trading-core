package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.uber.org/fx"

	"trading-core/internal/application/command"
	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/config"
	"trading-core/internal/domain/account"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/shared"

	httpiface "trading-core/internal/interfaces/http"

	marketconsumer "trading-core/internal/interfaces/consumer/market"
	orderconsumer "trading-core/internal/interfaces/consumer/order"
	outboxconsumer "trading-core/internal/interfaces/consumer/outbox"

	outboxdb "trading-core/internal/infrastructure/database/outbox"
	marketdatawebsocket "trading-core/internal/infrastructure/external/marketdata/websocket"
)

// bootstrap ensures the single PAPER account and the operational_modes
// singleton row exist before anything else runs. It is idempotent: on
// every subsequent restart both lookups succeed and it does nothing.
func bootstrap(lc fx.Lifecycle, accounts output.AccountRepository, modes output.OperationalModeRepository, clock output.Clock, cfg *config.Config, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			now := clock.Now()

			if _, err := accounts.GetActive(ctx); err != nil {
				if !errors.Is(err, output.ErrNotFound) {
					return err
				}
				acct, err := account.NewPaperAccount(shared.NewMoney(cfg.Broker.PaperStartingBalance), now)
				if err != nil {
					return err
				}
				if err := accounts.Save(ctx, acct); err != nil {
					return err
				}
				logger.Info("bootstrap: created initial PAPER account", "starting_balance", cfg.Broker.PaperStartingBalance.String())
			}

			if _, err := modes.Get(ctx); err != nil {
				if !errors.Is(err, output.ErrNotFound) {
					return err
				}
				state := operation.NewState(now)
				if err := modes.Save(ctx, state); err != nil {
					return err
				}
				logger.Info("bootstrap: initialized operational mode", "mode", state.CurrentMode)
			}

			if cfg.UsingInsecureDefaultJWTSecret {
				logger.Warn("SECURITY_JWT_SIGNING_SECRET is not set; using an insecure development default. Set it explicitly outside local PAPER development.")
			}

			return nil
		},
	})
}

// startHTTPServer runs the HTTP server in the background and shuts it down
// gracefully when the app stops.
func startHTTPServer(lc fx.Lifecycle, server *httpiface.Server, cfg *config.Config, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("http server stopped unexpectedly", "error", err)
				}
			}()
			logger.Info("http server listening", "host", cfg.Server.Host, "port", cfg.Server.Port)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
			defer cancel()
			return server.Shutdown(shutdownCtx)
		},
	})
}

// subscribeConsumers wires the order and outbox consumers (both
// EventBus-driven) once at startup. The market and event consumers are
// invoked directly by their respective feeds/pollers instead of through
// the bus.
func subscribeConsumers(lc fx.Lifecycle, bus output.EventBus, orderConsumer *orderconsumer.Consumer, outboxConsumer *outboxconsumer.Consumer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			orderConsumer.Subscribe(bus)
			outboxConsumer.Subscribe(bus)
			return nil
		},
	})
}

// startOutboxWorker drains the transactional outbox on a fixed interval.
func startOutboxWorker(lc fx.Lifecycle, worker *outboxdb.Worker) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			go worker.Run(ctx)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			return nil
		},
	})
}

// startReconciliationScheduler runs ReconcileBrokerState on a fixed
// interval, in every operational mode (spec section 11.4: reconciliation
// never stops).
func startReconciliationScheduler(lc fx.Lifecycle, reconcile input.ReconcileBrokerStateUseCase, logger *slog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			go func() {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						report, err := reconcile.Execute(ctx, command.ReconcileBrokerStateCommand{Reason: "scheduled"})
						if err != nil {
							logger.Error("reconciliation failed", "error", err)
							continue
						}
						if report.DivergencesFound > 0 {
							logger.Warn("reconciliation found divergences", "count", report.DivergencesFound, "critical", report.CriticalDivergence)
						}
					}
				}
			}()
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			return nil
		},
	})
}

// startMarketDataFeed connects to Binance's public futures kline stream
// (no API key required) and drives every closed candle through the
// market consumer's signal -> risk -> execution pipeline. Connection
// failures (e.g. no outbound network access in a sandboxed environment)
// are logged and retried with backoff; they never crash the process or
// block any other part of the system.
func startMarketDataFeed(lc fx.Lifecycle, consumer *marketconsumer.Consumer, logger *slog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			go runMarketDataFeed(ctx, consumer, logger)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			return nil
		},
	})
}

const (
	defaultFeedSymbol    = "BTCUSDT"
	defaultFeedTimeframe = "1m"
	defaultFeedBaseURL   = "wss://fstream.binance.com"
	maxRecentCandles     = 50
)

func runMarketDataFeed(ctx context.Context, consumer *marketconsumer.Consumer, logger *slog.Logger) {
	symbol := shared.MustNewSymbol(defaultFeedSymbol)
	timeframe := shared.MustNewTimeframe(defaultFeedTimeframe)
	client := marketdatawebsocket.NewClient(defaultFeedBaseURL)

	var recentCandles []market.Candle
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		err := client.Stream(ctx, symbol, timeframe, func(candle market.Candle) {
			consumer.HandleCandle(ctx, command.EvaluateMarketSignalCommand{
				Candle:        candle,
				RecentCandles: append([]market.Candle(nil), recentCandles...),
			})
			recentCandles = append(recentCandles, candle)
			if len(recentCandles) > maxRecentCandles {
				recentCandles = recentCandles[len(recentCandles)-maxRecentCandles:]
			}
			backoff = time.Second
		})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			logger.Warn("market data feed disconnected, retrying", "symbol", defaultFeedSymbol, "error", err, "backoff", backoff)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}
