//go:build integration

// Package integration_test exercises the real quant-engine gRPC service —
// built from ../../quant-engine's own Dockerfile, then run via
// Testcontainers by image reference (not testcontainers-go's
// FromDockerfile: its build-context tar builder mishandles
// ExcludePatterns+IncludeFiles together — verified empirically: the exact
// same Dockerfile/.dockerignore build cleanly with a plain `docker build`,
// but fail with "COPY failed: no source files were specified" when built
// through testcontainers-go v0.43.0's GetContext/archive.TarWithOptions
// path). Building explicitly via `docker build` first, then handing
// testcontainers an image tag, sidesteps the bug entirely and is also
// just faster: one build, reused across every test in this file, same
// as newTestPool(t)'s single shared Postgres container image pull. Run
// with `make test-integration` (requires Docker and network access to
// build the quant-engine image on first run).
package integration_test

import (
	"bytes"
	"context"
	"os/exec"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
	quantgrpc "trading-core/internal/infrastructure/external/quant/grpc"
	"trading-core/internal/infrastructure/observability/metrics"
)

const quantEngineTestImage = "quant-engine:trading-core-integration-test"

var buildQuantEngineImageOnce sync.Once

// buildQuantEngineImage builds ../../../quant-engine's Dockerfile once per
// test binary run (`sync.Once`) — every test in this package that needs
// the container reuses the same image tag instead of triggering its own
// build. The first build compiles quant-engine's Rust release binary and
// downloads ONNX Runtime (~10 minutes cold); Docker's own layer cache
// makes every build after the first one on a given machine fast.
func buildQuantEngineImage(t *testing.T) {
	t.Helper()
	buildQuantEngineImageOnce.Do(func() {
		cmd := exec.Command("docker", "build", "-t", quantEngineTestImage, ".")
		cmd.Dir = "../../../quant-engine"
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("building quant-engine image: %v\n%s", err, out.String())
		}
	})
}

