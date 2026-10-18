package db_test

import (
	"context"
	"testing"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/db/dbtest"
)

func TestRunMigrationsIsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t) // already migrated once

	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("second run: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("schema_migrations count=%d err=%v", n, err)
	}
	for _, table := range []string{"profiles", "preferences", "interests", "user_interests", "photos", "swipes",
		"matches", "conversations", "messages", "blocks", "reports", "notifications", "password_resets"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil || !exists {
			t.Fatalf("table %s missing (err=%v)", table, err)
		}
	}
	for _, gone := range []string{"organizations", "subscriptions", "posts", "comments", "votes", "files"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, gone).Scan(&exists); err != nil || exists {
			t.Fatalf("legacy table %s should have been dropped (err=%v)", gone, err)
		}
	}
}

func TestSchemaEnforcesMatchOrderingAndSelfSwipe(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	var a, b string
	for _, dst := range []*string{&a, &b} {
		if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES (gen_random_uuid()::text || '@t.io', 'x') RETURNING id`).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $1, 'like')`, a); err == nil {
		t.Fatal("self swipe must be rejected")
	}
	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, hi, lo); err == nil {
		t.Fatal("unordered match pair must be rejected")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, lo, hi); err != nil {
		t.Fatalf("ordered pair: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, lo, hi); err == nil {
		t.Fatal("duplicate match must be rejected")
	}
}
