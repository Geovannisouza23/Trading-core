package config

// NatsConfig configures the NATS JetStream event bus adapter. Mirrors
// quant-engine's own config::nats::NatsConfig exactly: optional by
// default, never blocks startup — an unreachable broker with
// Required=false just falls back to the in-memory EventBus.
type NatsConfig struct {
	URL          string
	Required     bool
	StreamPrefix string
}

func loadNatsConfig() (NatsConfig, error) {
	required, err := getEnvBool("NATS_REQUIRED", false)
	if err != nil {
		return NatsConfig{}, err
	}
	return NatsConfig{
		URL:          getEnv("NATS_URL", "nats://localhost:4222"),
		Required:     required,
		StreamPrefix: getEnv("NATS_STREAM_PREFIX", "quant"),
	}, nil
}
