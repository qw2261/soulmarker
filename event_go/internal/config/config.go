package config

import (
	"os"
	"strconv"
)

type Config struct {
	AdminToken          string
	CORSOrigin          string
	LogFormat           string
	LogLevel            string
	DatabasePath        string
	Port                string
	Version             string
	CancelDeadlineHours int
}

func Load() *Config {
	return &Config{
		AdminToken:          getEnv("ADMIN_TOKEN", ""),
		CORSOrigin:          getEnv("CORS_ORIGIN", "*"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DatabasePath:        getEnv("DATABASE_PATH", "data/event_go.db"),
		Port:                getEnv("PORT", "8080"),
		Version:             getEnv("VERSION", "dev"),
		CancelDeadlineHours: getEnvInt("CANCEL_DEADLINE_HOURS", 24),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
