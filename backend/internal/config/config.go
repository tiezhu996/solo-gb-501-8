package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment   string
	Port          string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	JWTSecret     string
	TokenTTL      time.Duration
	RateLimit     int64
	RateWindow    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:   env("APP_ENV", "development"),
		Port:          env("PORT", "8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://sterile:sterile_dev_password@localhost:5432/sterile_release?sslmode=disable"),
		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		JWTSecret:     env("JWT_SECRET", "local-development-secret-change-me"),
		TokenTTL:      durationEnv("TOKEN_TTL", 12*time.Hour),
		RateLimit:     int64Env("RATE_LIMIT", 180),
		RateWindow:    durationEnv("RATE_WINDOW", time.Minute),
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET cannot be empty")
	}
	if cfg.Environment == "production" && (len(cfg.JWTSecret) < 32 || cfg.JWTSecret == "local-development-secret-change-me") {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters and non-default in production")
	}
	redisDB, err := strconv.Atoi(env("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	cfg.RedisDB = redisDB
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func int64Env(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
