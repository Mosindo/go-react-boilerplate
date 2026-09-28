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

func setBase(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("APP_ENV", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("SMTP_PORT", "")
}

func TestLoadDefaults(t *testing.T) {
	setBase(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AppEnv != EnvDevelopment || cfg.Port != "8080" || cfg.SMTP.Port != 587 || cfg.SMTP.Configured() {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadProductionRequiresSMTP(t *testing.T) {
	setBase(t)
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("expected production without SMTP to be rejected")
	}
	t.Setenv("SMTP_HOST", "smtp.mail.test")
	t.Setenv("SMTP_FROM", "noreply@mail.test")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.IsProduction() || !cfg.SMTP.Configured() {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsUnknownEnvAndBadPort(t *testing.T) {
	setBase(t)
	t.Setenv("APP_ENV", "staging-ish")
	if _, err := Load(); err == nil {
		t.Fatal("expected unknown APP_ENV to be rejected")
	}
	t.Setenv("APP_ENV", "test")
	t.Setenv("SMTP_PORT", "notaport")
	if _, err := Load(); err == nil {
		t.Fatal("expected bad SMTP_PORT to be rejected")
	}
}
