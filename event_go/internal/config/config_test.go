package config

import (
	"strings"
	"testing"
)

func TestValidateDevelopmentDefaults(t *testing.T) {
	cfg := Load()
	cfg.Environment = "development"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development defaults should be valid: %v", err)
	}
}

func TestValidateSecureEnvironmentRequiresSecrets(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "missing admin token",
			cfg: Config{
				Environment: "production", JWTExpireHours: 168, PasswordResetTTLMin: 30, RecoveryEmailTTLMin: 30,
				NotificationReminderHours: 24, NotificationScanIntervalSec: 60,
				JWTSecret: "12345678901234567890123456789012", CORSOrigin: "https://example.com",
			},
		},
		{
			name: "default jwt secret",
			cfg: Config{
				Environment: "production", JWTExpireHours: 168, PasswordResetTTLMin: 30, RecoveryEmailTTLMin: 30,
				NotificationReminderHours: 24, NotificationScanIntervalSec: 60,
				AdminToken: "admin-token", JWTSecret: DefaultJWTSecret, CORSOrigin: "https://example.com",
			},
		},
		{
			name: "wildcard cors",
			cfg: Config{
				Environment: "staging", JWTExpireHours: 168, PasswordResetTTLMin: 30, RecoveryEmailTTLMin: 30,
				NotificationReminderHours: 24, NotificationScanIntervalSec: 60,
				AdminToken: "admin-token", JWTSecret: "12345678901234567890123456789012", CORSOrigin: "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateProductionConfig(t *testing.T) {
	cfg := Config{
		Environment:                    "production",
		AdminToken:                     "admin-token",
		JWTSecret:                      "12345678901234567890123456789012",
		JWTExpireHours:                 168,
		CORSOrigin:                     "https://events.example.com",
		PublicBaseURL:                  "https://events.example.com",
		PasswordResetTTLMin:            30,
		RecoveryEmailTTLMin:            30,
		NotificationReminderHours:      24,
		NotificationScanIntervalSec:    60,
		OrganizationInvitationTTLHours: 72,
		RateLimitPerMinute:             600,
		SMTPHost:                       "smtp.example.com",
		SMTPPort:                       "587",
		SMTPUsername:                   "mailer",
		SMTPPassword:                   "secret",
		SMTPFrom:                       "Soulmark <no-reply@example.com>",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}

func TestLoadRejectsInvalidOrganizationInvitationTTL(t *testing.T) {
	t.Setenv("ORGANIZATION_INVITATION_TTL_HOURS", "0")
	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid organization invitation TTL must fail validation")
	}
}

func TestValidateRejectsUnknownEnvironment(t *testing.T) {
	cfg := Config{
		Environment: "prod", JWTExpireHours: 1, PasswordResetTTLMin: 30, RecoveryEmailTTLMin: 30,
		NotificationReminderHours: 24, NotificationScanIntervalSec: 60,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid APP_ENV error")
	}
}

func TestValidateProductionRequiresPasswordResetDelivery(t *testing.T) {
	cfg := Config{
		Environment: "production", AdminToken: "admin-token",
		JWTSecret: "12345678901234567890123456789012", JWTExpireHours: 168,
		CORSOrigin: "https://events.example.com", PublicBaseURL: "https://events.example.com",
		PasswordResetTTLMin: 30, RecoveryEmailTTLMin: 30,
		NotificationReminderHours: 24, NotificationScanIntervalSec: 60,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("production without SMTP password reset delivery must fail closed")
	}
}

func TestValidateProductionRejectsInsecureResetURLAndSMTPPort(t *testing.T) {
	cfg := Config{
		Environment: "production", AdminToken: "admin-token",
		JWTSecret: "12345678901234567890123456789012", JWTExpireHours: 168,
		CORSOrigin: "https://events.example.com", PublicBaseURL: "http://events.example.com",
		PasswordResetTTLMin: 30, RecoveryEmailTTLMin: 30,
		NotificationReminderHours: 24, NotificationScanIntervalSec: 60,
		SMTPHost: "smtp.example.com", SMTPPort: "invalid",
		SMTPUsername: "mailer", SMTPPassword: "secret", SMTPFrom: "no-reply@example.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("production HTTP reset URL must fail closed")
	}
	cfg.PublicBaseURL = "https://events.example.com"
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid SMTP port must fail closed")
	}
}

func TestLoadRejectsInvalidPasswordResetTTL(t *testing.T) {
	t.Setenv("PASSWORD_RESET_TTL_MINUTES", "not-a-number")
	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid PASSWORD_RESET_TTL_MINUTES must not fall back to the default")
	}
}

