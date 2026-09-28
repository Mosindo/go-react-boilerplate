package main

import (
	"context"
	"strings"
	"testing"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/imaging"
	"example.com/api/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestPortraitIsAValidPhoto(t *testing.T) {
	data, w, h, err := portrait(3)
	if err != nil || w != 480 || h != 600 {
		t.Fatalf("portrait: %d %d %v", w, h, err)
	}
	res, err := imaging.Process(data)
	if err != nil {
		t.Fatalf("generated portrait must pass the upload pipeline: %v", err)
	}
	if res.Width != 480 || res.Height != 600 {
		t.Fatalf("unexpected size %dx%d", res.Width, res.Height)
	}
	other, _, _, _ := portrait(4)
	if string(other) == string(data) {
		t.Error("portraits must differ per seed")
	}
}

func TestSeedIsIdempotentAndLoginWorks(t *testing.T) {
	pool := testutil.NewDB(t)
	ctx := context.Background()
	hash, err := auth.HashPassword(DemoPassword, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	counts := func() [5]int {
		var c [5]int
		err := pool.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM users), (SELECT count(*) FROM photos), (SELECT count(*) FROM matches),
			(SELECT count(*) FROM match_messages), (SELECT count(*) FROM swipes)`).Scan(&c[0], &c[1], &c[2], &c[3], &c[4])
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	for i := 0; i < 2; i++ {
		if err := seed(ctx, pool, hash, 48.8566, 2.3522, "Demo City", 15); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	c := counts()
	if c[0] != userCount || c[1] < userCount || c[2] != 5 || c[3] != 6 {
		t.Fatalf("unexpected counts after two runs: %v", c)
	}
	var bad int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email NOT LIKE '%@demo.invalid'`).Scan(&bad)
	if bad != 0 {
		t.Errorf("all seeded emails must end in @demo.invalid")
	}
	// coordinates are stored at API precision
	var unrounded int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM profiles WHERE latitude <> round(latitude::numeric, 2) OR longitude <> round(longitude::numeric, 2)`).Scan(&unrounded)
	if unrounded != 0 {
		t.Errorf("%d profiles with unrounded coordinates", unrounded)
	}
	// every profile is complete and adult
	var incomplete int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM profiles p WHERE latitude IS NULL OR NOT EXISTS (SELECT 1 FROM photos WHERE user_id=p.user_id) OR birth_date > CURRENT_DATE - INTERVAL '18 years'`).Scan(&incomplete)
	if incomplete != 0 {
		t.Errorf("%d incomplete/underage demo profiles", incomplete)
	}
	// the shared password works through the real auth service
	svc := auth.NewService(auth.NewPGRepository(pool), []byte(strings.Repeat("k", 40)), nil, bcrypt.MinCost)
	if _, _, err := svc.Login(ctx, "demo01@demo.invalid", DemoPassword, "", ""); err != nil {
		t.Fatalf("demo login: %v", err)
	}
}
