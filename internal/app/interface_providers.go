package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"trading-core/internal/application/ports/input"
	"trading-core/internal/config"
	"trading-core/internal/infrastructure/database"
	"trading-core/internal/infrastructure/observability/metrics"

	httpiface "trading-core/internal/interfaces/http"
	httphandler "trading-core/internal/interfaces/http/handler"
	httprouter "trading-core/internal/interfaces/http/router"

	eventconsumer "trading-core/internal/interfaces/consumer/event"
	marketconsumer "trading-core/internal/interfaces/consumer/market"
	orderconsumer "trading-core/internal/interfaces/consumer/order"
	outboxconsumer "trading-core/internal/interfaces/consumer/outbox"
	quantevents "trading-core/internal/interfaces/consumer/quantevents"

	wshandler "trading-core/internal/interfaces/websocket/handler"
	wshub "trading-core/internal/interfaces/websocket/hub"
)

func provideQueryHandler(queries input.QueryService) *httphandler.QueryHandler {
	return httphandler.NewQueryHandler(queries)
}

func provideSystemHandler(
	changeMode input.ChangeOperationalModeUseCase,
	killSwitch input.ActivateKillSwitchUseCase,
	resetPaper input.ResetPaperStateUseCase,
	cfg *config.Config,
) *httphandler.SystemHandler {
	return httphandler.NewSystemHandler(changeMode, killSwitch, resetPaper, cfg.Broker.Mode)
}

func provideHub(logger *slog.Logger) *wshub.Hub {
	return wshub.NewHub(logger)
}

func provideWebSocketHandler(h *wshub.Hub, logger *slog.Logger) *wshandler.Handler {
	return wshandler.NewHandler(h, logger)
}

func provideBacktestHandler(backtests input.BacktestService) *httphandler.BacktestHandler {
	return httphandler.NewBacktestHandler(backtests)
}

func provideOptimizationHandler(optimizations input.OptimizationService) *httphandler.OptimizationHandler {
	return httphandler.NewOptimizationHandler(optimizations)
}

func provideDatasetHandler(exports input.DatasetExportService) *httphandler.DatasetHandler {
	return httphandler.NewDatasetHandler(exports)
}

func provideStrategyHandler(diagnostics input.QuantDiagnosticsService) *httphandler.StrategyHandler {
	return httphandler.NewStrategyHandler(diagnostics)
}

func provideQuantDiagnosticsHandler(diagnostics input.QuantDiagnosticsService, publisher input.QuantEventPublisherService) *httphandler.QuantDiagnosticsHandler {
	return httphandler.NewQuantDiagnosticsHandler(diagnostics, publisher)
}

func provideRouter(
	cfg *config.Config,
	logger *slog.Logger,
	m *metrics.Metrics,
	pool *pgxpool.Pool,
	queryHandler *httphandler.QueryHandler,
	systemHandler *httphandler.SystemHandler,
	backtestHandler *httphandler.BacktestHandler,
	optimizationHandler *httphandler.OptimizationHandler,
	datasetHandler *httphandler.DatasetHandler,
	strategyHandler *httphandler.StrategyHandler,
	quantDiagnosticsHandler *httphandler.QuantDiagnosticsHandler,
	ws *wshandler.Handler,
) http.Handler {
	return httprouter.New(httprouter.Config{
		Logger:              logger,
		JWTSigningSecret:    cfg.Security.JWTSigningSecret,
		CORSAllowedOrigins:  cfg.Server.CORSAllowedOrigins,
		MaxRequestBodyBytes: cfg.Server.MaxRequestBodyBytes,
		RequestTimeout:      cfg.Server.RequestTimeout,
		RateLimitRPS:        cfg.Server.RateLimitRPS,
		RateLimitBurst:      cfg.Server.RateLimitBurst,

		HealthHandler: httphandler.Health,
		ReadyHandler: httphandler.Ready(func(ctx context.Context) error {
			return database.CheckHealth(ctx, pool)
		}),
		MetricsHandler:   m.Handler,
		WebSocketHandler: ws,

		QueryHandler:  queryHandler,
		SystemHandler: systemHandler,

		BacktestHandler:         backtestHandler,
		OptimizationHandler:     optimizationHandler,
		DatasetHandler:          datasetHandler,
		StrategyHandler:         strategyHandler,
		QuantDiagnosticsHandler: quantDiagnosticsHandler,
	})
}

func provideHTTPServer(cfg *config.Config, router http.Handler) *httpiface.Server {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return httpiface.NewServer(addr, router, cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout)
}

func provideMarketConsumer(
	evaluateSignal input.EvaluateMarketSignalUseCase,
	evaluateRisk input.EvaluateRiskUseCase,
	executeOrder input.ExecuteApprovedOrderUseCase,
	logger *slog.Logger,
) *marketconsumer.Consumer {
	return marketconsumer.NewConsumer(evaluateSignal, evaluateRisk, executeOrder, logger)
}

func provideEventConsumer(processEvent input.ProcessMarketEventUseCase, logger *slog.Logger) *eventconsumer.Consumer {
	return eventconsumer.NewConsumer(processEvent, logger)
}

func provideOrderConsumer(m *metrics.Metrics) *orderconsumer.Consumer {
	return orderconsumer.NewConsumer(orderconsumer.Counters{
		Created:   m.OrdersCreated,
		Submitted: m.OrdersSubmitted,
	})
}

func provideOutboxConsumer(h *wshub.Hub) *outboxconsumer.Consumer {
	return outboxconsumer.NewConsumer(h)
}

func provideQuantEventsConsumer(h *wshub.Hub, logger *slog.Logger) *quantevents.Consumer {
	return quantevents.NewConsumer(h, logger)
}
