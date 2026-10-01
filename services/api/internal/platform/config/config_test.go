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

func TestLoadProductionRequiresSMTP(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("expected production without SMTP to be rejected")
	}
	t.Setenv("SMTP_HOST", "smtp.example.org")
	t.Setenv("SMTP_FROM", "no-reply@example.org")
	if _, err := Load(); err != nil {
		t.Fatalf("production with SMTP should load: %v", err)
	}
	t.Setenv("RATE_LIMIT", "off")
	if _, err := Load(); err == nil {
		t.Fatal("expected RATE_LIMIT=off to be rejected in production")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Port != "8080" || cfg.Env != "development" || !cfg.RateLimit {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}