func TestLoadRejectsInvalidRecoveryEmailTTL(t *testing.T) {
	t.Setenv("RECOVERY_EMAIL_TTL_MINUTES", "not-a-number")
	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid RECOVERY_EMAIL_TTL_MINUTES must not fall back to the default")
	}
}

func TestLoadRejectsInvalidNotificationSchedule(t *testing.T) {
	t.Setenv("NOTIFICATION_REMINDER_HOURS", "0")
	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid notification reminder window must fail validation")
	}
	t.Setenv("NOTIFICATION_REMINDER_HOURS", "24")
	t.Setenv("NOTIFICATION_SCAN_INTERVAL_SECONDS", "not-a-number")
	cfg = Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid notification scan interval must fail validation")
	}
}

func TestLoadOrganizationAuthorizationFeatureFlag(t *testing.T) {
	t.Setenv("ORGANIZATION_AUTH_ENABLED", "false")
	if cfg := Load(); cfg.OrganizationAuthEnabled {
		t.Fatal("organization authorization feature flag was not disabled")
	}
	t.Setenv("ORGANIZATION_AUTH_ENABLED", "true")
	if cfg := Load(); !cfg.OrganizationAuthEnabled {
		t.Fatal("organization authorization feature flag was not enabled")
	}
	t.Setenv("ORGANIZATION_AUTH_ENABLED", "maybe")
	if cfg := Load(); cfg.Validate() == nil {
		t.Fatal("invalid organization authorization feature flag must fail validation")
	}
}

func TestLoadRejectsNegativeBackupIntervals(t *testing.T) {
	t.Setenv("BACKUP_INTERVAL_SECONDS", "-5")
	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative BACKUP_INTERVAL_SECONDS must fail validation")
	}
	t.Setenv("BACKUP_INTERVAL_SECONDS", "0")
	t.Setenv("BACKUP_DRILL_INTERVAL_SECONDS", "-1")
	cfg = Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative BACKUP_DRILL_INTERVAL_SECONDS must fail validation")
	}
}

func TestValidateProductionRequiresRateLimit(t *testing.T) {
	cfg := Config{
		Environment: "production", AdminToken: "admin-token",
		JWTSecret: "12345678901234567890123456789012", JWTExpireHours: 168,
		CORSOrigin: "https://events.example.com", PublicBaseURL: "https://events.example.com",
		PasswordResetTTLMin: 30, RecoveryEmailTTLMin: 30,
		NotificationReminderHours: 24, NotificationScanIntervalSec: 60,
		OrganizationInvitationTTLHours: 72,
		SMTPHost:                       "smtp.example.com", SMTPPort: "587",
		SMTPUsername: "mailer", SMTPPassword: "secret", SMTPFrom: "no-reply@example.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("production without a positive RATE_LIMIT_REQUESTS_PER_MINUTE must fail closed")
	}
	cfg.RateLimitPerMinute = 600
	if err := cfg.Validate(); err != nil {
		t.Fatalf("production with positive rate limit must be accepted: %v", err)
	}
}

func TestLoadRejectsInvalidRateLimit(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("RATE_LIMIT_REQUESTS_PER_MINUTE", "not-a-number")
	cfg := Load()
	// Production also requires other secrets, but an invalid rate limit must
	// not silently fall back to 0 (disabled) without failing closed.
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_REQUESTS_PER_MINUTE") {
		t.Fatalf("invalid RATE_LIMIT_REQUESTS_PER_MINUTE must fail closed, got: %v", err)
	}
}

func TestLoadRejectsNegativeRateLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS_PER_MINUTE", "-1")
	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative RATE_LIMIT_REQUESTS_PER_MINUTE must fail validation")
	}
}

func TestValidateAlertWebhookURL(t *testing.T) {
	valid := Load()
	valid.Environment = "development"
	valid.AlertWebhookURL = "https://hooks.example.com/alerts"
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid HTTPS alert webhook rejected: %v", err)
	}

	invalid := []struct {
		name string
		url  string
	}{
		{"http scheme", "http://hooks.example.com/alerts"},
		{"missing host", "https://"},
		{"not a url", "hooks.example.com/alerts"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Load()
			cfg.Environment = "development"
			cfg.AlertWebhookURL = tt.url
			if err := cfg.Validate(); err == nil {
				t.Fatalf("invalid alert webhook %q must fail validation", tt.url)
			}
		})
	}
}

func TestLoadReadsAlertWebhookURL(t *testing.T) {
	t.Setenv("ALERT_WEBHOOK_URL", "https://hooks.example.com/alerts")
	cfg := Load()
	if cfg.AlertWebhookURL != "https://hooks.example.com/alerts" {
		t.Fatalf("expected ALERT_WEBHOOK_URL to be loaded, got %q", cfg.AlertWebhookURL)
	}
}
