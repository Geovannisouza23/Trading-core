// Package tracing wires the OpenTelemetry SDK tracer provider. It defaults
// to a stdout exporter so the system has meaningful traces with zero extra
// infrastructure; swapping in an OTLP exporter later is a one-function
// change contained entirely to this package.
package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"trading-core/internal/config"
)

// NewTracerProvider builds and registers a global TracerProvider. When
// tracing is disabled in configuration, it installs a no-op provider so
// callers never need to branch on whether tracing is active.
func NewTracerProvider(ctx context.Context, cfg config.ObservabilityConfig) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithResource(res))

	if cfg.TracingEnabled {
		exporter, err := stdouttrace.New(stdouttrace.WithWriter(os.Stdout), stdouttrace.WithoutTimestamps())
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	} else {
		opts = append(opts, sdktrace.WithSampler(sdktrace.NeverSample()))
	}

	provider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(provider)
	return provider, nil
}
