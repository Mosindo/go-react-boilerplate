package db

import (
	"context"
	"os"
	"testing"
)

func TestRunMigrationsIsIdempotent(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	ctx := context.Background()
	pool, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer pool.Close()

	for i := 0; i < 2; i++ {
		if err := RunMigrations(ctx, pool); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	var applied, interests int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM interests`).Scan(&interests); err != nil {
		t.Fatal(err)
	}
	if applied < 15 || interests < 40 {
		t.Fatalf("unexpected state: %d migrations, %d interests", applied, interests)
	}
}
