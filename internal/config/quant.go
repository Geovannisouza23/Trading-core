package config

import "time"

// QuantConfig configures the Quant Engine adapter. Mode defaults to "fake"
// so the system runs without the (not-yet-implemented) Rust service.
type QuantConfig struct {
	Mode       string // fake | grpc
	GRPCTarget string
	Timeout    time.Duration
}

func loadQuantConfig() (QuantConfig, error) {
	timeout, err := getEnvDuration("QUANT_TIMEOUT", 5*time.Second)
	if err != nil {
		return QuantConfig{}, err
	}
	return QuantConfig{
		Mode:       getEnv("QUANT_MODE", "fake"),
		GRPCTarget: getEnv("QUANT_GRPC_TARGET", "localhost:50051"),
		Timeout:    timeout,
	}, nil
}
