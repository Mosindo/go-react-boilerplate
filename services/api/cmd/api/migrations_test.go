package main

import (
	"context"
	"sync"
	"testing"

	"example.com/api/internal/platform/db"
)

func TestMigrationsAreIdempotentAndConcurrencySafe(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	var before int
	if err := e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&before); err != nil || before < 16 {
		t.Fatalf("expected recorded migrations, got %d (%v)", before, err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- db.RunMigrations(ctx, e.pool)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent re-run failed: %v", err)
		}
	}
	var after int
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&after)
	if after != before {
		t.Fatalf("re-running migrations changed the ledger: %d -> %d", before, after)
	}
}

func TestSchemaEnforcesCoreInvariants(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()
	ctx := context.Background()

	// Canonical pair ordering: user_a must sort before user_b, and a pair is unique.
	lo, hi := a.ID, b.ID
	if lo > hi {
		lo, hi = hi, lo
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, hi, lo); err == nil {
		t.Fatal("matches must enforce user_a < user_b")
	}
	// Swipes cannot target oneself or repeat.
	if _, err := e.pool.Exec(ctx, `INSERT INTO swipes (user_id, target_id, action) VALUES ($1, $1, 'like')`, a.ID); err == nil {
		t.Fatal("self swipe accepted by the database")
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO swipes (user_id, target_id, action) VALUES ($1, $2, 'like')`, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO swipes (user_id, target_id, action) VALUES ($1, $2, 'pass')`, a.ID, b.ID); err == nil {
		t.Fatal("duplicate swipe accepted by the database")
	}
	// Underage preferences and invalid enumerations are rejected at the DB level too.
	if _, err := e.pool.Exec(ctx, `UPDATE preferences SET min_age = 16 WHERE user_id = $1`, a.ID); err == nil {
		t.Fatal("minimum age below 18 accepted")
	}
	if _, err := e.pool.Exec(ctx, `UPDATE profiles SET gender = 'x' WHERE user_id = $1`, a.ID); err == nil {
		t.Fatal("invalid gender accepted")
	}
	// Seventh photo position is impossible.
	if _, err := e.pool.Exec(ctx, `UPDATE photos SET position = 6 WHERE user_id = $1`, a.ID); err == nil {
		t.Fatal("photo position 6 accepted")
	}
}