// newQuantEngineContainer starts the real Rust quant-engine service from
// the image buildQuantEngineImage built, waiting for its HTTP /health
// endpoint (:8081) before returning — the gRPC server (:50051) comes up
// in the same process, so a healthy HTTP check is a reliable proxy for
// "gRPC is also ready."
func newQuantEngineContainer(t *testing.T) string {
	t.Helper()
	buildQuantEngineImage(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        quantEngineTestImage,
		ExposedPorts: []string{"50051/tcp", "8081/tcp"},
		Env: map[string]string{
			"QUANT_ENGINE__DATABASE__REQUIRED": "false",
			"QUANT_ENGINE__NATS__REQUIRED":     "false",
		},
		WaitingFor: wait.ForHTTP("/health").WithPort("8081/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting quant-engine container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminating quant-engine container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "50051/tcp")
	if err != nil {
		t.Fatalf("container mapped port: %v", err)
	}
	return host + ":" + port.Port()
}

func newTestQuantClient(t *testing.T, target string) *quantgrpc.Client {
	t.Helper()
	m, err := metrics.NewMetrics("trading-core-integration-test")
	if err != nil {
		t.Fatalf("building metrics: %v", err)
	}
	client, err := quantgrpc.NewClient(target, 10*time.Second, m)
	if err != nil {
		t.Fatalf("building quant grpc client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func testCandle(t *testing.T, symbol string, closeOffset time.Duration, closePrice int64) market.Candle {
	t.Helper()
	sym := shared.MustNewSymbol(symbol)
	timeframe := shared.MustNewTimeframe("1m")
	now := time.Now().Add(closeOffset)
	candle, err := market.NewClosedCandle(
		sym, timeframe,
		shared.MustNewPrice(decimal.NewFromInt(closePrice-1)),
		shared.MustNewPrice(decimal.NewFromInt(closePrice+1)),
		shared.MustNewPrice(decimal.NewFromInt(closePrice-2)),
		shared.MustNewPrice(decimal.NewFromInt(closePrice)),
		shared.MustNewQuantity(decimal.NewFromInt(10)),
		now.Add(-time.Minute), now,
	)
	if err != nil {
		t.Fatalf("building candle: %v", err)
	}
	return candle
}

// testCandleSeries builds n consecutive closed candles ending "now",
// trending upward — quant-engine's own regime detector needs at least 55
// candles of history, and its default `trend_following` strategy needs
// at least 228 (both discovered empirically against the real service, not
// documented anywhere reachable ahead of time).
func testCandleSeries(t *testing.T, symbol string, n int) []market.Candle {
	t.Helper()
	candles := make([]market.Candle, n)
	for i := range n {
		offset := -time.Duration(n-i) * time.Minute
		price := int64(50000 + i*10)
		candles[i] = testCandle(t, symbol, offset, price)
	}
	return candles
}

func TestQuantGrpcEvaluateSignalAgainstARealServer(t *testing.T) {
	target := newQuantEngineContainer(t)
	client := newTestQuantClient(t, target)
	ctx := context.Background()

	series := testCandleSeries(t, "BTCUSDT", 250)
	current := series[len(series)-1]
	recent := series[:len(series)-1]

	signal, err := client.EvaluateSignal(ctx, output.QuantInput{Candle: current, RecentCandles: recent})
	if err != nil {
		t.Fatalf("EvaluateSignal against real quant-engine: %v", err)
	}
	t.Logf("real EvaluateSignal response: %+v", signal)
	// The point of this test is that the call succeeds against the real
	// service and returns a well-formed response the client accepted —
	// not a specific trading decision, which depends on quant-engine's
	// own strategy logic.
}

func TestQuantGrpcAnalyzeMarketRegimeAgainstARealServer(t *testing.T) {
	target := newQuantEngineContainer(t)
	client := newTestQuantClient(t, target)
	ctx := context.Background()

	candles := testCandleSeries(t, "BTCUSDT", 60)

	regime, err := client.AnalyzeMarketRegime(ctx, output.MarketRegimeInput{
		Symbol:        shared.MustNewSymbol("BTCUSDT"),
		RecentCandles: candles,
	})
	if err != nil {
		t.Fatalf("AnalyzeMarketRegime against real quant-engine: %v", err)
	}
	t.Logf("real AnalyzeMarketRegime response: %s", regime)
}

func TestQuantGrpcBacktestSubmitAndPoll(t *testing.T) {
	target := newQuantEngineContainer(t)
	client := newTestQuantClient(t, target)
	ctx := context.Background()

	now := time.Now()
	candles := testCandleSeries(t, "BTCUSDT", 250)

	result, err := client.RunBacktest(ctx, output.RunBacktestInput{
		Config: output.BacktestConfig{
			Symbol:          "BTCUSDT",
			Timeframe:       "1m",
			StrategyName:    "trend_following",
			StrategyVersion: "1.0.0",
			InitialCapital:  "10000",
			FeeRate:         "0.0004",
			Slippage:        "0.0001",
			ExecutionModel:  "CLOSE_PRICE",
			StartAt:         now.Add(-250 * time.Minute),
			EndAt:           now,
		},
		Candles: candles,
	})
	if err != nil {
		t.Fatalf("RunBacktest against real quant-engine: %v", err)
	}
	if result.BacktestID == "" {
		t.Fatal("expected a non-empty backtest_id")
	}
	t.Logf("submitted backtest %s, status %s", result.BacktestID, result.Status)

	polled, err := client.GetBacktestResult(ctx, result.BacktestID)
	if err != nil {
		t.Fatalf("GetBacktestResult against real quant-engine: %v", err)
	}
	if polled.Status == output.JobStatusFailed {
		t.Fatalf("backtest failed: %s", polled.ErrorMessage)
	}
	t.Logf("polled backtest %s: status=%s, metrics=%+v", polled.BacktestID, polled.Status, polled.Metrics)
}

// TestQuantGrpcSurfacesUnavailableAsARecognizableGrpcStatus proves the
// server's error mapping survives the client, not just that some error
// comes back. Per docs/grpc-contract.md's error-mapping table,
// ApplicationError::ModelInference maps to UNAVAILABLE — and EvaluateModel
// documents this exact behavior when inference.provider=disabled, which
// is this container's default (.env.example's QUANT_ENGINE__INFERENCE__PROVIDER).
func TestQuantGrpcSurfacesUnavailableAsARecognizableGrpcStatus(t *testing.T) {
	target := newQuantEngineContainer(t)
	client := newTestQuantClient(t, target)
	ctx := context.Background()

	_, err := client.EvaluateModel(ctx, output.EvaluateModelInput{
		ModelName:            "primary",
		FeatureSchemaVersion: "v1",
		Features:             []output.FeatureValue{{Name: "rsi_14", Kind: "numeric", NumericValue: "55.0"}},
	})
	if err == nil {
		t.Fatal("expected an error: inference.provider=disabled by default")
	}
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("got grpc code %s, want Unavailable — the server's error mapping did not survive the client", status.Code(err))
	}
}
