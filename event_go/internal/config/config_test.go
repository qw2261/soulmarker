package config

import "testing"

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
		Environment:                 "production",
		AdminToken:                  "admin-token",
		JWTSecret:                   "12345678901234567890123456789012",
		JWTExpireHours:              168,
		CORSOrigin:                  "https://events.example.com",
		PublicBaseURL:               "https://events.example.com",
		PasswordResetTTLMin:         30,
		RecoveryEmailTTLMin:         30,
		NotificationReminderHours:   24,
		NotificationScanIntervalSec: 60,
		SMTPHost:                    "smtp.example.com",
		SMTPPort:                    "587",
		SMTPUsername:                "mailer",
		SMTPPassword:                "secret",
		SMTPFrom:                    "Soulmark <no-reply@example.com>",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
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
