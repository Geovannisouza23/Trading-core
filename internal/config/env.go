// Package config is the single place in the codebase allowed to read
// process environment variables. Every other package receives configuration
// as already-parsed, already-validated Go values.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvRequired(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("missing required environment variable %s", key)
	}
	return v, nil
}

func getEnvInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func getEnvBool(key string, fallback bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("environment variable %s must be a boolean: %w", key, err)
	}
	return parsed, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s must be a Go duration (e.g. 30s): %w", key, err)
	}
	return parsed, nil
}

func getEnvDecimal(key string, fallback decimal.Decimal) (decimal.Decimal, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	parsed, err := decimal.NewFromString(v)
	if err != nil {
		return decimal.Zero, fmt.Errorf("environment variable %s must be a decimal number: %w", key, err)
	}
	return parsed, nil
}

func getEnvStringSlice(key string, fallback []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// mask redacts a secret for safe logging, keeping only a short prefix/suffix
// so operators can still tell which credential is configured.
func mask(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 6 {
		return "****"
	}
	return secret[:2] + "****" + secret[len(secret)-2:]
}
