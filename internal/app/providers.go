// Package app is the only place (besides cmd/*) allowed to import Uber Fx.
// providers.go holds every constructor function the fx modules wire
// together: mode-based adapter factories (Strategy/Factory pattern per
// spec section 8.3) and use case constructors.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"

	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/usecase"
	"trading-core/internal/config"
	"trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/observability/metrics"

	realbroker "trading-core/internal/infrastructure/external/broker/real"
	testnetbroker "trading-core/internal/infrastructure/external/broker/testnet"

	paperbroker "trading-core/internal/infrastructure/external/broker/paper"
	redispkg "trading-core/internal/infrastructure/external/cache/redis"
	geminillm "trading-core/internal/infrastructure/external/llm/gemini"
	noopllm "trading-core/internal/infrastructure/external/llm/noop"
	"trading-core/internal/infrastructure/external/messaging/memory"
	natspkg "trading-core/internal/infrastructure/external/messaging/nats"
	quantfake "trading-core/internal/infrastructure/external/quant/grpc"

	"trading-core/internal/infrastructure/external/notification/telegram"
)

// --- Mode-selected adapter factories (Factory pattern) ---------------------

func provideBroker(cfg *config.Config) (output.Broker, error) {
	switch cfg.Broker.Mode {
	case "PAPER":
		return paperbroker.NewBroker(cfg.Broker.PaperStartingBalance, cfg.Broker.PaperFeePct, cfg.Broker.PaperSlippagePct), nil
	case "TESTNET":
		return testnetbroker.NewBroker(cfg.Broker.BinanceTestnetURL, cfg.Broker.BinanceAPIKey, cfg.Broker.BinanceAPISecret, cfg.Broker.BinanceRecvWindowMs, cfg.Broker.BinanceRequestTimeout), nil
	case "REAL":
		return realbroker.NewBroker(cfg.Broker.BinanceRealURL, cfg.Broker.BinanceAPIKey, cfg.Broker.BinanceAPISecret, cfg.Broker.BinanceRecvWindowMs, cfg.Broker.BinanceRequestTimeout, cfg.Broker.RealTradingEnabled, cfg.Broker.RealTradingConfirmation)
	default:
		return nil, fmt.Errorf("unknown broker.mode %q", cfg.Broker.Mode)
	}
}

// provideQuantGrpcClient builds the single shared gRPC connection to
// quant-engine when quant.mode == "grpc", nil otherwise. Every one of the
// five quant output ports below (QuantEngine, BacktestService,
// OptimizationService, DatasetExportService, QuantDiagnostics) is backed
// by this same *grpc.Client instance in grpc mode — never five separate
// connections.
func provideQuantGrpcClient(lc fx.Lifecycle, cfg *config.Config, m *metrics.Metrics) (*quantfake.Client, error) {
	if cfg.Quant.Mode != "grpc" {
		return nil, nil
	}
	client, err := quantfake.NewClient(cfg.Quant.GRPCTarget, cfg.Quant.Timeout, m)
	if err != nil {
		return nil, fmt.Errorf("building quant engine grpc client: %w", err)
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return client.Close() }})
	return client, nil
}

func provideQuantEngine(cfg *config.Config, client *quantfake.Client) (output.QuantEngine, error) {
	switch cfg.Quant.Mode {
	case "fake":
		return quantfake.NewEngine(), nil
	case "grpc":
		return client, nil
	default:
		return nil, fmt.Errorf("unknown quant.mode %q (expected \"fake\" or \"grpc\")", cfg.Quant.Mode)
	}
}

// The four capabilities below have no meaningful fake-mode behavior (see
// quantfake.Unavailable) — they are only real when quant.mode == "grpc".

func provideQuantBacktestService(cfg *config.Config, client *quantfake.Client) output.BacktestService {
	if cfg.Quant.Mode == "grpc" {
		return client
	}
	return quantfake.NewUnavailable(cfg.Quant.Mode)
}

func provideQuantOptimizationService(cfg *config.Config, client *quantfake.Client) output.OptimizationService {
	if cfg.Quant.Mode == "grpc" {
		return client
	}
	return quantfake.NewUnavailable(cfg.Quant.Mode)
}

func provideQuantDatasetExportService(cfg *config.Config, client *quantfake.Client) output.DatasetExportService {
	if cfg.Quant.Mode == "grpc" {
		return client
	}
	return quantfake.NewUnavailable(cfg.Quant.Mode)
}

