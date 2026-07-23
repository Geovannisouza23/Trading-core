package config

import (
	"time"

	"github.com/shopspring/decimal"
)

// ServerConfig configures the HTTP/WebSocket server.
type ServerConfig struct {
	Host                string
	Port                int
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	RequestTimeout      time.Duration
	ShutdownTimeout     time.Duration
	MaxRequestBodyBytes int64
	CORSAllowedOrigins  []string
	RateLimitRPS        float64
	RateLimitBurst      int
}

func loadServerConfig() (ServerConfig, error) {
	port, err := getEnvInt("SERVER_PORT", 8080)
	if err != nil {
		return ServerConfig{}, err
	}
	readTimeout, err := getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}
	writeTimeout, err := getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}
	idleTimeout, err := getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}
	requestTimeout, err := getEnvDuration("SERVER_REQUEST_TIMEOUT", 15*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}
	shutdownTimeout, err := getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}
	maxBodyBytes, err := getEnvInt("SERVER_MAX_REQUEST_BODY_BYTES", 1<<20)
	if err != nil {
		return ServerConfig{}, err
	}
	rateLimitRPS, err := getEnvDecimal("SERVER_RATE_LIMIT_RPS", decimal.NewFromInt(20))
	if err != nil {
		return ServerConfig{}, err
	}
	rateLimitBurst, err := getEnvInt("SERVER_RATE_LIMIT_BURST", 40)
	if err != nil {
		return ServerConfig{}, err
	}

	rps, _ := rateLimitRPS.Float64()

	return ServerConfig{
		Host:                getEnv("SERVER_HOST", "0.0.0.0"),
		Port:                port,
		ReadTimeout:         readTimeout,
		WriteTimeout:        writeTimeout,
		IdleTimeout:         idleTimeout,
		RequestTimeout:      requestTimeout,
		ShutdownTimeout:     shutdownTimeout,
		MaxRequestBodyBytes: int64(maxBodyBytes),
		CORSAllowedOrigins:  getEnvStringSlice("SERVER_CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		RateLimitRPS:        rps,
		RateLimitBurst:      rateLimitBurst,
	}, nil
}
