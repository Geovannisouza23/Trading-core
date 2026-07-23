// Package request holds the JSON request bodies handlers decode.
package request

// ReasonRequest is the body accepted by every /v1/system/* control
// endpoint: a free-text operator-supplied reason, persisted into the audit
// trail (operation.State / SystemIncident).
type ReasonRequest struct {
	Reason string `json:"reason"`
}

// KillSwitchRequest additionally lets the caller decide whether closing
// existing positions stays allowed while the kill switch is active.
type KillSwitchRequest struct {
	Reason     string `json:"reason"`
	AllowClose bool   `json:"allow_close"`
}

// BacktestRequest is accepted (and validated) by POST /v1/backtest/request,
// even though no backtest engine is implemented in this delivery — see
// internal/interfaces/http/handler/backtest.go.
type BacktestRequest struct {
	Symbol       string `json:"symbol"`
	Timeframe    string `json:"timeframe"`
	StrategyName string `json:"strategy_name"`
	From         string `json:"from"`
	To           string `json:"to"`
}
