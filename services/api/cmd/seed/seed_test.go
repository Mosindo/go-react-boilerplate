package main

import (
	"context"
	"image/jpeg"
	"testing"
	"time"

	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/testutil"
)

// The test seeds a small, remote cohort under its own domain so it cannot interfere with
// other packages' tests running against the same database.
func TestSeedIdempotentAndPurge(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const domain = "seedtest.alba.invalid"
	opts := Options{Count: 5, Domain: domain, CenterLat: -50, CenterLng: -170, Password: "SeedTestPass!1", Now: time.Now()}
	t.Cleanup(func() { _, _ = Purge(context.Background(), pool, store, domain) })
	_, _ = Purge(ctx, pool, store, domain)

	keepID, _ := testutil.User(t, pool, "keepme", "1990-01-01")

	created, skipped, err := Seed(ctx, pool, store, opts)
	if err != nil || created != 5 || skipped != 0 {
		t.Fatalf("seed: %d %d %v", created, skipped, err)
	}
	created, skipped, err = Seed(ctx, pool, store, opts)
	if err != nil || created != 0 || skipped != 5 {
		t.Fatalf("second run must be a no-op: %d %d %v", created, skipped, err)
	}

	var users, complete, photos int
	if err := pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM profiles p WHERE p.user_id=u.id AND p.latitude IS NOT NULL)
		                        AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id=u.id)),
		       (SELECT count(*) FROM photos ph JOIN users x ON x.id=ph.user_id WHERE x.email LIKE '%@seedtest.alba.invalid')
		FROM users u WHERE u.email LIKE 'demo+%@seedtest.alba.invalid'`).Scan(&users, &complete, &photos); err != nil {
		t.Fatal(err)
	}
	if users != 5 || complete != 5 || photos < 10 {
		t.Fatalf("users=%d complete=%d photos=%d", users, complete, photos)
	}

	var key string
	var age int
	if err := pool.QueryRow(ctx, `
		SELECT ph.storage_key, date_part('year', age(u.birth_date))::int FROM photos ph JOIN users u ON u.id=ph.user_id
		WHERE u.email = $1 AND ph.position = 0`, "demo+1@"+domain).Scan(&key, &age); err != nil {
		t.Fatal(err)
	}
	rc, err := store.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jpeg.Decode(rc); err != nil {
		t.Fatalf("placeholder portrait is not a JPEG: %v", err)
	}
	rc.Close()
	if age < 19 || age > 45 {
		t.Fatalf("age %d out of range", age)
	}

	n, err := Purge(ctx, pool, store, domain)
	if err != nil || n != 5 {
		t.Fatalf("purge: %d %v", n, err)
	}
	if _, err := store.Open(ctx, key); err == nil {
		t.Fatal("portrait file survived purge")
	}
	var remaining, kept int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email LIKE 'demo+%@seedtest.alba.invalid'`).Scan(&remaining)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE id=$1`, keepID).Scan(&kept)
	if remaining != 0 || kept != 1 {
		t.Fatalf("remaining=%d kept=%d", remaining, kept)
	}
}

func TestPurgeRejectsBadDomain(t *testing.T) {
	if _, err := Purge(context.Background(), nil, nil, "x' OR 1=1 --"); err == nil {
		t.Fatal("bad domain accepted")
	}
}
