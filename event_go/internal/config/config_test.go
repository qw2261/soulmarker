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
				Environment: "production",
				JWTSecret:   "12345678901234567890123456789012",
				CORSOrigin:  "https://example.com",
			},
		},
		{
			name: "default jwt secret",
			cfg: Config{
				Environment: "production",
				AdminToken:  "admin-token",
				JWTSecret:   DefaultJWTSecret,
				CORSOrigin:  "https://example.com",
			},
		},
		{
			name: "wildcard cors",
			cfg: Config{
				Environment: "staging",
				AdminToken:  "admin-token",
				JWTSecret:   "12345678901234567890123456789012",
				CORSOrigin:  "*",
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
		Environment: "production",
		AdminToken:  "admin-token",
		JWTSecret:   "12345678901234567890123456789012",
		CORSOrigin:  "https://events.example.com",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}

func TestValidateRejectsUnknownEnvironment(t *testing.T) {
	cfg := Config{Environment: "prod"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid APP_ENV error")
	}
}
