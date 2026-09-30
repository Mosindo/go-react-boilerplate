package config

import "testing"

const strongSecret = "0123456789abcdef0123456789abcdef"

func TestLoadRejectsWeakJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	for _, secret := range []string{"change-me-in-dev", "replace-with-a-random-secret-at-least-32-characters-long", "short"} {
		t.Setenv("JWT_SECRET", secret)
		if _, err := Load(); err == nil {
			t.Fatalf("expected weak JWT secret %q to be rejected", secret)
		}
	}
}

func TestLoadDefaultsAndLists(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", strongSecret)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com ")
	t.Setenv("APP_ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AppEnv != EnvDevelopment || cfg.Port != "8080" || cfg.SMTP.Port != 587 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "https://admin.example.com" {
		t.Fatalf("unexpected allowed origins: %#v", cfg.AllowedOrigins)
	}
}

func TestLoadValidatesEnvAndSMTP(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", strongSecret)
	t.Setenv("APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("expected unknown APP_ENV to be rejected")
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("SMTP_HOST", "smtp.example.org")
	if _, err := Load(); err == nil {
		t.Fatal("expected SMTP_FROM to be required with SMTP_HOST")
	}
}