func provideQuantDiagnosticsService(cfg *config.Config, client *quantfake.Client) output.QuantDiagnostics {
	if cfg.Quant.Mode == "grpc" {
		return client
	}
	return quantfake.NewUnavailable(cfg.Quant.Mode)
}

func provideEventIntelligence(cfg *config.Config) (output.EventIntelligence, error) {
	switch cfg.LLM.Provider {
	case "noop":
		return noopllm.NewClient(), nil
	case "gemini":
		return geminillm.NewClient(cfg.LLM.GeminiBaseURL, cfg.LLM.GeminiAPIKey, cfg.LLM.GeminiModel, cfg.LLM.RequestTimeout, cfg.LLM.MaxPayloadBytes), nil
	default:
		return nil, fmt.Errorf("unknown llm.provider %q", cfg.LLM.Provider)
	}
}

func provideNotifier(cfg *config.Config) output.Notifier {
	return telegram.NewNotifier(cfg.Security.TelegramBotToken, cfg.Security.TelegramChatID)
}

// provideNatsConnection dials NATS once, shared by the EventBus and the
// QuantEventPublisher below — never two separate connections for one
// logical dependency, same reasoning as provideQuantGrpcClient. Returns
// (nil, nil, nil) on a reachability failure when cfg.Nats.Required is
// false: callers fall back to their pre-existing behavior (in-memory
// EventBus, a QuantEventPublisher that errors clearly on publish)
// instead of failing to boot — mirrors quant-engine's own
// app::providers::build_event_bus exactly.
func provideNatsConnection(lc fx.Lifecycle, cfg *config.Config, logger *slog.Logger) (*natsgo.Conn, jetstream.JetStream, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, js, err := natspkg.Connect(ctx, cfg.Nats.URL, cfg.Nats.StreamPrefix)
	if err != nil {
		if cfg.Nats.Required {
			return nil, nil, fmt.Errorf("connecting to required nats at %s: %w", cfg.Nats.URL, err)
		}
		logger.Warn("nats unavailable at startup; falling back to the in-memory event bus and a non-publishing QuantEventPublisher", "url", cfg.Nats.URL, "error", err)
		return nil, nil, nil
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return conn.Drain() }})
	return conn, js, nil
}

func provideEventBus(conn *natsgo.Conn, js jetstream.JetStream, cfg *config.Config, logger *slog.Logger) output.EventBus {
	if js == nil {
		return memory.NewEventBus()
	}
	return natspkg.NewEventBus(conn, js, cfg.Nats.StreamPrefix, logger)
}

func provideQuantEventPublisher(js jetstream.JetStream) output.QuantEventPublisher {
	return natspkg.NewPublisher(js)
}

func provideActivityPublisher(js jetstream.JetStream) output.ActivityPublisher {
	return natspkg.NewPublisher(js)
}

// provideRedisClient builds the shared go-redis client, used by both the
// market-data cache and the cross-service idempotency store below.
// Optional: constructed eagerly (go-redis dials lazily on first command,
// so this never blocks startup), Ping-checked once at boot only to log a
// clear warning — a Redis outage after startup degrades individual
// cache/dedup calls, it never fails the request that triggered them.
func provideRedisClient(lc fx.Lifecycle, cfg *config.Config, logger *slog.Logger) *redispkg.Client {
	client := redispkg.NewClient(cfg.Redis.Addr, "", 0)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Ping(ctx); err != nil {
				if cfg.Redis.Required {
					return fmt.Errorf("connecting to required redis at %s: %w", cfg.Redis.Addr, err)
				}
				logger.Warn("redis unavailable at startup; market-data caching and cross-service idempotency dedup are disabled", "addr", cfg.Redis.Addr, "error", err)
			}
			return nil
		},
		OnStop: func(context.Context) error { return client.Close() },
	})
	return client
}

func provideMarketDataCache(client *redispkg.Client) output.MarketDataCache {
	return redispkg.NewMarketDataCache(client)
}

// provideRedisIdempotencyRepository is named "natsPublishIdempotency" via
// fx.Annotate in ExternalModule() — see provideQuantEventPublisherUseCase
// for why this second binding of output.IdempotencyRepository needs a
// name to coexist with the Postgres-backed one from RepositoryModule().
func provideRedisIdempotencyRepository(client *redispkg.Client) output.IdempotencyRepository {
	return redispkg.NewIdempotencyRepository(client)
}

