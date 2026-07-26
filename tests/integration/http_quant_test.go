//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/application/usecase"
	httphandler "trading-core/internal/interfaces/http/handler"
)

// newQuantHTTPTestServer wires the real backtest/diagnostics HTTP
// handlers directly on top of the real gRPC client — skipping Fx and the
// Postgres-backed parts of the app (out of scope for this test) — to
// prove the full HTTP path (JSON decode -> command -> use case -> real
// gRPC call -> real server -> JSON encode) works end to end, not just
// that the gRPC client alone does.
func newQuantHTTPTestServer(t *testing.T, target string) *httptest.Server {
	t.Helper()
	client := newTestQuantClient(t, target)

	backtests := usecase.NewBacktestService(client)
	diagnostics := usecase.NewQuantDiagnosticsService(client)

	backtestHandler := httphandler.NewBacktestHandler(backtests)
	// nil publisher: this test only registers the FeatureSchema route
	// below, never the NATS-publish routes that would actually call it.
	diagnosticsHandler := httphandler.NewQuantDiagnosticsHandler(diagnostics, nil)

	r := chi.NewRouter()
	r.Post("/v1/backtest/request", backtestHandler.Request)
	r.Get("/v1/backtest/{id}", backtestHandler.Result)
	r.Get("/v1/quant/feature-schema", diagnosticsHandler.FeatureSchema)

	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	return server
}

func TestHTTPBacktestRequestAndResultAgainstARealServer(t *testing.T) {
	target := newQuantEngineContainer(t)
	server := newQuantHTTPTestServer(t, target)

	now := time.Now()
	// quant-engine's default trend_following strategy needs at least 228
	// candles (discovered empirically against the real service, see
	// testCandleSeries in quant_grpc_test.go).
	const numCandles = 250
	candles := make([]map[string]any, 0, numCandles)
	for i := numCandles; i > 0; i-- {
		closeTime := now.Add(-time.Duration(i) * time.Minute)
		openTime := closeTime.Add(-time.Minute)
		price := 50000 + i*10
		candles = append(candles, map[string]any{
			"open_time":  openTime.Format(time.RFC3339),
			"close_time": closeTime.Format(time.RFC3339),
			"open":       strconv.Itoa(price - 1),
			"high":       strconv.Itoa(price + 1),
			"low":        strconv.Itoa(price - 2),
			"close":      strconv.Itoa(price),
			"volume":     "10",
		})
	}

	body := map[string]any{
		"config": map[string]any{
			"symbol":           "BTCUSDT",
			"timeframe":        "1m",
			"strategy_name":    "trend_following",
			"strategy_version": "1.0.0",
			"initial_capital":  "10000",
			"fee_rate":         "0.0004",
			"slippage":         "0.0001",
			"execution_model":  "CLOSE_PRICE",
			"from":             now.Add(-numCandles * time.Minute).Format(time.RFC3339),
			"to":               now.Format(time.RFC3339),
		},
		"candles": candles,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling request body: %v", err)
	}

	resp, err := http.Post(server.URL+"/v1/backtest/request", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /v1/backtest/request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("got status %d, want 202: %s", resp.StatusCode, respBody)
	}

	var accepted output.RunBacktestResult
	if err := json.NewDecoder(resp.Body).Decode(&accepted); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if accepted.BacktestID == "" {
		t.Fatal("expected a non-empty backtest_id in the HTTP response")
	}

	resultResp, err := http.Get(server.URL + "/v1/backtest/" + accepted.BacktestID)
	if err != nil {
		t.Fatalf("GET /v1/backtest/{id}: %v", err)
	}
	defer resultResp.Body.Close()
	if resultResp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resultResp.StatusCode)
	}

	var result output.GetBacktestResultResult
	if err := json.NewDecoder(resultResp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding result response: %v", err)
	}
	if result.BacktestID != accepted.BacktestID {
		t.Fatalf("got backtest_id %q, want %q", result.BacktestID, accepted.BacktestID)
	}
	t.Logf("HTTP-driven backtest %s: status=%s", result.BacktestID, result.Status)
}

func TestHTTPFeatureSchemaAgainstARealServer(t *testing.T) {
	target := newQuantEngineContainer(t)
	server := newQuantHTTPTestServer(t, target)

	resp, err := http.Get(server.URL + "/v1/quant/feature-schema")
	if err != nil {
		t.Fatalf("GET /v1/quant/feature-schema: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var result output.GetFeatureSchemaResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(result.Features) != 34 {
		t.Fatalf("got %d features, want 34 (the v1 schema)", len(result.Features))
	}
}
