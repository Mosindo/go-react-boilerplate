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

func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
}

func TestLoadDefaults(t *testing.T) {
	baseEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.AppEnv != "development" || cfg.UploadDir == "" || cfg.IsProduction() || cfg.SMTP.Enabled() {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadProductionRequiresMailAndValidEnv(t *testing.T) {
	baseEnv(t)
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("production without SMTP must be refused: recovery emails could not be delivered")
	}
	t.Setenv("SMTP_HOST", "smtp.example.com")
	if _, err := Load(); err == nil {
		t.Fatal("SMTP_FROM is required with SMTP_HOST")
	}
	t.Setenv("SMTP_FROM", "noreply@example.com")
	if cfg, err := Load(); err != nil || !cfg.IsProduction() || !cfg.SMTP.Enabled() {
		t.Fatalf("valid production config rejected: %v", err)
	}

	t.Setenv("APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("unknown APP_ENV must be refused")
	}
}

func TestLoadRequiresDatabaseAndResetPlaceholder(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("DATABASE_URL is required")
	}
	baseEnv(t)
	t.Setenv("PASSWORD_RESET_URL", "https://app.example.com/reset")
	if _, err := Load(); err == nil {
		t.Fatal("PASSWORD_RESET_URL must contain {token}")
	}
}
