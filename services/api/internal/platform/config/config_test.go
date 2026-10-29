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

func TestLoadRequiresSMTPInProduction(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://app:app@localhost:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("production without SMTP must be rejected: recovery emails could never be delivered")
	}

	t.Setenv("SMTP_HOST", "smtp.example.org")
	if _, err := Load(); err == nil {
		t.Fatal("SMTP_FROM is required when SMTP_HOST is set")
	}
	t.Setenv("SMTP_FROM", "no-reply@example.org")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	if !cfg.IsProduction() || !cfg.SMTP.Enabled() {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsMissingDatabaseAndInvalidPort(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("DATABASE_URL is required")
	}
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("PORT", "99999")
	if _, err := Load(); err == nil {
		t.Fatal("invalid PORT must be rejected")
	}
}
