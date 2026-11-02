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

func TestProductionRequiresSMTPAndBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("production without SMTP must fail")
	}
	t.Setenv("SMTP_HOST", "smtp.example.org")
	t.Setenv("SMTP_FROM", "no-reply@example.org")
	t.Setenv("APP_BASE_URL", "https://tandem.example.org/")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppBaseURL != "https://tandem.example.org" {
		t.Errorf("trailing slash must be trimmed: %q", cfg.AppBaseURL)
	}
}

func TestRejectsUnknownEnvAndBadPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("unknown APP_ENV must fail")
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("non numeric PORT must fail")
	}
}
