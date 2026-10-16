package config

import "testing"

const strongSecret = "k3J9xQ2vLm8Zr5TnB7wYc1HdPf6GsAeU"

func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://app:app@postgres:5432/app?sslmode=disable")
	t.Setenv("JWT_SECRET", strongSecret)
	t.Setenv("APP_ENV", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_FROM", "")
}

func TestLoadDefaults(t *testing.T) {
	baseEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Port != "8080" || cfg.Env != "development" || cfg.UploadDir == "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsWeakSecrets(t *testing.T) {
	baseEnv(t)
	for _, secret := range []string{"", "short", "replace-with-a-random-secret-at-least-32-characters-long"} {
		t.Setenv("JWT_SECRET", secret)
		if _, err := Load(); err == nil {
			t.Errorf("secret %q should be rejected", secret)
		}
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	baseEnv(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is missing")
	}
}

func TestProductionRequiresSMTP(t *testing.T) {
	baseEnv(t)
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("production without SMTP must fail")
	}
	t.Setenv("SMTP_HOST", "smtp.mail.test")
	t.Setenv("SMTP_FROM", "no-reply@mail.test")
	if _, err := Load(); err != nil {
		t.Fatalf("production with SMTP should load: %v", err)
	}
}

func TestInvalidEnvRejected(t *testing.T) {
	baseEnv(t)
	t.Setenv("APP_ENV", "staging-ish")
	if _, err := Load(); err == nil {
		t.Fatal("unknown APP_ENV must be rejected")
	}
}
