// Package grpc holds both the deterministic in-process fake (fake.go,
// output.QuantEngine's default implementation) and the real gRPC client
// (this file) that talks to the external Rust quant-engine over the
// vendored contract at internal/contracts/grpc/quant/v1. Same package,
// same output.QuantEngine interface, per ADR 0008/0010: swapping one for
// the other is a one-line change in internal/app/providers.go, no use
// case or risk logic changes.
package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"trading-core/internal/application/ports/output"
	quantv1 "trading-core/internal/contracts/grpc/quant/v1"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
	"trading-core/internal/infrastructure/observability/metrics"
)

// Client is the real QuantEngineService client. TLS is not configured —
// both trading-core and quant-engine default to plaintext transport
// today (see docs/decisions/0010-real-grpc-client-for-quant-engine.md);
// enabling TLS on the channel is a deploy-time follow-up, not addressed
// here.
type Client struct {
	conn    *grpc.ClientConn
	stub    quantv1.QuantEngineServiceClient
	timeout time.Duration
	metrics *metrics.Metrics
}

// NewClient dials target (host:port, e.g. "localhost:50051") and returns a
// Client ready to use. Dialing with grpc.NewClient is lazy/non-blocking:
// this does not itself prove the server is reachable, the first RPC call
// does.
func NewClient(target string, timeout time.Duration, m *metrics.Metrics) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dialing quant engine at %s: %w", target, err)
	}
	return &Client{
		conn:    conn,
		stub:    quantv1.NewQuantEngineServiceClient(conn),
		timeout: timeout,
		metrics: m,
	}, nil
}

// Close releases the underlying connection. Registered as an
// fx.Lifecycle.OnStop hook by internal/app/providers.go, same pattern as
// the Postgres pool's shutdown.
func (c *Client) Close() error {
	return c.conn.Close()
}

var _ output.QuantEngine = (*Client)(nil)

func (c *Client) recordLatency(start time.Time) {
	if c.metrics == nil || c.metrics.QuantEngineLatency == nil {
		return
	}
	c.metrics.QuantEngineLatency.Record(context.Background(), time.Since(start).Seconds())
}

func (c *Client) EvaluateSignal(ctx context.Context, input output.QuantInput) (output.QuantSignal, error) {
	defer c.recordLatency(time.Now())

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	allCandles := make([]market.Candle, 0, len(input.RecentCandles)+1)
	allCandles = append(allCandles, input.RecentCandles...)
	allCandles = append(allCandles, input.Candle)

	req := &quantv1.EvaluateSignalRequest{
		RequestId:     uuid.NewString(),
		CorrelationId: uuid.NewString(),
		Symbol:        input.Candle.Symbol.String(),
		Timeframe:     input.Candle.Timeframe.String(),
		Candles:       candlesToProto(allCandles),
	}

	resp, err := c.stub.EvaluateSignal(ctx, req)
	if err != nil {
		return output.QuantSignal{}, fmt.Errorf("quant engine EvaluateSignal: %w", err)
	}

	return quantSignalFromProto(input.Candle, resp)
}

func quantSignalFromProto(candle market.Candle, resp *quantv1.EvaluateSignalResponse) (output.QuantSignal, error) {
	var side shared.Side
	switch resp.GetAction() {
	case quantv1.Action_ACTION_BUY:
		side = shared.SideBuy
	case quantv1.Action_ACTION_SELL:
		side = shared.SideSell
	default:
		// ACTION_HOLD or ACTION_UNSPECIFIED: no signal.
		return output.QuantSignal{HasSignal: false}, nil
	}

	entryValue, err := decimalFromWire("entry_price", resp.GetEntryPrice())
	if err != nil {
		return output.QuantSignal{}, err
	}
	entry, err := shared.NewPrice(entryValue)
	if err != nil {
		return output.QuantSignal{}, fmt.Errorf("quant engine returned an invalid entry price: %w", err)
	}

	stopValue, err := decimalFromWire("stop_price", resp.GetStopPrice())
	if err != nil {
		return output.QuantSignal{}, err
	}
	stop, err := shared.NewPrice(stopValue)
	if err != nil {
		return output.QuantSignal{}, fmt.Errorf("quant engine returned an invalid stop price: %w", err)
	}

	targetValue, err := decimalFromWire("target_price", resp.GetTargetPrice())
	if err != nil {
		return output.QuantSignal{}, err
	}
	target, err := shared.NewPrice(targetValue)
	if err != nil {
		return output.QuantSignal{}, fmt.Errorf("quant engine returned an invalid target price: %w", err)
	}

	confidenceValue, err := decimalFromWire("adjusted_confidence", resp.GetAdjustedConfidence())
	if err != nil {
		return output.QuantSignal{}, err
	}
	confidence, err := shared.NewConfidence(confidenceValue)
	if err != nil {
		return output.QuantSignal{}, fmt.Errorf("quant engine returned an invalid confidence: %w", err)
	}

	return output.QuantSignal{
		HasSignal:    true,
		Symbol:       candle.Symbol,
		Side:         side,
		EntryPrice:   entry,
		StopPrice:    stop,
		TargetPrice:  target,
		Confidence:   confidence,
		StrategyName: resp.GetStrategy(),
		Regime:       regimeFromProto(resp.GetMarketRegime()),
	}, nil
}

func (c *Client) AnalyzeMarketRegime(ctx context.Context, input output.MarketRegimeInput) (market.Regime, error) {
	defer c.recordLatency(time.Now())

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	timeframe := ""
	if len(input.RecentCandles) > 0 {
		timeframe = input.RecentCandles[0].Timeframe.String()
	}

	req := &quantv1.AnalyzeMarketRegimeRequest{
		RequestId: uuid.NewString(),
		Symbol:    input.Symbol.String(),
		Timeframe: timeframe,
		Candles:   candlesToProto(input.RecentCandles),
	}

	resp, err := c.stub.AnalyzeMarketRegime(ctx, req)
	if err != nil {
		return market.RegimeUnknown, fmt.Errorf("quant engine AnalyzeMarketRegime: %w", err)
	}

	return regimeFromProto(resp.GetRegime()), nil
}
