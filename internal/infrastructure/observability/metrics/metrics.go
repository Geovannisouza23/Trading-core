// Package metrics wires OpenTelemetry metric instruments to a Prometheus
// exporter, implementing every metric named in spec section 28.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics holds every instrument the application records against.
type Metrics struct {
	Provider *sdkmetric.MeterProvider
	Handler  http.Handler

	OrdersCreated             metric.Int64Counter
	OrdersSubmitted           metric.Int64Counter
	OrdersRejected            metric.Int64Counter
	OrdersFailed              metric.Int64Counter
	OrdersUnknown             metric.Int64Counter
	RiskDecisions             metric.Int64Counter
	RiskBlocks                metric.Int64Counter
	SignalsCreated            metric.Int64Counter
	ReconciliationDivergences metric.Int64Counter
	ConsumerFailures          metric.Int64Counter

	BrokerLatency      metric.Float64Histogram
	QuantEngineLatency metric.Float64Histogram
	LLMLatency         metric.Float64Histogram

	ActivePositions      metric.Int64Gauge
	CurrentDrawdown      metric.Float64Gauge
	DailyPnL             metric.Float64Gauge
	WeeklyPnL            metric.Float64Gauge
	SystemMode           metric.Int64Gauge
	OutboxPendingEvents  metric.Int64Gauge
	OutboxFailedEvents   metric.Int64Gauge
	WebSocketConnections metric.Int64Gauge
}

// NewMetrics builds a Prometheus-backed MeterProvider and every named
// instrument. The returned Handler should be mounted at GET /metrics.
func NewMetrics(serviceName string) (*Metrics, error) {
	registry := prometheus.NewRegistry()
	exporter, err := otelprometheus.New(otelprometheus.WithRegisterer(registry))
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := provider.Meter(serviceName)

	m := &Metrics{
		Provider: provider,
		Handler:  promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}

	m.OrdersCreated, err = meter.Int64Counter("orders_created_total")
	if err != nil {
		return nil, err
	}
	m.OrdersSubmitted, err = meter.Int64Counter("orders_submitted_total")
	if err != nil {
		return nil, err
	}
	m.OrdersRejected, err = meter.Int64Counter("orders_rejected_total")
	if err != nil {
		return nil, err
	}
	m.OrdersFailed, err = meter.Int64Counter("orders_failed_total")
	if err != nil {
		return nil, err
	}
	m.OrdersUnknown, err = meter.Int64Counter("orders_unknown_total")
	if err != nil {
		return nil, err
	}
	m.RiskDecisions, err = meter.Int64Counter("risk_decisions_total")
	if err != nil {
		return nil, err
	}
	m.RiskBlocks, err = meter.Int64Counter("risk_blocks_total")
	if err != nil {
		return nil, err
	}
	m.SignalsCreated, err = meter.Int64Counter("signals_created_total")
	if err != nil {
		return nil, err
	}
	m.ReconciliationDivergences, err = meter.Int64Counter("reconciliation_divergences_total")
	if err != nil {
		return nil, err
	}
	m.ConsumerFailures, err = meter.Int64Counter("consumer_failures_total")
	if err != nil {
		return nil, err
	}

	m.BrokerLatency, err = meter.Float64Histogram("broker_latency_seconds")
	if err != nil {
		return nil, err
	}
	m.QuantEngineLatency, err = meter.Float64Histogram("quant_engine_latency_seconds")
	if err != nil {
		return nil, err
	}
	m.LLMLatency, err = meter.Float64Histogram("llm_latency_seconds")
	if err != nil {
		return nil, err
	}

	m.ActivePositions, err = meter.Int64Gauge("active_positions")
	if err != nil {
		return nil, err
	}
	m.CurrentDrawdown, err = meter.Float64Gauge("current_drawdown")
	if err != nil {
		return nil, err
	}
	m.DailyPnL, err = meter.Float64Gauge("daily_pnl")
	if err != nil {
		return nil, err
	}
	m.WeeklyPnL, err = meter.Float64Gauge("weekly_pnl")
	if err != nil {
		return nil, err
	}
	// SystemMode is recorded as a fixed value of 1 tagged with a "mode"
	// attribute on every transition; consumers should treat the most
	// recently scraped sample per series as authoritative.
	m.SystemMode, err = meter.Int64Gauge("system_mode")
	if err != nil {
		return nil, err
	}
	m.OutboxPendingEvents, err = meter.Int64Gauge("outbox_pending_events")
	if err != nil {
		return nil, err
	}
	m.OutboxFailedEvents, err = meter.Int64Gauge("outbox_failed_events")
	if err != nil {
		return nil, err
	}
	m.WebSocketConnections, err = meter.Int64Gauge("websocket_connections")
	if err != nil {
		return nil, err
	}

	return m, nil
}
