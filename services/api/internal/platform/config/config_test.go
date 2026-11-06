package config

import "testing"

func setBase(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "k3j9d8s7f6g5h4j3k2l1m0n9b8v7c6x5z4")
}

func TestLoadDefaults(t *testing.T) {
	setBase(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Port != "8080" || cfg.UploadDir == "" || cfg.SMTP.Enabled() {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsWeakSecrets(t *testing.T) {
	setBase(t)
	for _, secret := range []string{"", "short", "replace-with-a-random-secret-at-least-32-characters-long"} {
		t.Setenv("JWT_SECRET", secret)
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for secret %q", secret)
		}
	}
}

func TestLoadRequiresDatabaseAndMailFrom(t *testing.T) {
	setBase(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error without DATABASE_URL")
	}
	setBase(t)
	t.Setenv("SMTP_HOST", "smtp.local")
	if _, err := Load(); err == nil {
		t.Fatal("expected error without MAIL_FROM")
	}
}
