package config

import (
	"fmt"
	"time"
)

// DatabaseConfig configures the PostgreSQL connection pool.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

func loadDatabaseConfig() (DatabaseConfig, error) {
	port, err := getEnvInt("DATABASE_PORT", 5432)
	if err != nil {
		return DatabaseConfig{}, err
	}
	maxConns, err := getEnvInt("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return DatabaseConfig{}, err
	}
	minConns, err := getEnvInt("DATABASE_MIN_CONNS", 2)
	if err != nil {
		return DatabaseConfig{}, err
	}
	maxConnLifetime, err := getEnvDuration("DATABASE_MAX_CONN_LIFETIME", time.Hour)
	if err != nil {
		return DatabaseConfig{}, err
	}
	maxConnIdleTime, err := getEnvDuration("DATABASE_MAX_CONN_IDLE_TIME", 30*time.Minute)
	if err != nil {
		return DatabaseConfig{}, err
	}
	connectTimeout, err := getEnvDuration("DATABASE_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return DatabaseConfig{}, err
	}

	return DatabaseConfig{
		Host:            getEnv("DATABASE_HOST", "localhost"),
		Port:            port,
		User:            getEnv("DATABASE_USER", "trading"),
		Password:        getEnv("DATABASE_PASSWORD", "trading"),
		Name:            getEnv("DATABASE_NAME", "trading_core"),
		SSLMode:         getEnv("DATABASE_SSL_MODE", "disable"),
		MaxConns:        int32(maxConns),
		MinConns:        int32(minConns),
		MaxConnLifetime: maxConnLifetime,
		MaxConnIdleTime: maxConnIdleTime,
		ConnectTimeout:  connectTimeout,
	}, nil
}

// DSN builds a libpq/pgx connection string. Never logged as-is; use
// Redacted() for logging.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode, int(c.ConnectTimeout.Seconds()),
	)
}

// Redacted returns a connection string safe for logs (password masked).
func (c DatabaseConfig) Redacted() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, mask(c.Password), c.Host, c.Port, c.Name, c.SSLMode,
	)
}
