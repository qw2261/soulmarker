package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const DefaultJWTSecret = "event-go-dev-secret-change-in-production"

type Config struct {
	Environment         string
	AdminToken          string
	CORSOrigin          string
	LogFormat           string
	LogLevel            string
	DatabasePath        string
	Port                string
	Version             string
	CancelDeadlineHours int
	JWTSecret           string
	JWTExpireHours      int
}

func Load() *Config {
	return &Config{
		Environment:         getEnv("APP_ENV", "development"),
		AdminToken:          getEnv("ADMIN_TOKEN", ""),
		CORSOrigin:          getEnv("CORS_ORIGIN", "*"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DatabasePath:        getEnv("DATABASE_PATH", "data/event_go.db"),
		Port:                getEnv("PORT", "8080"),
		Version:             getEnv("VERSION", "dev"),
		CancelDeadlineHours: getEnvInt("CANCEL_DEADLINE_HOURS", 24),
		JWTSecret:           getEnv("JWT_SECRET", DefaultJWTSecret),
		JWTExpireHours:      getEnvInt("JWT_EXPIRE_HOURS", 168),
	}
}

func (c *Config) Validate() error {
	env := strings.ToLower(strings.TrimSpace(c.Environment))
	switch env {
	case "development", "test":
		return nil
	case "staging", "production":
		if strings.TrimSpace(c.AdminToken) == "" {
			return fmt.Errorf("%s 环境必须设置 ADMIN_TOKEN", env)
		}
		if c.JWTSecret == DefaultJWTSecret || len(c.JWTSecret) < 32 {
			return fmt.Errorf("%s 环境必须设置至少 32 字节的非默认 JWT_SECRET", env)
		}
		if strings.TrimSpace(c.CORSOrigin) == "" || c.CORSOrigin == "*" {
			return fmt.Errorf("%s 环境必须设置明确的 CORS_ORIGIN", env)
		}
		return nil
	default:
		return fmt.Errorf("无效的 APP_ENV: %s", c.Environment)
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
