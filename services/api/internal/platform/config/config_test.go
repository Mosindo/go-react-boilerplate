package config

import "testing"

const goodSecret = "0123456789abcdef0123456789abcdef"

func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", goodSecret)
	t.Setenv("APP_ENV", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("ALLOWED_ORIGINS", "")
}

func TestLoadRejectsWeakJWTSecret(t *testing.T) {
	baseEnv(t)
	for _, secret := range []string{"change-me-in-dev", "short", "replace-with-a-random-secret-at-least-32-characters-long"} {
		t.Setenv("JWT_SECRET", secret)
		if _, err := Load(); err == nil {
			t.Fatalf("expected %q to be rejected", secret)
		}
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	baseEnv(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("DATABASE_URL is required")
	}
}

func TestLoadParsesAllowedOrigins(t *testing.T) {
	baseEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "https://app.example.com" || cfg.AllowedOrigins[1] != "https://admin.example.com" {
		t.Fatalf("unexpected allowed origins: %#v", cfg.AllowedOrigins)
	}
	if cfg.Port != "8080" || cfg.Env != "development" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestProductionRequiresEmailDelivery(t *testing.T) {
	baseEnv(t)
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("production without SMTP must fail: recovery codes would only be logged")
	}
	t.Setenv("SMTP_HOST", "smtp.example.org")
	if _, err := Load(); err == nil {
		t.Fatal("SMTP_FROM is required with SMTP_HOST")
	}
	t.Setenv("SMTP_FROM", "no-reply@example.org")
	if _, err := Load(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}