// --- Risk wiring -------------------------------------------------------

func provideRiskThresholds(cfg *config.Config) (risk.Thresholds, error) {
	perTrade, err := shared.NewPercentage(cfg.Risk.PerTradePct)
	if err != nil {
		return risk.Thresholds{}, err
	}
	maxDailyLoss, err := shared.NewPercentage(cfg.Risk.MaxDailyLossPct)
	if err != nil {
		return risk.Thresholds{}, err
	}
	maxWeeklyLoss, err := shared.NewPercentage(cfg.Risk.MaxWeeklyLossPct)
	if err != nil {
		return risk.Thresholds{}, err
	}
	maxDrawdown, err := shared.NewDrawdown(cfg.Risk.MaxDrawdownPct)
	if err != nil {
		return risk.Thresholds{}, err
	}
	maxLeverage, err := shared.NewLeverage(cfg.Risk.MaxLeverage)
	if err != nil {
		return risk.Thresholds{}, err
	}
	maxSlippage, err := shared.NewSlippage(cfg.Risk.MaxSlippagePct)
	if err != nil {
		return risk.Thresholds{}, err
	}
	return risk.Thresholds{
		PerTradePct:      perTrade,
		MaxDailyLossPct:  maxDailyLoss,
		MaxWeeklyLossPct: maxWeeklyLoss,
		MaxDrawdown:      maxDrawdown,
		MaxOpenPositions: cfg.Risk.MaxOpenPositions,
		MaxLeverage:      maxLeverage,
		MaxSignalAge:     time.Duration(cfg.Risk.MaxSignalAgeSeconds) * time.Second,
		MaxMarketDataAge: time.Duration(cfg.Risk.MaxMarketDataAgeSeconds) * time.Second,
		MaxSlippage:      maxSlippage,
	}, nil
}

func provideRiskPolicy() *risk.CompositeRiskPolicy {
	return risk.NewDefaultRiskPolicy()
}

func providePositionSizer() risk.PositionSizer {
	return risk.FixedRiskPositionSizing{}
}

// provideSymbolTradingRules returns conservative MVP defaults: the Broker
// port (spec section 13.1, given verbatim) has no exchange-info method, so
// per-symbol min quantity/step/notional cannot be looked up dynamically.
func provideSymbolTradingRules() usecase.SymbolTradingRules {
	return usecase.SymbolTradingRules{
		MinQuantity: shared.MustNewQuantity(decimal.NewFromFloat(0.001)),
		StepSize:    decimal.NewFromFloat(0.001),
		MinNotional: shared.NewMoney(decimal.NewFromInt(5)),
	}
}

// --- Use case constructors ----------------------------------------------

func provideEvaluateMarketSignal(quant output.QuantEngine, signals output.TradeSignalRepository, bus output.EventBus, clock output.Clock, cfg *config.Config) input.EvaluateMarketSignalUseCase {
	maxCandleAge := time.Duration(cfg.Risk.MaxMarketDataAgeSeconds) * time.Second
	signalValidity := time.Duration(cfg.Risk.MaxSignalAgeSeconds) * time.Second
	return usecase.NewEvaluateMarketSignal(quant, signals, bus, clock, maxCandleAge, signalValidity)
}

func provideEvaluateRisk(
	accounts output.AccountRepository, positions output.PositionRepository, orders output.OrderRepository,
	events output.MarketEventRepository, signals output.TradeSignalRepository, decisions output.RiskDecisionRepository,
	broker output.Broker, bus output.EventBus, clock output.Clock,
	policy *risk.CompositeRiskPolicy, sizer risk.PositionSizer, thresholds risk.Thresholds, rules usecase.SymbolTradingRules,
) input.EvaluateRiskUseCase {
	return usecase.NewEvaluateRisk(accounts, positions, orders, events, signals, decisions, broker, bus, clock, policy, sizer, thresholds, rules)
}

