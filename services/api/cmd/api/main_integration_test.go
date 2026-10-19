package main

import (
	"context"
	"os"
	"testing"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
)

func TestChooseMailer(t *testing.T) {
	if _, ok := chooseMailer(config.Config{Environment: "development"}).(auth.LogMailer); !ok {
		t.Fatal("development without SMTP should log codes")
	}
	if _, ok := chooseMailer(config.Config{Environment: "production"}).(auth.DisabledMailer); !ok {
		t.Fatal("production without SMTP must never log recovery codes")
	}
	smtp := config.Config{Environment: "production", SMTPHost: "smtp.example.com", SMTPFrom: "noreply@example.com"}
	if _, ok := chooseMailer(smtp).(*auth.SMTPMailer); !ok {
		t.Fatal("SMTP configuration should select the SMTP mailer")
	}
}

// Migrations must be safe to run on every boot: the second run is a no-op.
func TestMigrationsAreIdempotent(t *testing.T) {
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	for i := 0; i < 2; i++ {
		if err := db.RunMigrations(ctx, pool); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	var applied int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version = '014_dating_schema.sql'`).Scan(&applied); err != nil || applied != 1 {
		t.Fatalf("dating schema should be recorded once: %d %v", applied, err)
	}
	for _, table := range []string{"profiles", "preferences", "photos", "swipes", "matches", "conversations", "messages", "blocks", "reports", "notifications", "interests", "user_interests"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil || !exists {
			t.Errorf("table %s missing", table)
		}
	}
}
