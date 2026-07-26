package app

import "go.uber.org/fx"

// Module is the full application graph: every module plus the lifecycle
// hooks that turn constructed dependencies into a running system (HTTP
// server, background workers, event subscriptions, initial DB bootstrap).
// cmd/api uses this directly; cmd/worker uses WorkerOnlyModule instead.
func Module() fx.Option {
	return fx.Options(
		ConfigModule(),
		ObservabilityModule(),
		DatabaseModule(),
		RepositoryModule(),
		ExternalModule(),
		ApplicationModule(),
		InterfaceModule(),
		WorkerModule(),

		fx.Invoke(
			bootstrap,
			startHTTPServer,
			subscribeConsumers,
			startOutboxWorker,
			startReconciliationScheduler,
			startMarketDataFeed,
			startQuantEventsConsumer,
			startActivityWatcher,
		),
	)
}

// WorkerOnlyModule loads every module needed to run background processing
// without an HTTP server — cmd/worker's entry point. It deliberately does
// NOT start the market data feed: that pipeline creates a new TradeSignal/
// RiskDecision per candle with freshly generated ids, so two independent
// feeds (one per process) would each approve and execute their own order
// for the same market move — unlike the outbox worker (SELECT ... FOR
// UPDATE SKIP LOCKED) and reconciliation (read-mostly, idempotent
// incident creation), that path is not safe to duplicate across
// processes. Exactly one process (cmd/api) owns the feed.
func WorkerOnlyModule() fx.Option {
	return fx.Options(
		ConfigModule(),
		ObservabilityModule(),
		DatabaseModule(),
		RepositoryModule(),
		ExternalModule(),
		ApplicationModule(),
		WorkerModule(),

		fx.Invoke(
			bootstrap,
			startOutboxWorker,
			startReconciliationScheduler,
			startActivityWatcher,
		),
	)
}
