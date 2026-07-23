package app

import (
	"context"
	"log/slog"

	"go.uber.org/fx"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/config"
	"trading-core/internal/infrastructure/database"
	"trading-core/internal/infrastructure/database/transaction"
	"trading-core/internal/infrastructure/observability/logging"
	"trading-core/internal/infrastructure/observability/metrics"
	"trading-core/internal/infrastructure/observability/tracing"

	systemclock "trading-core/internal/infrastructure/clock/system"

	accountdb "trading-core/internal/infrastructure/database/account"
	eventdb "trading-core/internal/infrastructure/database/event"
	idempotencydb "trading-core/internal/infrastructure/database/idempotency"
	incidentdb "trading-core/internal/infrastructure/database/incident"
	operationdb "trading-core/internal/infrastructure/database/operation"
	orderdb "trading-core/internal/infrastructure/database/order"
	outboxdb "trading-core/internal/infrastructure/database/outbox"
	positiondb "trading-core/internal/infrastructure/database/position"
	riskdb "trading-core/internal/infrastructure/database/risk"
	signaldb "trading-core/internal/infrastructure/database/signal"
	snapshotdb "trading-core/internal/infrastructure/database/snapshot"
)

// ConfigModule loads and validates configuration once for the whole graph.
func ConfigModule() fx.Option {
	return fx.Provide(config.Load)
}

// ObservabilityModule provides the logger, metrics and tracer provider.
func ObservabilityModule() fx.Option {
	return fx.Options(
		fx.Provide(func(cfg *config.Config) *slog.Logger {
			return logging.NewLogger(cfg.Observability)
		}),
		fx.Provide(func(cfg *config.Config) (*metrics.Metrics, error) {
			return metrics.NewMetrics(cfg.Observability.ServiceName)
		}),
		fx.Provide(func(lc fx.Lifecycle, cfg *config.Config) (*sdktrace.TracerProvider, error) {
			tp, err := tracing.NewTracerProvider(context.Background(), cfg.Observability)
			if err != nil {
				return nil, err
			}
			lc.Append(fx.Hook{OnStop: func(ctx context.Context) error { return tp.Shutdown(ctx) }})
			return tp, nil
		}),
		fx.Provide(fx.Annotate(systemclock.NewClock, fx.As(new(output.Clock)))),
	)
}

// DatabaseModule provides the pgx pool (lifecycle-managed) and the
// transaction manager built on top of it.
func DatabaseModule() fx.Option {
	return fx.Options(
		database.Module(),
		fx.Provide(fx.Annotate(transaction.NewManager, fx.As(new(output.TransactionManager)))),
	)
}

// RepositoryModule binds one PostgreSQL repository implementation per
// domain aggregate to its application/ports/output interface.
func RepositoryModule() fx.Option {
	return fx.Provide(
		fx.Annotate(accountdb.NewRepository, fx.As(new(output.AccountRepository))),
		fx.Annotate(orderdb.NewRepository, fx.As(new(output.OrderRepository))),
		fx.Annotate(positiondb.NewRepository, fx.As(new(output.PositionRepository))),
		fx.Annotate(signaldb.NewRepository, fx.As(new(output.TradeSignalRepository))),
		fx.Annotate(riskdb.NewRepository, fx.As(new(output.RiskDecisionRepository))),
		fx.Annotate(eventdb.NewRepository, fx.As(new(output.MarketEventRepository))),
		fx.Annotate(snapshotdb.NewRepository, fx.As(new(output.AccountSnapshotRepository))),
		fx.Annotate(incidentdb.NewRepository, fx.As(new(output.SystemIncidentRepository))),
		fx.Annotate(operationdb.NewRepository, fx.As(new(output.OperationalModeRepository))),
		fx.Annotate(idempotencydb.NewRepository, fx.As(new(output.IdempotencyRepository))),
		fx.Annotate(outboxdb.NewRepository, fx.As(new(output.OutboxRepository))),
	)
}

// ExternalModule provides every mode-selected external adapter (Factory
// pattern): broker, quant engine, event intelligence, notifier, event bus.
func ExternalModule() fx.Option {
	return fx.Provide(
		provideBroker,
		provideQuantEngine,
		provideEventIntelligence,
		provideNotifier,
		provideEventBus,
	)
}

// ApplicationModule provides every use case and the risk-policy wiring
// they share.
func ApplicationModule() fx.Option {
	return fx.Provide(
		provideRiskThresholds,
		provideRiskPolicy,
		providePositionSizer,
		provideSymbolTradingRules,
		provideEvaluateMarketSignal,
		provideEvaluateRisk,
		provideExecuteApprovedOrder,
		provideReconcileBrokerState,
		provideProcessMarketEvent,
		provideActivateKillSwitch,
		provideChangeOperationalMode,
		provideResetPaperState,
		provideQueryService,
	)
}

// InterfaceModule provides HTTP handlers/router/server, the WebSocket hub,
// and every consumer. Lifecycle wiring (starting the server, subscribing
// consumers) lives in lifecycle.go.
func InterfaceModule() fx.Option {
	return fx.Provide(
		provideQueryHandler,
		provideSystemHandler,
		provideHub,
		provideWebSocketHandler,
		provideRouter,
		provideHTTPServer,
		provideMarketConsumer,
		provideEventConsumer,
		provideOrderConsumer,
		provideOutboxConsumer,
	)
}

// WorkerModule provides the background workers (outbox drain, and — via
// lifecycle.go — the reconciliation scheduler, market feed and news
// poller).
func WorkerModule() fx.Option {
	return fx.Provide(
		provideOutboxPublisher,
		provideOutboxWorker,
	)
}
