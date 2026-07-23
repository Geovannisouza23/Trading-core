package config

import "time"

// LLMConfig configures the event-intelligence LLM adapter. Provider
// defaults to "noop" so the system runs with zero external credentials.
type LLMConfig struct {
	Provider        string // noop | gemini
	GeminiAPIKey    string
	GeminiModel     string
	GeminiBaseURL   string
	RequestTimeout  time.Duration
	MaxPayloadBytes int
}

func loadLLMConfig() (LLMConfig, error) {
	timeout, err := getEnvDuration("LLM_REQUEST_TIMEOUT", 8*time.Second)
	if err != nil {
		return LLMConfig{}, err
	}
	maxPayload, err := getEnvInt("LLM_MAX_PAYLOAD_BYTES", 16*1024)
	if err != nil {
		return LLMConfig{}, err
	}
	return LLMConfig{
		Provider:        getEnv("LLM_PROVIDER", "noop"),
		GeminiAPIKey:    getEnv("LLM_GEMINI_API_KEY", ""),
		GeminiModel:     getEnv("LLM_GEMINI_MODEL", "gemini-2.5-flash"),
		GeminiBaseURL:   getEnv("LLM_GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"),
		RequestTimeout:  timeout,
		MaxPayloadBytes: maxPayload,
	}, nil
}

// Redacted returns a copy safe for structured logging.
func (c LLMConfig) Redacted() LLMConfig {
	c.GeminiAPIKey = mask(c.GeminiAPIKey)
	return c
}
