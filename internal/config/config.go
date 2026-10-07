package config

import (
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
	KafkaBrokers                string
	KafkaTopicSpans             string
	KafkaMinBytes               int
	KafkaAllowAutoTopicCreation bool
	KafkaAnalysisWorkers        int
	KafkaAnalysisQueueCapacity  int

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

	// Phase 5A LLM Explanation & Cost Attribution
	LLMModelName              string
	LLMAPIKey                 string
	LLMInputPricePerMillion   float64
	LLMOutputPricePerMillion  float64
	LLMCostCeilingPerIncident float64
	LLMHourlySpendCap         float64
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

	cfg.Environment = getEnv("ENVIRONMENT", "development")

	if cfg.Environment == "production" {
		cfg.DatabaseURL, err = getRequiredEnv("DATABASE_URL")
		if err != nil {
			return nil, err
		}

		cfg.ClickHouseURL, err = getRequiredEnv("CLICKHOUSE_URL")
		if err != nil {
			return nil, err
		}

		cfg.RedisURL, err = getRequiredEnv("REDIS_URL")
		if err != nil {
			return nil, err
		}

		cfg.KafkaBrokers, err = getRequiredEnv("KAFKA_BROKERS")
		if err != nil {
			return nil, err
		}

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

		cfg.AWSRegion, err = getRequiredEnv("AWS_REGION")
		if err != nil {
			return nil, err
		}

		cfg.AWSSecretsARN, err = getRequiredEnv("AWS_SECRETS_ARN")
		if err != nil {
			return nil, err
		}
	} else {
		cfg.DatabaseURL = getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/distributedtrace?sslmode=disable")
		cfg.ClickHouseURL = getEnv("CLICKHOUSE_URL", "127.0.0.1:9000")
		cfg.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
		cfg.KafkaBrokers = getEnv("KAFKA_BROKERS", "localhost:9092")
		cfg.JWTPrivateKeyPath = getEnv("JWT_PRIVATE_KEY_PATH", "")
		cfg.JWTPublicKeyPath = getEnv("JWT_PUBLIC_KEY_PATH", "")
		cfg.AllowedOrigin = getEnv("ALLOWED_ORIGIN", "http://localhost:5173")
		cfg.AWSRegion = getEnv("AWS_REGION", "us-east-1")
		cfg.AWSSecretsARN = getEnv("AWS_SECRETS_ARN", "")
	}

	cfg.ClickHouseDB = getEnv("CLICKHOUSE_DB", "default")
	cfg.KafkaTopicSpans = getEnv("KAFKA_TOPIC_SPANS", "otel-spans")

	cfg.KafkaMinBytes, err = getEnvInt("KAFKA_MIN_BYTES", 10240)
	if err != nil {
		return nil, err
	}
	cfg.KafkaAllowAutoTopicCreation = getEnv("KAFKA_ALLOW_AUTO_TOPIC_CREATION", "false") == "true"
	cfg.KafkaAnalysisWorkers, err = getEnvInt("KAFKA_ANALYSIS_WORKERS", 2)
	if err != nil {
		return nil, err
	}
	cfg.KafkaAnalysisQueueCapacity, err = getEnvInt("KAFKA_ANALYSIS_QUEUE_CAPACITY", 32)
	if err != nil {
		return nil, err
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

	cfg.LLMModelName = getEnv("LLM_MODEL_NAME", "gemini-3.5-flash-lite")
	cfg.LLMAPIKey = getEnv("LLM_API_KEY", getEnv("GEMINI_API_KEY", ""))
	cfg.LLMInputPricePerMillion, err = getEnvFloat("LLM_INPUT_PRICE_PER_MILLION", 0.075)
	if err != nil {
		return nil, err
	}
	cfg.LLMOutputPricePerMillion, err = getEnvFloat("LLM_OUTPUT_PRICE_PER_MILLION", 0.300)
	if err != nil {
		return nil, err
	}
	cfg.LLMCostCeilingPerIncident, err = getEnvFloat("LLM_COST_CEILING_PER_INCIDENT", 0.010)
	if err != nil {
		return nil, err
	}
	cfg.LLMHourlySpendCap, err = getEnvFloat("LLM_HOURLY_SPEND_CAP", 1.000)
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

func getEnvFloat(key string, defaultVal float64) (float64, error) {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal, nil
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value for %s: %w", key, err)
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
