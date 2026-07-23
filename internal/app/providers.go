// Package app is the only place (besides cmd/*) allowed to import Uber Fx.
// providers.go holds every constructor function the fx modules wire
// together: mode-based adapter factories (Strategy/Factory pattern per
// spec section 8.3) and use case constructors.
package app

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/input"
	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/usecase"
	"trading-core/internal/config"
	"trading-core/internal/domain/risk"
	"trading-core/internal/domain/shared"

	realbroker "trading-core/internal/infrastructure/external/broker/real"
	testnetbroker "trading-core/internal/infrastructure/external/broker/testnet"

	paperbroker "trading-core/internal/infrastructure/external/broker/paper"
	geminillm "trading-core/internal/infrastructure/external/llm/gemini"
	noopllm "trading-core/internal/infrastructure/external/llm/noop"
	"trading-core/internal/infrastructure/external/messaging/memory"
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

func provideQuantEngine(cfg *config.Config) (output.QuantEngine, error) {
	switch cfg.Quant.Mode {
	case "fake":
		return quantfake.NewEngine(), nil
	default:
		return nil, fmt.Errorf("quant.mode %q is not implemented in this delivery; use \"fake\" (see internal/contracts/grpc/quant)", cfg.Quant.Mode)
	}
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

func provideEventBus() output.EventBus {
	return memory.NewEventBus()
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
