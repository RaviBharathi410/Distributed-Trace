package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort         int
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ServerIdleTimeout  time.Duration

	DatabaseURL       string
	ClickHouseURL     string
	ClickHouseDB      string
	RedisURL          string
	KafkaBrokers      string
	KafkaTopicSpans   string

	JWTPrivateKeyPath string
	JWTPublicKeyPath  string

	AllowedOrigin     string
	AWSRegion         string
	AWSSecretsARN     string

	LogLevel    string
	LogFormat   string
	Environment string

	MaxWSConnections int
	RateLimitAuth    int
	RateLimitAPI     int
}

func Load() (*Config, error) {
	// Optional load for local development .env
	_ = godotenv.Load()

	cfg := &Config{}

	var err error

	cfg.ServerPort, err = getEnvInt("SERVER_PORT", 8080)
	if err != nil {
		return nil, err
	}

	cfg.ServerReadTimeout, err = getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	cfg.ServerWriteTimeout, err = getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}

	cfg.ServerIdleTimeout, err = getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second)
	if err != nil {
		return nil, err
	}

	cfg.DatabaseURL, err = getRequiredEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	cfg.ClickHouseURL, err = getRequiredEnv("CLICKHOUSE_URL")
	if err != nil {
		return nil, err
	}

	cfg.ClickHouseDB = getEnv("CLICKHOUSE_DB", "default")

	cfg.RedisURL, err = getRequiredEnv("REDIS_URL")
	if err != nil {
		return nil, err
	}

	cfg.KafkaBrokers, err = getRequiredEnv("KAFKA_BROKERS")
	if err != nil {
		return nil, err
	}

	cfg.KafkaTopicSpans = getEnv("KAFKA_TOPIC_SPANS", "otel-spans")

	cfg.JWTPrivateKeyPath, err = getRequiredEnv("JWT_PRIVATE_KEY_PATH")
	if err != nil {
		return nil, err
	}

	cfg.JWTPublicKeyPath, err = getRequiredEnv("JWT_PUBLIC_KEY_PATH")
	if err != nil {
		return nil, err
	}

	cfg.AllowedOrigin, err = getRequiredEnv("ALLOWED_ORIGIN")
	if err != nil {
		return nil, err
	}

	cfg.Environment, err = getRequiredEnv("ENVIRONMENT")
	if err != nil {
		return nil, err
	}

	if cfg.Environment == "production" {
		cfg.AWSRegion, err = getRequiredEnv("AWS_REGION")
		if err != nil {
			return nil, err
		}
		cfg.AWSSecretsARN, err = getRequiredEnv("AWS_SECRETS_ARN")
		if err != nil {
			return nil, err
		}
	} else {
		cfg.AWSRegion = getEnv("AWS_REGION", "us-east-1")
		cfg.AWSSecretsARN = getEnv("AWS_SECRETS_ARN", "")
	}

	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.LogFormat = getEnv("LOG_FORMAT", "json")

	cfg.MaxWSConnections, err = getEnvInt("MAX_WS_CONNECTIONS", 1000)
	if err != nil {
		return nil, err
	}

	cfg.RateLimitAuth, err = getEnvInt("RATE_LIMIT_AUTH", 10)
	if err != nil {
		return nil, err
	}

	cfg.RateLimitAPI, err = getEnvInt("RATE_LIMIT_API", 300)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getRequiredEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return val, nil
}

func getEnvInt(key string, defaultVal int) (int, error) {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid int value for %s: %w", key, err)
	}
	return val, nil
}

func getEnvDuration(key string, defaultVal time.Duration) (time.Duration, error) {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal, nil
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value for %s: %w", key, err)
	}
	return val, nil
}

// AWS Secrets Manager placeholder config loader
func LoadSecrets(cfg *Config) error {
	if cfg.AWSSecretsARN == "" {
		return nil
	}
	// AWS Secrets Manager loading logic would reside here in real environment
	// using the Go SDK. For now we will return nil to allow configuration fallbacks.
	return nil
}
