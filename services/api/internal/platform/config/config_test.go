package config

import "testing"

func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for _, k := range []string{"PORT", "APP_ENV", "DATABASE_URL", "JWT_SECRET", "SMTP_HOST", "SMTP_FROM", "ALLOWED_ORIGINS"} {
		t.Setenv(k, "")
	}
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

const goodSecret = "k3v9Xq2LmZp8Rt5Wy7Ub4NcAe1HdGfJs"

func TestLoadDefaults(t *testing.T) {
	setEnv(t, map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret})
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.Env != "development" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsBadSecrets(t *testing.T) {
	for _, secret := range []string{"", "short", "replace-with-a-random-secret-at-least-32-characters-long"} {
		setEnv(t, map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": secret})
		if _, err := Load(); err == nil {
			t.Fatalf("secret %q should be rejected", secret)
		}
	}
}

func TestLoadRequiresDatabaseAndSMTPInProduction(t *testing.T) {
	setEnv(t, map[string]string{"JWT_SECRET": goodSecret})
	if _, err := Load(); err == nil {
		t.Fatal("DATABASE_URL must be required")
	}
	setEnv(t, map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret, "APP_ENV": "production"})
	if _, err := Load(); err == nil {
		t.Fatal("production must require SMTP_HOST")
	}
	setEnv(t, map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret, "APP_ENV": "production", "SMTP_HOST": "smtp.local", "SMTP_FROM": "a@b.c"})
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsUnknownEnv(t *testing.T) {
	setEnv(t, map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret, "APP_ENV": "staging"})
	if _, err := Load(); err == nil {
		t.Fatal("unknown APP_ENV must fail")
	}
}
