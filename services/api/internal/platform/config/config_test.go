package config

import "testing"

func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
}

func TestLoadRejectsWeakJWTSecret(t *testing.T) {
	baseEnv(t)
	for _, weak := range []string{"", "short", "change-me-in-dev-0123456789abcdef0123456789"} {
		t.Setenv("JWT_SECRET", weak)
		if _, err := Load(); err == nil {
			t.Errorf("secret %q should be rejected", weak)
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

func TestLoadDefaultsAndParsing(t *testing.T) {
	baseEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com ")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Port != "8080" || cfg.AppEnv != "development" || cfg.UploadDir == "" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "https://admin.example.com" {
		t.Errorf("unexpected allowed origins: %#v", cfg.AllowedOrigins)
	}
}

func TestProductionRequiresSMTP(t *testing.T) {
	baseEnv(t)
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("production without SMTP must be refused: recovery mails could not be delivered")
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
