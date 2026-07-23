package config

import (
	"time"

	"github.com/shopspring/decimal"
)

// BrokerConfig configures which broker adapter is active and the
// credentials/parameters each concrete adapter needs. Mode is intentionally
// a plain string here (not the domain operation.Mode enum) so this package
// never imports the domain layer.
type BrokerConfig struct {
	Mode string // PAPER | TESTNET | REAL

	// Paper broker simulation parameters.
	PaperStartingBalance decimal.Decimal
	PaperFeePct          decimal.Decimal
	PaperSlippagePct     decimal.Decimal

	// Binance Futures credentials, shared by the testnet and real adapters;
	// each adapter picks the matching base URL.
	BinanceAPIKey         string
	BinanceAPISecret      string
	BinanceTestnetURL     string
	BinanceRealURL        string
	BinanceRecvWindowMs   int
	BinanceRequestTimeout time.Duration

	// RealTradingEnabled and RealTradingConfirmation are the two explicit,
	// persisted, non-HTTP-reachable controls required before the system is
	// allowed to transition into REAL mode (spec section 14).
	RealTradingEnabled      bool
	RealTradingConfirmation string
}

func loadBrokerConfig() (BrokerConfig, error) {
	startingBalance, err := getEnvDecimal("BROKER_PAPER_STARTING_BALANCE", decimal.NewFromInt(10000))
	if err != nil {
		return BrokerConfig{}, err
	}
	feePct, err := getEnvDecimal("BROKER_PAPER_FEE_PCT", decimal.NewFromFloat(0.0004))
	if err != nil {
		return BrokerConfig{}, err
	}
	slippagePct, err := getEnvDecimal("BROKER_PAPER_SLIPPAGE_PCT", decimal.NewFromFloat(0.0005))
	if err != nil {
		return BrokerConfig{}, err
	}
	recvWindow, err := getEnvInt("BROKER_BINANCE_RECV_WINDOW_MS", 5000)
	if err != nil {
		return BrokerConfig{}, err
	}
	requestTimeout, err := getEnvDuration("BROKER_BINANCE_REQUEST_TIMEOUT", 10*time.Second)
	if err != nil {
		return BrokerConfig{}, err
	}
	realEnabled, err := getEnvBool("BROKER_REAL_TRADING_ENABLED", false)
	if err != nil {
		return BrokerConfig{}, err
	}

	return BrokerConfig{
		Mode:                    getEnv("BROKER_MODE", "PAPER"),
		PaperStartingBalance:    startingBalance,
		PaperFeePct:             feePct,
		PaperSlippagePct:        slippagePct,
		BinanceAPIKey:           getEnv("BROKER_BINANCE_API_KEY", ""),
		BinanceAPISecret:        getEnv("BROKER_BINANCE_API_SECRET", ""),
		BinanceTestnetURL:       getEnv("BROKER_BINANCE_TESTNET_URL", "https://testnet.binancefuture.com"),
		BinanceRealURL:          getEnv("BROKER_BINANCE_REAL_URL", "https://fapi.binance.com"),
		BinanceRecvWindowMs:     recvWindow,
		BinanceRequestTimeout:   requestTimeout,
		RealTradingEnabled:      realEnabled,
		RealTradingConfirmation: getEnv("BROKER_REAL_TRADING_CONFIRMATION", ""),
	}, nil
}

// Redacted returns a copy safe for structured logging.
func (c BrokerConfig) Redacted() BrokerConfig {
	c.BinanceAPIKey = mask(c.BinanceAPIKey)
	c.BinanceAPISecret = mask(c.BinanceAPISecret)
	c.RealTradingConfirmation = mask(c.RealTradingConfirmation)
	return c
}
