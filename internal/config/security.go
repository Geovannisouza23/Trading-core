package config

import "time"

// SecurityConfig configures JWT auth, RBAC defaults, and notification
// credentials.
type SecurityConfig struct {
	JWTSigningSecret  string
	JWTIssuer         string
	JWTAccessTokenTTL time.Duration

	TelegramBotToken string
	TelegramChatID   string

	AllowedNewsSources  []string
	MaxNewsPayloadBytes int
}

func loadSecurityConfig() (SecurityConfig, error) {
	ttl, err := getEnvDuration("SECURITY_JWT_ACCESS_TOKEN_TTL", time.Hour)
	if err != nil {
		return SecurityConfig{}, err
	}
	maxNewsPayload, err := getEnvInt("SECURITY_MAX_NEWS_PAYLOAD_BYTES", 8*1024)
	if err != nil {
		return SecurityConfig{}, err
	}
	return SecurityConfig{
		JWTSigningSecret:  getEnv("SECURITY_JWT_SIGNING_SECRET", ""),
		JWTIssuer:         getEnv("SECURITY_JWT_ISSUER", "trading-core"),
		JWTAccessTokenTTL: ttl,
		TelegramBotToken:  getEnv("SECURITY_TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    getEnv("SECURITY_TELEGRAM_CHAT_ID", ""),
		AllowedNewsSources: getEnvStringSlice("SECURITY_ALLOWED_NEWS_SOURCES", []string{
			"reuters.com", "bloomberg.com", "coindesk.com", "theblock.co",
		}),
		MaxNewsPayloadBytes: maxNewsPayload,
	}, nil
}

// Redacted returns a copy safe for structured logging.
func (c SecurityConfig) Redacted() SecurityConfig {
	c.JWTSigningSecret = mask(c.JWTSigningSecret)
	c.TelegramBotToken = mask(c.TelegramBotToken)
	return c
}
