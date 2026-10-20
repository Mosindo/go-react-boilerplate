// Command seed fills a development database with clearly fake demo users.
//
// Demo accounts use the reserved ".invalid" TLD (demo-*@demo.invalid) and one
// shared password, so they can never be confused with real users or receive
// mail. The command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed          # create demo users (idempotent)
//	go run ./cmd/seed -reset   # delete every demo user and their photos
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	demoDomain   = "demo.invalid"
	demoPassword = "demo-password-123"
)

type demoUser struct {
	slug, name, gender, bio string
	birthYear               int
	interestedIn            []string
	lat, lng                float64
	interests               []string
	hue                     int
}

var demoUsers = []demoUser{
	{"ines", "Inès", "woman", "Climbing walls on Saturdays, cooking on Sundays.", 1994, []string{"man"}, 48.8566, 2.3522, []string{"hiking", "cooking", "travel"}, 10},
	{"lea", "Léa", "woman", "Bookshop wanderer and amateur ceramist.", 1991, []string{"man", "woman"}, 48.8700, 2.3300, []string{"reading", "art", "coffee"}, 40},
	{"maya", "Maya", "woman", "Runner, cat person, slow-travel enthusiast.", 1997, []string{"man"}, 48.8400, 2.3700, []string{"running", "pets", "travel"}, 70},
	{"sam", "Sam", "non_binary", "Synth tinkerer. Concerts > clubs.", 1995, []string{"man", "woman", "non_binary"}, 48.8800, 2.3400, []string{"music", "concerts", "tech"}, 100},
	{"hugo", "Hugo", "man", "Weekend cyclist, weekday engineer.", 1992, []string{"woman"}, 48.8600, 2.3600, []string{"cycling", "tech", "coffee"}, 130},
	{"noah", "Noah", "man", "Film nerd. Will debate endings.", 1996, []string{"woman"}, 48.8500, 2.3200, []string{"movies", "reading", "music"}, 160},
	{"adam", "Adam", "man", "Chef by trade, hiker by choice.", 1990, []string{"woman", "non_binary"}, 48.8900, 2.3800, []string{"cooking", "hiking", "wine"}, 190},
	{"lucas", "Lucas", "man", "Photographer chasing good light.", 1993, []string{"woman"}, 48.8300, 2.3500, []string{"photography", "travel", "art"}, 220},
	{"chloe", "Chloé", "woman", "Yoga teacher with a board game problem.", 1995, []string{"man"}, 48.8650, 2.3900, []string{"yoga", "board_games", "gardening"}, 250},
	{"yanis", "Yanis", "man", "Football on TV, dancing in real life.", 1998, []string{"woman"}, 48.8450, 2.3100, []string{"football", "dancing", "foodie"}, 280},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo users instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *reset {
		if err := resetDemo(ctx, pool, store); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := seed(ctx, pool, store); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Seeded %d demo users. Sign in with demo-<name>@%s / %s\n", len(demoUsers), demoDomain, demoPassword)
	fmt.Println("e.g. demo-ines@demo.invalid (woman) and demo-hugo@demo.invalid (man)")
}

func seed(ctx context.Context, pool *pgxpool.Pool, store storage.Store) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	for _, u := range demoUsers {
		email := fmt.Sprintf("demo-%s@%s", u.slug, demoDomain)
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := createUser(ctx, pool, store, u, email, string(hash)); err != nil {
			return fmt.Errorf("seed %s: %w", u.slug, err)
		}
	}
	return nil
}

func createUser(ctx context.Context, pool *pgxpool.Pool, store storage.Store, u demoUser, email, hash string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id::text`, email, hash).Scan(&id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, 'Paris', $6, $7, NOW())`,
		id, u.name, time.Date(u.birthYear, 5, 17, 0, 0, 0, 0, time.UTC), u.gender, u.bio, u.lat, u.lng); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id, interested_in) VALUES ($1, $2)`, id, u.interestedIn); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)`, id, u.interests); err != nil {
		return err
	}

	data, w, h, err := photos.ProcessImage(placeholder(u.hue))
	if err != nil {
		return err
	}
	key := uuid.NewString() + ".jpg"
	if err := store.Save(key, data); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes) VALUES ($1, 0, $2, $3, $4, $5)`,
		id, key, w, h, len(data)); err != nil {
		_ = store.Delete(key)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		_ = store.Delete(key)
		return err
	}
	return nil
}

func resetDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store) error {
	rows, err := pool.Query(ctx, `
		SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.email LIKE '%@' || $1`, demoDomain)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, k)
	}
	rows.Close()
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain)
	if err != nil {
		return err
	}
	for _, k := range keys {
		_ = store.Delete(k)
	}
	fmt.Fprintf(os.Stdout, "Deleted %d demo users\n", tag.RowsAffected())
	return nil
}

// placeholder renders a soft diagonal gradient so demo profiles have a photo.
func placeholder(hue int) []byte {
	const w, h = 640, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / float64(w+h)
			img.Set(x, y, color.RGBA{
				R: uint8(60 + 150*t),
				G: uint8(40 + float64(hue)*0.35 + 80*(1-t)),
				B: uint8(90 + 120*(1-t)),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}
