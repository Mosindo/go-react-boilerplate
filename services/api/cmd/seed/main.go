// Command seed creates clearly-labelled demo profiles for local testing.
//
//	go run ./cmd/seed                 # 24 demo profiles around Paris
//	go run ./cmd/seed -lat 45.76 -lon 4.84 -count 30
//	go run ./cmd/seed -purge          # delete every demo account
//
// Demo accounts are flagged users.is_demo = true, use the reserved
// "@demo.lueur.invalid" domain, carry "Profil de démonstration" in their bio,
// and the command refuses to run when APP_ENV=production.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.lueur.invalid"

var (
	women   = []string{"Camille", "Léa", "Manon", "Chloé", "Inès", "Sarah", "Jade", "Louise", "Alice", "Emma", "Zoé", "Nina"}
	men     = []string{"Lucas", "Hugo", "Louis", "Nathan", "Gabriel", "Arthur", "Jules", "Adam", "Raphaël", "Tom", "Noah", "Paul"}
	nb      = []string{"Sacha", "Charlie", "Alex", "Eden"}
	jobs    = []string{"Architecte", "Infirmier·e", "Développeur·se", "Professeur·e", "Photographe", "Chef·fe de projet", "Designer", "Étudiant·e", "Libraire", "Ingénieur·e"}
	bios    = []string{"Amateur·rice de cafés de quartier et de longues balades.", "Toujours partant·e pour un concert ou une expo.", "Je cuisine mieux que je ne chante, promis.", "Week-ends en montagne, soirées jeux de société.", "À la recherche de bonnes conversations et de fous rires."}
	goals   = []string{"long_term", "short_term", "friendship", "undecided"}
	palette = [][2]color.RGBA{
		{{0xE0, 0x7A, 0x5F, 255}, {0x3D, 0x40, 0x5B, 255}},
		{{0xF2, 0xCC, 0x8F, 255}, {0x81, 0xB2, 0x9A, 255}},
		{{0x9B, 0x5D, 0xE5, 255}, {0xF1, 0x5B, 0xB5, 255}},
		{{0x2A, 0x9D, 0x8F, 255}, {0xE9, 0xC4, 0x6A, 255}},
		{{0x26, 0x46, 0x53, 255}, {0xF4, 0xA2, 0x61, 255}},
	}
)

func main() {
	count := flag.Int("count", 24, "number of demo profiles to create")
	lat := flag.Float64("lat", 48.8566, "center latitude")
	lon := flag.Float64("lon", 2.3522, "center longitude")
	radius := flag.Float64("radius", 15, "spread radius in km")
	purge := flag.Bool("purge", false, "delete all demo accounts and exit")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	password := os.Getenv("DEMO_PASSWORD")
	if password == "" {
		password = "DemoLueur2026"
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, databaseURL, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.NewLocalStore(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		rows, err := pool.Query(ctx, `SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			log.Fatal(err)
		}
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		for _, k := range keys {
			_ = store.Delete(ctx, k)
		}
		log.Printf("deleted %d demo accounts", tag.RowsAffected())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	created := 0
	for i := 0; i < *count; i++ {
		gender, name, interested := pick(i)
		email := fmt.Sprintf("demo-%s-%d@%s", strings.ToLower(asciiOnly(name)), time.Now().UnixNano()%1_000_000+int64(i), demoDomain)
		birth := time.Now().AddDate(-(21 + rand.IntN(20)), -rand.IntN(12), -rand.IntN(28))
		dlat, dlon := randomOffset(*lat, *radius)
		job := jobs[rand.IntN(len(jobs))]
		bio := bios[rand.IntN(len(bios))] + " (Profil de démonstration)"
		goal := goals[rand.IntN(len(goals))]

		err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			var userID string
			if err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE) RETURNING id`, email, string(hash)).Scan(&userID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, relationship_goal, latitude, longitude, location_updated_at, last_active_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'Démo', $7, $8, $9, NOW(), NOW() - make_interval(hours => $10))
			`, userID, name, birth, gender, bio, job, goal, geo.Coarsen(*lat+dlat), geo.Coarsen(*lon+dlon), rand.IntN(96)); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 18, 60, 100)
			`, userID, interested); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id)
				SELECT $1, id FROM interests ORDER BY random() LIMIT 4
			`, userID); err != nil {
				return err
			}
			for pos := 0; pos < 1+rand.IntN(3); pos++ {
				processed, err := photos.Process(demoImage(i*7 + pos))
				if err != nil {
					return err
				}
				key := fmt.Sprintf("%016x%016x.jpg", rand.Uint64(), rand.Uint64())
				if err := store.Put(ctx, key, processed.Data); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes) VALUES ($1, $2, $3, $4, $5, $6)
				`, userID, pos, key, processed.Width, processed.Height, len(processed.Data)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Fatalf("seed profile %d: %v", i, err)
		}
		created++
	}
	log.Printf("created %d demo profiles around %.4f,%.4f (password for every demo account: %q)", created, *lat, *lon, password)
}

func pick(i int) (gender, name string, interestedIn []string) {
	switch {
	case i%9 == 8:
		return "nonbinary", nb[i%len(nb)], []string{"woman", "man", "nonbinary"}
	case i%2 == 0:
		if i%6 == 0 {
			return "woman", women[i%len(women)], []string{"woman", "man"}
		}
		return "woman", women[i%len(women)], []string{"man"}
	default:
		if i%7 == 0 {
			return "man", men[i%len(men)], []string{"man"}
		}
		return "man", men[i%len(men)], []string{"woman"}
	}
}

func randomOffset(lat, radiusKm float64) (float64, float64) {
	r := radiusKm * math.Sqrt(rand.Float64())
	theta := rand.Float64() * 2 * math.Pi
	dLat := r * math.Cos(theta) / 111.0
	dLon := r * math.Sin(theta) / (111.320 * math.Cos(lat*math.Pi/180))
	return dLat, dLon
}

func asciiOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 128 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// demoImage paints an abstract portrait-format gradient (no real faces).
func demoImage(seed int) []byte {
	const w, h = 720, 960
	pal := palette[seed%len(palette)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)/2+float64(seed%5-2)*40, float64(h)*0.42
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			c := color.RGBA{
				R: uint8(float64(pal[0].R)*(1-t) + float64(pal[1].R)*t),
				G: uint8(float64(pal[0].G)*(1-t) + float64(pal[1].G)*t),
				B: uint8(float64(pal[0].B)*(1-t) + float64(pal[1].B)*t),
				A: 255,
			}
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			if d < 150 {
				c.R, c.G, c.B = lighten(c.R), lighten(c.G), lighten(c.B)
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func lighten(v uint8) uint8 { return uint8(float64(v) + (255-float64(v))*0.35) }
