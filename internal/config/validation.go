package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/shopspring/decimal"
)

var (
	zero = decimal.Zero
	one  = decimal.NewFromInt(1)
)

func validatePercentage(field string, v decimal.Decimal) error {
	if v.LessThan(zero) || v.GreaterThan(one) {
		return fmt.Errorf("%s must be between 0 and 1, got %s", field, v.String())
	}
	return nil
}

func validateURL(field, raw string) error {
	if raw == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid absolute URL, got %q", field, raw)
	}
	return nil
}

// Validate runs every cross-field invariant required before the process is
// allowed to serve traffic. It never mutates the Config; failure here must
// stop startup.
func Validate(c *Config) error {
	var errs []error

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, errors.New("server.port must be between 1 and 65535"))
	}
	if c.Server.MaxRequestBodyBytes <= 0 {
		errs = append(errs, errors.New("server.max_request_body_bytes must be positive"))
	}
	if c.Server.RateLimitRPS <= 0 {
		errs = append(errs, errors.New("server.rate_limit_rps must be positive"))
	}

	if c.Database.Host == "" {
		errs = append(errs, errors.New("database.host must not be empty"))
	}
	if c.Database.Name == "" {
		errs = append(errs, errors.New("database.name must not be empty"))
	}
	if c.Database.MaxConns < c.Database.MinConns || c.Database.MinConns < 1 {
		errs = append(errs, errors.New("database.max_conns must be >= database.min_conns >= 1"))
	}

	switch c.Broker.Mode {
	case "PAPER":
		// no external credentials required
	case "TESTNET", "REAL":
		if c.Broker.BinanceAPIKey == "" || c.Broker.BinanceAPISecret == "" {
			errs = append(errs, fmt.Errorf("broker.mode=%s requires BROKER_BINANCE_API_KEY and BROKER_BINANCE_API_SECRET", c.Broker.Mode))
		}
		if err := validateURL("broker.binance_testnet_url", c.Broker.BinanceTestnetURL); c.Broker.Mode == "TESTNET" && err != nil {
			errs = append(errs, err)
		}
		if err := validateURL("broker.binance_real_url", c.Broker.BinanceRealURL); c.Broker.Mode == "REAL" && err != nil {
			errs = append(errs, err)
		}
	default:
		errs = append(errs, fmt.Errorf("broker.mode must be one of PAPER, TESTNET, REAL, got %q", c.Broker.Mode))
	}
	if c.Broker.Mode == "REAL" {
		// REAL can never be reachable through configuration alone without
		// both an explicit boolean flag and a non-empty confirmation
		// phrase; the domain layer additionally checks the phrase's exact
		// value at the moment of transition.
		if !c.Broker.RealTradingEnabled || c.Broker.RealTradingConfirmation == "" {
			errs = append(errs, errors.New("broker.mode=REAL requires BROKER_REAL_TRADING_ENABLED=true and a non-empty BROKER_REAL_TRADING_CONFIRMATION"))
		}
	}
	if c.Broker.PaperStartingBalance.LessThanOrEqual(zero) {
		errs = append(errs, errors.New("broker.paper_starting_balance must be positive"))
	}

	if err := validatePercentage("risk.per_trade_pct", c.Risk.PerTradePct); err != nil {
		errs = append(errs, err)
	}
	if err := validatePercentage("risk.max_daily_loss_pct", c.Risk.MaxDailyLossPct); err != nil {
		errs = append(errs, err)
	}
	if err := validatePercentage("risk.max_weekly_loss_pct", c.Risk.MaxWeeklyLossPct); err != nil {
		errs = append(errs, err)
	}
	if err := validatePercentage("risk.max_drawdown_pct", c.Risk.MaxDrawdownPct); err != nil {
		errs = append(errs, err)
	}
	if c.Risk.MaxOpenPositions < 1 {
		errs = append(errs, errors.New("risk.max_open_positions must be >= 1"))
	}
	if c.Risk.MaxLeverage.LessThan(one) {
		errs = append(errs, errors.New("risk.max_leverage must be >= 1"))
	}
	if c.Risk.MaxSignalAgeSeconds <= 0 {
		errs = append(errs, errors.New("risk.max_signal_age_seconds must be positive"))
	}
	if c.Risk.MaxMarketDataAgeSeconds <= 0 {
		errs = append(errs, errors.New("risk.max_market_data_age_seconds must be positive"))
	}
	if c.Risk.MaxSlippagePct.LessThan(zero) {
		errs = append(errs, errors.New("risk.max_slippage_pct must not be negative"))
	}

	switch c.LLM.Provider {
	case "noop":
	case "gemini":
		if c.LLM.GeminiAPIKey == "" {
			errs = append(errs, errors.New("llm.provider=gemini requires LLM_GEMINI_API_KEY"))
		}
		if err := validateURL("llm.gemini_base_url", c.LLM.GeminiBaseURL); err != nil {
			errs = append(errs, err)
		}
	default:
		errs = append(errs, fmt.Errorf("llm.provider must be one of noop, gemini, got %q", c.LLM.Provider))
	}
	if c.LLM.MaxPayloadBytes <= 0 {
		errs = append(errs, errors.New("llm.max_payload_bytes must be positive"))
	}

	switch c.Quant.Mode {
	case "fake":
	case "grpc":
		if c.Quant.GRPCTarget == "" {
			errs = append(errs, errors.New("quant.mode=grpc requires QUANT_GRPC_TARGET"))
		}
	default:
		errs = append(errs, fmt.Errorf("quant.mode must be one of fake, grpc, got %q", c.Quant.Mode))
	}

	switch c.Observability.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("observability.log_level must be one of debug, info, warn, error, got %q", c.Observability.LogLevel))
	}
	switch c.Observability.LogFormat {
	case "json", "text":
	default:
		errs = append(errs, fmt.Errorf("observability.log_format must be one of json, text, got %q", c.Observability.LogFormat))
	}

	if len(c.Security.JWTSigningSecret) < 32 {
		errs = append(errs, errors.New("security.jwt_signing_secret must be at least 32 characters"))
	}

	return errors.Join(errs...)
}
