package config

import "testing"

func TestLoadRejectsWeakJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "change-me-in-dev")
	if _, err := Load(); err == nil {
		t.Fatal("expected weak JWT secret to be rejected")
	}
}

func TestLoadParsesAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "https://app.example.com" || cfg.AllowedOrigins[1] != "https://admin.example.com" {
		t.Fatalf("unexpected allowed origins: %#v", cfg.AllowedOrigins)
	}
}

func TestLoadRateLimitDefaultsAndValidation(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AuthRatePerMinute != 20 || cfg.APIRatePerMinute != 600 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	t.Setenv("AUTH_RATE_PER_MINUTE", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid rate limit to be rejected")
	}
}

func TestLoadRequiresSMTPFromWithHost(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("SMTP_HOST", "smtp.example.org")
	if _, err := Load(); err == nil {
		t.Fatal("expected SMTP_FROM to be required")
	}
}
