// Package quantevents decodes quant-engine's job-completion events —
// published on quant.backtest.*.events / quant.optimization.*.events,
// consumed by the JetStream wiring in internal/app/lifecycle.go — and
// forwards the ones the dashboard cares about to the WebSocket hub. It
// carries no business logic: decode envelope, recognize event_type,
// forward payload. Real-time notification of a job trading-core kicked
// off asynchronously (POST /v1/backtest, POST /v1/optimization/run),
// in place of the dashboard polling GET /v1/backtest/{id}.
package quantevents

import (
	"encoding/json"
	"log/slog"
)

// Broadcaster is the minimal surface this consumer needs from the
// WebSocket hub.
type Broadcaster interface {
	Broadcast(msgType string, payload any)
}

// envelopeHeader decodes just enough of quant-engine's EventEnvelope
// (see infrastructure/external/messaging/nats.Envelope, which mirrors
// it field for field) to route on event_type before committing to a
// concrete payload shape.
type envelopeHeader struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

// These three mirror quant-engine's contracts::events::v1 payloads
// field for field (src/contracts/events/v1/backtest.rs,
// src/contracts/events/v1/optimization.rs) for exactly the three
// event_types this consumer forwards. BacktestRequested/Started and
// OptimizationRequested (and any other event quant-engine publishes)
// are deliberately not decoded here — out of scope, nothing downstream
// needs them.
type backtestCompletedPayload struct {
	BacktestID  string `json:"backtest_id"`
	TotalTrades uint32 `json:"total_trades"`
	NetProfit   string `json:"net_profit"`
}

type backtestFailedPayload struct {
	BacktestID string `json:"backtest_id"`
	Reason     string `json:"reason"`
}

type optimizationCompletedPayload struct {
	OptimizationID      string `json:"optimization_id"`
	BestScore           string `json:"best_score"`
	IterationsEvaluated uint32 `json:"iterations_evaluated"`
}

type Consumer struct {
	hub    Broadcaster
	logger *slog.Logger
}

func NewConsumer(hub Broadcaster, logger *slog.Logger) *Consumer {
	return &Consumer{hub: hub, logger: logger}
}

// Handle decodes one message and, for a recognized event_type, forwards
// it to the hub under the same name the dashboard already expects from
// the synchronous WebSocket events. An undecodable envelope or payload
// is logged and dropped rather than retried — same reasoning as
// quant-engine's own consumer: redelivering a message this side can
// never parse would just loop forever.
func (c *Consumer) Handle(data []byte) {
	var header envelopeHeader
	if err := json.Unmarshal(data, &header); err != nil {
		c.logger.Error("quantevents consumer: invalid envelope", "error", err)
		return
	}

	switch header.EventType {
	case "BacktestCompleted":
		var payload backtestCompletedPayload
		if err := json.Unmarshal(header.Payload, &payload); err != nil {
			c.logger.Error("quantevents consumer: invalid BacktestCompleted payload", "error", err)
			return
		}
		c.hub.Broadcast("BacktestCompleted", payload)
	case "BacktestFailed":
		var payload backtestFailedPayload
		if err := json.Unmarshal(header.Payload, &payload); err != nil {
			c.logger.Error("quantevents consumer: invalid BacktestFailed payload", "error", err)
			return
		}
		c.hub.Broadcast("BacktestFailed", payload)
	case "OptimizationCompleted":
		var payload optimizationCompletedPayload
		if err := json.Unmarshal(header.Payload, &payload); err != nil {
			c.logger.Error("quantevents consumer: invalid OptimizationCompleted payload", "error", err)
			return
		}
		c.hub.Broadcast("OptimizationCompleted", payload)
	}
}
