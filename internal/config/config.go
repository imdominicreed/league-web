package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port        string
	Environment string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret          string
	JWTExpirationHours int

	// Draft
	DefaultTimerDuration time.Duration

	// Data Dragon
	DataDragonVersion string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		Environment:          getEnv("ENVIRONMENT", "development"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5431/league_draft?sslmode=disable"),
		JWTSecret:            getEnv("JWT_SECRET", ""),
		JWTExpirationHours:   getEnvInt("JWT_EXPIRATION_HOURS", 24),
		DefaultTimerDuration: time.Duration(getEnvInt("DEFAULT_TIMER_SECONDS", 30)) * time.Second,
		DataDragonVersion:    getEnv("DDRAGON_VERSION", ""),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
