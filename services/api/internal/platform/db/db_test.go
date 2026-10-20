package db_test

import (
	"context"
	"testing"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/testutil"
)

func TestMigrationsAreTrackedAndIdempotent(t *testing.T) {
	pool := testutil.NewTestDB(t) // applies every migration once
	ctx := context.Background()

	var first int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&first); err != nil || first == 0 {
		t.Fatalf("migrations not recorded: %d %v", first, err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("second run must be a no-op: %v", err)
	}
	var second int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&second)
	if first != second {
		t.Fatalf("migrations re-applied: %d -> %d", first, second)
	}
}

func TestSchemaConstraints(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	var a, b string
	_ = pool.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ('a@example.org', 'x') RETURNING id::text`).Scan(&a)
	_ = pool.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ('b@example.org', 'x') RETURNING id::text`).Scan(&b)

	mustFail := func(name, sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err == nil {
			t.Errorf("%s: constraint did not reject the row", name)
		}
	}
	mustFail("uppercase email", `INSERT INTO users (email, password_hash) VALUES ('UP@example.org', 'x')`)
	mustFail("duplicate email", `INSERT INTO users (email, password_hash) VALUES ('a@example.org', 'x')`)
	mustFail("self swipe", `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $1, 'like')`, a)
	mustFail("bad swipe action", `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'superlike')`, a, b)
	mustFail("self block", `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $1)`, a)

	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	mustFail("unordered match pair", `INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, hi, lo)
	if _, err := pool.Exec(ctx, `INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, lo, hi); err != nil {
		t.Fatal(err)
	}
	mustFail("duplicate match", `INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, lo, hi)

	// Deleting a user cascades to everything referencing them.
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, a); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM matches`).Scan(&n)
	if n != 0 {
		t.Fatalf("match survived user deletion")
	}
}
