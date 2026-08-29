package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// PostgreSQL
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	DBSSLMode  string
	DBMaxConns int

	// Redis
	RedisHost     string
	RedisPassword string
	RedisDB       int

	// ClickHouse
	CHHost string
	CHUser string
	CHPass string
	CHDB   string

	// Kafka
	KafkaBrokers string
	KafkaTopic   string
	KafkaGroup   string

	// Application
	LogLevel       string
	AppEnv         string
	ServerPort     string
	IdempotencyTTL time.Duration
	StockCacheTTL  time.Duration
}

func Load() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "inventory"),
		DBPass:     getEnv("DB_PASS", "inventory"),
		DBName:     getEnv("DB_NAME", "inventory"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBMaxConns: getEnvAsInt("DB_MAX_CONNS", 10),

		RedisHost:     getEnv("REDIS_HOST", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		CHHost: getEnv("CLICKHOUSE_HOST", "localhost:9000"),
		CHUser: getEnv("CLICKHOUSE_USER", "inventory"),
		CHPass: getEnv("CLICKHOUSE_PASS", "inventory"),
		CHDB:   getEnv("CLICKHOUSE_DB", "analytics"),

		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "stock-events"),
		KafkaGroup:   getEnv("KAFKA_GROUP", "stock-worker-group"),

		LogLevel:       getEnv("LOG_LEVEL", "info"),
		AppEnv:         getEnv("APP_ENV", "development"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		IdempotencyTTL: getEnvAsDuration("IDEMPOTENCY_TTL", 5*time.Minute),
		StockCacheTTL:  getEnvAsDuration("STOCK_CACHE_TTL", 5*time.Minute),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if durVal, err := time.ParseDuration(value); err == nil {
			return durVal
		}
	}
	return defaultValue
}
