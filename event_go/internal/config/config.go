package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/buildinfo"
)

const DefaultJWTSecret = "event-go-dev-secret-change-in-production"

type Config struct {
	Environment                    string
	AdminToken                     string
	CORSOrigin                     string
	LogFormat                      string
	LogLevel                       string
	DatabasePath                   string
	Port                           string
	Version                        string
	CancelDeadlineHours            int
	JWTSecret                      string
	JWTExpireHours                 int
	PublicBaseURL                  string
	PasswordResetTTLMin            int
	RecoveryEmailTTLMin            int
	NotificationReminderHours      int
	NotificationScanIntervalSec    int
	OrganizationAuthEnabled        bool
	OrganizationInvitationTTLHours int
	SMTPHost                       string
	SMTPPort                       string
	SMTPUsername                   string
	SMTPPassword                   string
	SMTPFrom                       string
	BackupDir                      string
	BackupIntervalSec              int
	BackupRetain                   int
	BackupDrillIntervalSec         int
	RateLimitPerMinute             int
	AlertWebhookURL                string
	organizationAuthFlagInvalid    bool
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	organizationAuthEnabled, organizationAuthFlagInvalid := getEnvBoolStrict("ORGANIZATION_AUTH_ENABLED", true)
	return &Config{
		Environment:                    getEnv("APP_ENV", "development"),
		AdminToken:                     getEnv("ADMIN_TOKEN", ""),
		CORSOrigin:                     getEnv("CORS_ORIGIN", "*"),
		LogFormat:                      getEnv("LOG_FORMAT", "json"),
		LogLevel:                       getEnv("LOG_LEVEL", "info"),
		DatabasePath:                   getEnv("DATABASE_PATH", "data/event_go.db"),
		Port:                           port,
		Version:                        getEnv("VERSION", buildinfo.DefaultVersion()),
		CancelDeadlineHours:            getEnvInt("CANCEL_DEADLINE_HOURS", 24),
		JWTSecret:                      getEnv("JWT_SECRET", DefaultJWTSecret),
		JWTExpireHours:                 getEnvInt("JWT_EXPIRE_HOURS", 168),
		PublicBaseURL:                  getEnv("PUBLIC_BASE_URL", "http://localhost:"+port),
		PasswordResetTTLMin:            getEnvIntStrict("PASSWORD_RESET_TTL_MINUTES", 30),
		RecoveryEmailTTLMin:            getEnvIntStrict("RECOVERY_EMAIL_TTL_MINUTES", 30),
		NotificationReminderHours:      getEnvIntStrict("NOTIFICATION_REMINDER_HOURS", 24),
		NotificationScanIntervalSec:    getEnvIntStrict("NOTIFICATION_SCAN_INTERVAL_SECONDS", 60),
		OrganizationAuthEnabled:        organizationAuthEnabled,
		OrganizationInvitationTTLHours: getEnvIntStrict("ORGANIZATION_INVITATION_TTL_HOURS", 72),
		SMTPHost:                       getEnv("SMTP_HOST", ""),
		SMTPPort:                       getEnv("SMTP_PORT", "587"),
		SMTPUsername:                   getEnv("SMTP_USERNAME", ""),
		SMTPPassword:                   getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                       getEnv("SMTP_FROM", ""),
		BackupDir:                      getEnv("BACKUP_DIR", "data/backups"),
		BackupIntervalSec:              getEnvInt("BACKUP_INTERVAL_SECONDS", 0),
		BackupRetain:                   getEnvIntStrict("BACKUP_RETAIN", 7),
		BackupDrillIntervalSec:         getEnvInt("BACKUP_DRILL_INTERVAL_SECONDS", 0),
		RateLimitPerMinute:             getEnvIntStrict("RATE_LIMIT_REQUESTS_PER_MINUTE", 0),
		AlertWebhookURL:                getEnv("ALERT_WEBHOOK_URL", ""),
		organizationAuthFlagInvalid:    organizationAuthFlagInvalid,
	}
}

func (c *Config) Validate() error {
	if c.organizationAuthFlagInvalid {
		return fmt.Errorf("ORGANIZATION_AUTH_ENABLED 必须是 true 或 false")
	}
	if c.JWTExpireHours <= 0 {
		return fmt.Errorf("JWT_EXPIRE_HOURS 必须大于 0")
	}
	if c.PasswordResetTTLMin <= 0 || c.PasswordResetTTLMin > 1440 {
		return fmt.Errorf("PASSWORD_RESET_TTL_MINUTES 必须在 1 到 1440 之间")
	}
	if c.RecoveryEmailTTLMin <= 0 || c.RecoveryEmailTTLMin > 1440 {
		return fmt.Errorf("RECOVERY_EMAIL_TTL_MINUTES 必须在 1 到 1440 之间")
	}
	if c.NotificationReminderHours <= 0 || c.NotificationReminderHours > 168 {
		return fmt.Errorf("NOTIFICATION_REMINDER_HOURS 必须在 1 到 168 之间")
	}
	if c.NotificationScanIntervalSec <= 0 || c.NotificationScanIntervalSec > 3600 {
		return fmt.Errorf("NOTIFICATION_SCAN_INTERVAL_SECONDS 必须在 1 到 3600 之间")
	}
	if c.OrganizationInvitationTTLHours <= 0 || c.OrganizationInvitationTTLHours > 168 {
		return fmt.Errorf("ORGANIZATION_INVITATION_TTL_HOURS 必须在 1 到 168 之间")
	}
	if c.BackupIntervalSec < 0 {
		return fmt.Errorf("BACKUP_INTERVAL_SECONDS 不能为负数")
	}
	if c.BackupDrillIntervalSec < 0 {
		return fmt.Errorf("BACKUP_DRILL_INTERVAL_SECONDS 不能为负数")
	}
	if c.RateLimitPerMinute < 0 {
		return fmt.Errorf("RATE_LIMIT_REQUESTS_PER_MINUTE 不能为负数")
	}
	if c.AlertWebhookURL != "" {
		alertURL, err := url.Parse(c.AlertWebhookURL)
		if err != nil || alertURL.Scheme != "https" || alertURL.Host == "" {
			return fmt.Errorf("ALERT_WEBHOOK_URL 必须是有效的 HTTPS URL")
		}
	}
	env := strings.ToLower(strings.TrimSpace(c.Environment))
	switch env {
	case "development", "test":
		return nil
	case "staging", "production":
		if c.RateLimitPerMinute <= 0 {
			return fmt.Errorf("%s 环境必须设置大于 0 的 RATE_LIMIT_REQUESTS_PER_MINUTE（限流不能关闭）", env)
		}
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

func getEnvBoolStrict(key string, defaultValue bool) (bool, bool) {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "":
		return defaultValue, false
	case "1", "true", "yes", "on":
		return true, false
	case "0", "false", "no", "off":
		return false, false
	default:
		return defaultValue, true
	}
}
