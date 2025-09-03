package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL        string
	GRPCPort     string
	HTTPPort     string
	RedisAddr    string
	KafkaBrokers string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	return &Config{
		DBURL:        os.Getenv("DB_URL"),        // e.g., postgres://user:pass@db:5432/hrdb
		GRPCPort:     os.Getenv("GRPC_PORT"),     // e.g., :50051
		HTTPPort:     os.Getenv("HTTP_PORT"),     // e.g., :8080
		RedisAddr:    os.Getenv("REDIS_ADDR"),    // e.g., redis:6379
		KafkaBrokers: os.Getenv("KAFKA_BROKERS"), // e.g., kafka:9092
	}, nil
}
