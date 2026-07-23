package config

// ObservabilityConfig configures logging, tracing and metrics.
type ObservabilityConfig struct {
	ServiceName    string
	Environment    string
	LogLevel       string // debug | info | warn | error
	LogFormat      string // json | text
	TracingEnabled bool
	MetricsEnabled bool
	MetricsPath    string
}

func loadObservabilityConfig() (ObservabilityConfig, error) {
	tracingEnabled, err := getEnvBool("OBSERVABILITY_TRACING_ENABLED", true)
	if err != nil {
		return ObservabilityConfig{}, err
	}
	metricsEnabled, err := getEnvBool("OBSERVABILITY_METRICS_ENABLED", true)
	if err != nil {
		return ObservabilityConfig{}, err
	}
	return ObservabilityConfig{
		ServiceName:    getEnv("OBSERVABILITY_SERVICE_NAME", "trading-core"),
		Environment:    getEnv("OBSERVABILITY_ENVIRONMENT", "development"),
		LogLevel:       getEnv("OBSERVABILITY_LOG_LEVEL", "info"),
		LogFormat:      getEnv("OBSERVABILITY_LOG_FORMAT", "json"),
		TracingEnabled: tracingEnabled,
		MetricsEnabled: metricsEnabled,
		MetricsPath:    getEnv("OBSERVABILITY_METRICS_PATH", "/metrics"),
	}, nil
}
