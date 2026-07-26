package config

// RedisConfig configures the Redis client used for market-data caching
// and shared cross-service idempotency dedup. Optional by default — an
// unreachable Redis with Required=false just disables caching/dedup,
// never blocks startup or a request.
type RedisConfig struct {
	Addr     string
	Required bool
}

func loadRedisConfig() (RedisConfig, error) {
	required, err := getEnvBool("REDIS_REQUIRED", false)
	if err != nil {
		return RedisConfig{}, err
	}
	return RedisConfig{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Required: required,
	}, nil
}
