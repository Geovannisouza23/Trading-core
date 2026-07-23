package config

const insecureDefaultJWTSecret = "dev-only-insecure-default-secret-change-me-32chars"

// Config is the fully-loaded, fully-validated application configuration.
// It is built once at startup by Load and never mutated afterward; every
// package that needs configuration receives this struct (or a sub-struct of
// it) by value/pointer injection instead of reading the environment itself.
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Broker        BrokerConfig
	Risk          RiskConfig
	LLM           LLMConfig
	Quant         QuantConfig
	Observability ObservabilityConfig
	Security      SecurityConfig

	// UsingInsecureDefaultJWTSecret is true when no explicit JWT secret was
	// configured and the built-in development fallback is in use. The
	// observability/logging bootstrap should emit a loud warning when this
	// is true outside of PAPER mode.
	UsingInsecureDefaultJWTSecret bool
}

// Load reads every environment variable this application understands,
// applies safe defaults, and validates the result. It is the only exported
// entry point into this package's env access.
func Load() (*Config, error) {
	server, err := loadServerConfig()
	if err != nil {
		return nil, err
	}
	database, err := loadDatabaseConfig()
	if err != nil {
		return nil, err
	}
	broker, err := loadBrokerConfig()
	if err != nil {
		return nil, err
	}
	risk, err := loadRiskConfig()
	if err != nil {
		return nil, err
	}
	llm, err := loadLLMConfig()
	if err != nil {
		return nil, err
	}
	quant, err := loadQuantConfig()
	if err != nil {
		return nil, err
	}
	observability, err := loadObservabilityConfig()
	if err != nil {
		return nil, err
	}
	security, err := loadSecurityConfig()
	if err != nil {
		return nil, err
	}

	usingDefault := security.JWTSigningSecret == ""
	if usingDefault {
		security.JWTSigningSecret = insecureDefaultJWTSecret
	}

	cfg := &Config{
		Server:                        server,
		Database:                      database,
		Broker:                        broker,
		Risk:                          risk,
		LLM:                           llm,
		Quant:                         quant,
		Observability:                 observability,
		Security:                      security,
		UsingInsecureDefaultJWTSecret: usingDefault,
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
