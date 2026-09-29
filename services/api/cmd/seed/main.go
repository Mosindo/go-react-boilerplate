// Command seed fills a development database with clearly marked demo users (email domain
// seed.alba.invalid), generated placeholder portraits and realistic profile data.
//
//	go run ./cmd/seed -yes            # create (idempotent)
//	go run ./cmd/seed -yes -purge     # delete exactly the seeded users and their files
//
// It never runs when APP_ENV=production and never without -yes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
)

const (
	defaultPassword = "DemoPass!2026"
	// SeedDomain marks every seeded account; .invalid can never receive real mail.
	SeedDomain = "seed.alba.invalid"
	// Demo city: Lyon, France.
	demoLat = 45.76
	demoLng = 4.84
)

func main() {
	yes := flag.Bool("yes", false, "confirm that demo data may be written to DATABASE_URL")
	purge := flag.Bool("purge", false, "delete the seeded demo users and their photo files instead of creating them")
	count := flag.Int("n", 24, "number of demo users to create")
	flag.Parse()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		log.Fatal("seed: DATABASE_URL is required")
	}
	if !*yes {
		fmt.Fprintf(os.Stderr, "seed: this writes demo users (%s) into the database from DATABASE_URL.\nRe-run with -yes to confirm.\n", SeedDomain)
		os.Exit(2)
	}
	if *count < 1 || *count > 200 {
		log.Fatal("seed: -n must be between 1 and 200")
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = defaultPassword
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, dsn, 10*time.Second, time.Second)
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("seed: migrations: %v", err)
	}
	store, err := storage.NewLocal(uploadDir)
	if err != nil {
		log.Fatalf("seed: %v", err)
	}

	opts := Options{Count: *count, Domain: SeedDomain, CenterLat: demoLat, CenterLng: demoLng, Password: password, Now: time.Now()}
	if *purge {
		n, err := Purge(ctx, pool, store, opts.Domain)
		if err != nil {
			log.Fatalf("seed: purge: %v", err)
		}
		fmt.Printf("purged %d demo users\n", n)
		return
	}
	created, skipped, err := Seed(ctx, pool, store, opts)
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	fmt.Printf("created %d demo users (%d already existed)\n", created, skipped)
	fmt.Printf("login with demo+1@%s ... demo+%d@%s\npassword: %s\n", SeedDomain, *count, SeedDomain, password)
}
