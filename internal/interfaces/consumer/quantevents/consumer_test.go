package quantevents

import (
	"encoding/json"
	"log/slog"
	"testing"
)

type fakeBroadcaster struct {
	msgType string
	payload any
	calls   int
}

func (f *fakeBroadcaster) Broadcast(msgType string, payload any) {
	f.msgType = msgType
	f.payload = payload
	f.calls++
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func envelope(t *testing.T, eventType string, payload any) []byte {
	t.Helper()
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshaling payload: %v", err)
	}
	data, err := json.Marshal(map[string]any{
		"event_type": eventType,
		"payload":    json.RawMessage(rawPayload),
	})
	if err != nil {
		t.Fatalf("marshaling envelope: %v", err)
	}
	return data
}

func TestHandleForwardsBacktestCompletedToTheHub(t *testing.T) {
	hub := &fakeBroadcaster{}
	consumer := NewConsumer(hub, testLogger())

	data := envelope(t, "BacktestCompleted", backtestCompletedPayload{BacktestID: "bt-1", TotalTrades: 12, NetProfit: "150.25"})
	consumer.Handle(data)

	if hub.calls != 1 {
		t.Fatalf("expected exactly one broadcast, got %d", hub.calls)
	}
	if hub.msgType != "BacktestCompleted" {
		t.Errorf("msgType = %q, want %q", hub.msgType, "BacktestCompleted")
	}
	got, ok := hub.payload.(backtestCompletedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want backtestCompletedPayload", hub.payload)
	}
	if got.BacktestID != "bt-1" || got.TotalTrades != 12 || got.NetProfit != "150.25" {
		t.Errorf("payload = %+v, want {bt-1 12 150.25}", got)
	}
}

func TestHandleForwardsBacktestFailedToTheHub(t *testing.T) {
	hub := &fakeBroadcaster{}
	consumer := NewConsumer(hub, testLogger())

	data := envelope(t, "BacktestFailed", backtestFailedPayload{BacktestID: "bt-2", Reason: "candle series too short"})
	consumer.Handle(data)

	if hub.calls != 1 || hub.msgType != "BacktestFailed" {
		t.Fatalf("expected one BacktestFailed broadcast, got calls=%d msgType=%q", hub.calls, hub.msgType)
	}
}

func TestHandleForwardsOptimizationCompletedToTheHub(t *testing.T) {
	hub := &fakeBroadcaster{}
	consumer := NewConsumer(hub, testLogger())

	data := envelope(t, "OptimizationCompleted", optimizationCompletedPayload{OptimizationID: "opt-1", BestScore: "1.85", IterationsEvaluated: 40})
	consumer.Handle(data)

	if hub.calls != 1 || hub.msgType != "OptimizationCompleted" {
		t.Fatalf("expected one OptimizationCompleted broadcast, got calls=%d msgType=%q", hub.calls, hub.msgType)
	}
}

// TestHandleIgnoresEventTypesOutOfScope covers BacktestRequested/Started
// and OptimizationRequested — quant-engine publishes them on the same
// subjects this consumer subscribes to (wildcard filter), but nothing
// downstream needs them (see package doc comment).
func TestHandleIgnoresEventTypesOutOfScope(t *testing.T) {
	hub := &fakeBroadcaster{}
	consumer := NewConsumer(hub, testLogger())

	for _, eventType := range []string{"BacktestRequested", "BacktestStarted", "OptimizationRequested", "SomethingUnrelated"} {
		consumer.Handle(envelope(t, eventType, map[string]string{}))
	}

	if hub.calls != 0 {
		t.Fatalf("expected no broadcasts for out-of-scope event types, got %d", hub.calls)
	}
}

func TestHandleNeverPanicsOnMalformedInput(t *testing.T) {
	hub := &fakeBroadcaster{}
	consumer := NewConsumer(hub, testLogger())

	inputs := [][]byte{
		nil,
		[]byte(""),
		[]byte("not json"),
		[]byte(`{"event_type": "BacktestCompleted", "payload": "not an object"}`),
		[]byte(`{"event_type": 123}`),
	}
	for _, data := range inputs {
		consumer.Handle(data)
	}

	if hub.calls != 0 {
		t.Fatalf("expected no broadcasts from malformed input, got %d", hub.calls)
	}
}