func provideExecuteApprovedOrder(
	decisions output.RiskDecisionRepository, signals output.TradeSignalRepository, accounts output.AccountRepository,
	orders output.OrderRepository, positions output.PositionRepository, incidents output.SystemIncidentRepository,
	idempotent output.IdempotencyRepository, outbox output.OutboxRepository, broker output.Broker,
	tx output.TransactionManager, clock output.Clock,
) input.ExecuteApprovedOrderUseCase {
	return usecase.NewExecuteApprovedOrder(decisions, signals, accounts, orders, positions, incidents, idempotent, outbox, broker, tx, clock)
}

func provideReconcileBrokerState(
	accounts output.AccountRepository, positions output.PositionRepository, orders output.OrderRepository,
	incidents output.SystemIncidentRepository, modes output.OperationalModeRepository, broker output.Broker,
	outbox output.OutboxRepository, tx output.TransactionManager, clock output.Clock,
) input.ReconcileBrokerStateUseCase {
	tolerance := decimal.NewFromFloat(0.005)
	return usecase.NewReconcileBrokerState(accounts, positions, orders, incidents, modes, broker, outbox, tx, clock, tolerance)
}

func provideProcessMarketEvent(
	events output.MarketEventRepository, intelligence output.EventIntelligence, idempotent output.IdempotencyRepository,
	outbox output.OutboxRepository, tx output.TransactionManager, notifier output.Notifier, clock output.Clock, cfg *config.Config,
) input.ProcessMarketEventUseCase {
	const maxNewsAge = 48 * time.Hour
	return usecase.NewProcessMarketEvent(events, intelligence, idempotent, outbox, tx, notifier, clock, cfg.Security.AllowedNewsSources, cfg.Security.MaxNewsPayloadBytes, maxNewsAge)
}

func provideActivateKillSwitch(
	modes output.OperationalModeRepository, accounts output.AccountRepository, incidents output.SystemIncidentRepository,
	outbox output.OutboxRepository, tx output.TransactionManager, notifier output.Notifier, clock output.Clock,
) input.ActivateKillSwitchUseCase {
	return usecase.NewActivateKillSwitch(modes, accounts, incidents, outbox, tx, notifier, clock)
}

func provideChangeOperationalMode(
	modes output.OperationalModeRepository, accounts output.AccountRepository, tx output.TransactionManager,
	bus output.EventBus, clock output.Clock,
) input.ChangeOperationalModeUseCase {
	return usecase.NewChangeOperationalMode(modes, accounts, tx, bus, clock)
}

func provideResetPaperState(
	accounts output.AccountRepository, positions output.PositionRepository, orders output.OrderRepository,
	broker output.Broker, tx output.TransactionManager, clock output.Clock, cfg *config.Config,
) input.ResetPaperStateUseCase {
	return usecase.NewResetPaperState(accounts, positions, orders, broker, tx, clock, shared.NewMoney(cfg.Broker.PaperStartingBalance))
}

func provideQueryService(
	accounts output.AccountRepository, positions output.PositionRepository, orders output.OrderRepository,
	signals output.TradeSignalRepository, decisions output.RiskDecisionRepository, events output.MarketEventRepository,
	incidents output.SystemIncidentRepository, broker output.Broker,
) input.QueryService {
	return usecase.NewQueryService(accounts, positions, orders, signals, decisions, events, incidents, broker)
}

func provideBacktestUseCase(backtests output.BacktestService) input.BacktestService {
	return usecase.NewBacktestService(backtests)
}

func provideOptimizationUseCase(optimizations output.OptimizationService) input.OptimizationService {
	return usecase.NewOptimizationService(optimizations)
}

func provideDatasetExportUseCase(exports output.DatasetExportService) input.DatasetExportService {
	return usecase.NewDatasetExportService(exports)
}

func provideQuantDiagnosticsUseCase(diagnostics output.QuantDiagnostics) input.QuantDiagnosticsService {
	return usecase.NewQuantDiagnosticsService(diagnostics)
}

// provideQuantEventPublisherUseCase's idempotent param is the
// "natsPublishIdempotency"-named Redis-backed binding registered in
// ExternalModule(), disambiguated via fx.ParamTags from the default
// (Postgres-backed) output.IdempotencyRepository used everywhere else.
func provideQuantEventPublisherUseCase(publisher output.QuantEventPublisher, idempotent output.IdempotencyRepository, logger *slog.Logger) input.QuantEventPublisherService {
	return usecase.NewQuantEventPublisherService(publisher, idempotent, logger)
}
