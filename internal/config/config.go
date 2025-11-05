package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port     string
	Env      string
	LogLevel string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// CORS
	CORSAllowedOrigins []string

	// Matchmaking
	MatchmakingInterval time.Duration
	MaxELODifference    int

	// Executor Service
	ExecutorURL string
}

func Load() (*Config, error) {
	// .env 파일 로드 (있는 경우)
	_ = godotenv.Load()

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		Env:                 getEnv("ENV", "development"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:           getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiration:       parseDuration(getEnv("JWT_EXPIRATION", "24h")),
		MatchmakingInterval: parseDuration(getEnv("MATCHMAKING_INTERVAL", "10s")),
		MaxELODifference:    200,
		ExecutorURL:         getEnv("EXECUTOR_URL", "http://localhost:8081"),
		CORSAllowedOrigins:  []string{"http://localhost:3000", "http://localhost:5173"},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}
