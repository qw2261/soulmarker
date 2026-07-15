package config

import (
	"fmt"
	"net/mail"
	"net/url"
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
	PublicBaseURL       string
	PasswordResetTTLMin int
	SMTPHost            string
	SMTPPort            string
	SMTPUsername        string
	SMTPPassword        string
	SMTPFrom            string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	return &Config{
		Environment:         getEnv("APP_ENV", "development"),
		AdminToken:          getEnv("ADMIN_TOKEN", ""),
		CORSOrigin:          getEnv("CORS_ORIGIN", "*"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DatabasePath:        getEnv("DATABASE_PATH", "data/event_go.db"),
		Port:                port,
		Version:             getEnv("VERSION", "dev"),
		CancelDeadlineHours: getEnvInt("CANCEL_DEADLINE_HOURS", 24),
		JWTSecret:           getEnv("JWT_SECRET", DefaultJWTSecret),
		JWTExpireHours:      getEnvInt("JWT_EXPIRE_HOURS", 168),
		PublicBaseURL:       getEnv("PUBLIC_BASE_URL", "http://localhost:"+port),
		PasswordResetTTLMin: getEnvIntStrict("PASSWORD_RESET_TTL_MINUTES", 30),
		SMTPHost:            getEnv("SMTP_HOST", ""),
		SMTPPort:            getEnv("SMTP_PORT", "587"),
		SMTPUsername:        getEnv("SMTP_USERNAME", ""),
		SMTPPassword:        getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:            getEnv("SMTP_FROM", ""),
	}
}

func (c *Config) Validate() error {
	if c.JWTExpireHours <= 0 {
		return fmt.Errorf("JWT_EXPIRE_HOURS 必须大于 0")
	}
	if c.PasswordResetTTLMin <= 0 || c.PasswordResetTTLMin > 1440 {
		return fmt.Errorf("PASSWORD_RESET_TTL_MINUTES 必须在 1 到 1440 之间")
	}
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
		baseURL, err := url.Parse(c.PublicBaseURL)
		if err != nil || baseURL.Scheme != "https" || baseURL.Host == "" {
			return fmt.Errorf("%s 环境必须设置有效的 HTTPS PUBLIC_BASE_URL", env)
		}
		if strings.TrimSpace(c.SMTPHost) == "" || strings.TrimSpace(c.SMTPPort) == "" ||
			strings.TrimSpace(c.SMTPUsername) == "" || strings.TrimSpace(c.SMTPPassword) == "" ||
			strings.TrimSpace(c.SMTPFrom) == "" {
			return fmt.Errorf("%s 环境必须完整设置 SMTP_HOST、SMTP_PORT、SMTP_USERNAME、SMTP_PASSWORD、SMTP_FROM", env)
		}
		smtpPort, err := strconv.Atoi(c.SMTPPort)
		if err != nil || smtpPort < 1 || smtpPort > 65535 {
			return fmt.Errorf("SMTP_PORT 必须是 1 到 65535 的端口")
		}
		if _, err := mail.ParseAddress(c.SMTPFrom); err != nil {
			return fmt.Errorf("SMTP_FROM 不是有效邮箱地址")
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

func getEnvIntStrict(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
