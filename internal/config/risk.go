package config

import "github.com/shopspring/decimal"

// RiskConfig mirrors the risk thresholds from spec section 12. Defaults
// match the example YAML exactly. Values stay as decimal.Decimal / int here;
// the application layer's mapper converts them into domain risk.Thresholds
// so this package never imports the domain.
type RiskConfig struct {
	PerTradePct             decimal.Decimal
	MaxDailyLossPct         decimal.Decimal
	MaxWeeklyLossPct        decimal.Decimal
	MaxDrawdownPct          decimal.Decimal
	MaxOpenPositions        int
	MaxLeverage             decimal.Decimal
	MaxSignalAgeSeconds     int
	MaxMarketDataAgeSeconds int
	MaxSlippagePct          decimal.Decimal
}

func loadRiskConfig() (RiskConfig, error) {
	perTrade, err := getEnvDecimal("RISK_PER_TRADE_PCT", decimal.NewFromFloat(0.03))
	if err != nil {
		return RiskConfig{}, err
	}
	maxDailyLoss, err := getEnvDecimal("RISK_MAX_DAILY_LOSS_PCT", decimal.NewFromFloat(0.09))
	if err != nil {
		return RiskConfig{}, err
	}
	maxWeeklyLoss, err := getEnvDecimal("RISK_MAX_WEEKLY_LOSS_PCT", decimal.NewFromFloat(0.15))
	if err != nil {
		return RiskConfig{}, err
	}
	maxDrawdown, err := getEnvDecimal("RISK_MAX_DRAWDOWN_PCT", decimal.NewFromFloat(0.25))
	if err != nil {
		return RiskConfig{}, err
	}
	maxOpenPositions, err := getEnvInt("RISK_MAX_OPEN_POSITIONS", 1)
	if err != nil {
		return RiskConfig{}, err
	}
	maxLeverage, err := getEnvDecimal("RISK_MAX_LEVERAGE", decimal.NewFromInt(1))
	if err != nil {
		return RiskConfig{}, err
	}
	maxSignalAge, err := getEnvInt("RISK_MAX_SIGNAL_AGE_SECONDS", 30)
	if err != nil {
		return RiskConfig{}, err
	}
	maxMarketDataAge, err := getEnvInt("RISK_MAX_MARKET_DATA_AGE_SECONDS", 10)
	if err != nil {
		return RiskConfig{}, err
	}
	maxSlippage, err := getEnvDecimal("RISK_MAX_SLIPPAGE_PCT", decimal.NewFromFloat(0.30))
	if err != nil {
		return RiskConfig{}, err
	}

	return RiskConfig{
		PerTradePct:             perTrade,
		MaxDailyLossPct:         maxDailyLoss,
		MaxWeeklyLossPct:        maxWeeklyLoss,
		MaxDrawdownPct:          maxDrawdown,
		MaxOpenPositions:        maxOpenPositions,
		MaxLeverage:             maxLeverage,
		MaxSignalAgeSeconds:     maxSignalAge,
		MaxMarketDataAgeSeconds: maxMarketDataAge,
		MaxSlippagePct:          maxSlippage,
	}, nil
}
