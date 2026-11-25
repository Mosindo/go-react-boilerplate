// Command seed creates demonstration profiles for local development.
//
// Demo accounts are clearly separated from real members:
//   - email under the reserved domain demo.lueur.invalid
//   - users.is_demo = TRUE (shown with a "Démo" badge in the app)
//   - a random, discarded password: demo accounts can never sign in
//   - never shown in discovery when APP_ENV=production
//
// Usage:
//
//	go run ./cmd/seed                         # create 24 demo profiles around Paris
//	go run ./cmd/seed -count 40 -lat 45.76 -lon 4.83
//	go run ./cmd/seed -like-user you@example.com   # half of the demos like you
//	go run ./cmd/seed -reset                  # delete every demo account
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	mrand "math/rand/v2"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.lueur.invalid"

var (
	womenNames = []string{"Camille", "Léa", "Manon", "Chloé", "Inès", "Sarah", "Jade", "Louise", "Zoé", "Alice", "Juliette", "Nina"}
	menNames   = []string{"Lucas", "Hugo", "Louis", "Nathan", "Gabriel", "Arthur", "Jules", "Adam", "Raphaël", "Tom", "Noah", "Paul"}
	nbNames    = []string{"Alex", "Charlie", "Sasha", "Eden", "Noa", "Andréa"}
	bios       = []string{
		"Toujours partant·e pour un café en terrasse et une bonne discussion.",
		"Randonnées le week-end, séries le soir. Équilibre parfait.",
		"J'aime cuisiner pour les autres — tu ramènes le dessert ?",
		"Curieux·se de tout, surtout des gens. Parle-moi de ton dernier voyage.",
		"Fan de concerts, de vinyles et de longues balades au bord de l'eau.",
		"Je cherche quelqu'un avec qui refaire le monde autour d'un brunch.",
	}
	jobs    = []string{"Architecte", "Infirmier·e", "Développeur·se", "Professeur·e", "Photographe", "Chef·fe de projet", "Graphiste", "Libraire"}
	looking = []string{"relationship", "casual", "friendship", "unsure"}
)

func main() {
	count := flag.Int("count", 24, "number of demo profiles to create")
	lat := flag.Float64("lat", 48.8566, "latitude of the area to populate")
	lon := flag.Float64("lon", 2.3522, "longitude of the area to populate")
	radius := flag.Float64("radius", 15, "spread radius in km")
	likeUser := flag.String("like-user", "", "email of a real account that half of the demo profiles will like")
	reset := flag.Bool("reset", false, "delete all demo accounts and exit")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.NewLocal(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	removed, err := deleteDemo(ctx, pool, store)
	if err != nil {
		log.Fatal(err)
	}
	if *reset {
		log.Printf("deleted %d demo accounts", removed)
		return
	}

	var targetID string
	if *likeUser != "" {
		email := strings.ToLower(strings.TrimSpace(*likeUser))
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND NOT is_demo`, email).Scan(&targetID); err != nil {
			log.Fatalf("cannot find account %s: %v", email, err)
		}
	}

	interestIDs, err := loadInterestIDs(ctx, pool)
	if err != nil {
		log.Fatal(err)
	}

	rng := mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), 42))
	for i := 0; i < *count; i++ {
		if err := createDemo(ctx, pool, store, rng, i, *lat, *lon, *radius, interestIDs, targetID); err != nil {
			log.Fatalf("create demo %d: %v", i, err)
		}
	}
	log.Printf("created %d demo profiles around (%.4f, %.4f)", *count, *lat, *lon)
}

func deleteDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Storage) (int64, error) {
	rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.is_demo`)
	if err != nil {
		return 0, err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, err
	}
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		return 0, err
	}
	for _, k := range keys {
		_ = store.Delete(ctx, k)
	}
	return tag.RowsAffected(), nil
}

func loadInterestIDs(ctx context.Context, pool *pgxpool.Pool) ([]int, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM interests ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int])
}

func createDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Storage, rng *mrand.Rand, i int, lat, lon, radius float64, interestIDs []int, likeTarget string) error {
	var gender, name string
	var interestedIn []string
	switch i % 5 {
	case 0, 1:
		gender, name = "woman", womenNames[rng.IntN(len(womenNames))]
	case 2, 3:
		gender, name = "man", menNames[rng.IntN(len(menNames))]
	default:
		gender, name = "nonbinary", nbNames[rng.IntN(len(nbNames))]
	}
	switch rng.IntN(3) {
	case 0:
		interestedIn = []string{"woman", "man", "nonbinary"}
	case 1:
		interestedIn = []string{"woman", "nonbinary"}
	default:
		interestedIn = []string{"man", "nonbinary"}
	}

	age := 21 + rng.IntN(25)
	birth := time.Now().AddDate(-age, -rng.IntN(11), -rng.IntN(27))
	distance := rng.Float64() * radius
	angle := rng.Float64() * 2 * math.Pi
	pLat := lat + (distance/111.0)*math.Cos(angle)
	pLon := lon + (distance/(111.0*math.Cos(lat*math.Pi/180)))*math.Sin(angle)

	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword(secret, bcrypt.MinCost)
	if err != nil {
		return err
	}

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return err
	}
	key := hex.EncodeToString(keyBytes) + ".jpg"
	img, err := demoPortrait(rng, name)
	if err != nil {
		return err
	}
	if err := store.Save(ctx, key, img); err != nil {
		return err
	}

	picked := map[int]struct{}{}
	for len(picked) < 3+rng.IntN(4) && len(interestIDs) > 0 {
		picked[interestIDs[rng.IntN(len(interestIDs))]] = struct{}{}
	}
	interests := make([]int, 0, len(picked))
	for id := range picked {
		interests = append(interests, id)
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var userID string
		email := fmt.Sprintf("demo-%03d-%s@%s", i, hex.EncodeToString(keyBytes[:3]), demoDomain)
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, is_demo, last_active_at) VALUES ($1, $2, TRUE, NOW() - make_interval(hours => $3))
			RETURNING id`, email, string(hash), rng.IntN(200)).Scan(&userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, looking_for, latitude, longitude, location_updated_at, city)
			VALUES ($1, $2, $3, $4, $5, $6, $7, round($8::numeric, 2), round($9::numeric, 2), NOW(), $10)`,
			userID, name, birth, gender, "[Profil de démonstration] "+bios[rng.IntN(len(bios))],
			jobs[rng.IntN(len(jobs))], looking[rng.IntN(len(looking))], pLat, pLon, "Zone démo"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 18, 99, 100)`,
			userID, interestedIn); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, unnest($2::smallint[])`, userID, interests); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes, mime_type)
			VALUES ($1, $2, 0, 720, 960, $3, 'image/jpeg')`, userID, key, len(img)); err != nil {
			return err
		}
		if likeTarget != "" && i%2 == 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like')`, userID, likeTarget); err != nil {
				return err
			}
		}
		return nil
	})
}

// demoPortrait draws an abstract warm gradient with soft shapes: an obvious
// illustration rather than a photo of a real person.
func demoPortrait(rng *mrand.Rand, seed string) ([]byte, error) {
	const w, h = 720, 960
	palettes := [][2]color.RGBA{
		{{232, 115, 90, 255}, {58, 32, 64, 255}},
		{{233, 184, 114, 255}, {92, 45, 70, 255}},
		{{196, 106, 140, 255}, {34, 28, 52, 255}},
		{{120, 140, 200, 255}, {40, 30, 60, 255}},
		{{110, 170, 150, 255}, {30, 40, 50, 255}},
	}
	p := palettes[(len(seed)+rng.IntN(len(palettes)))%len(palettes)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx1, cy1 := float64(rng.IntN(w)), float64(rng.IntN(h/2))
	cx2, cy2 := float64(rng.IntN(w)), float64(h/2+rng.IntN(h/2))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			r := lerp(p[0].R, p[1].R, t)
			g := lerp(p[0].G, p[1].G, t)
			b := lerp(p[0].B, p[1].B, t)
			d1 := math.Hypot(float64(x)-cx1, float64(y)-cy1)
			d2 := math.Hypot(float64(x)-cx2, float64(y)-cy2)
			glow := math.Max(0, 1-d1/320)*60 - math.Max(0, 1-d2/280)*40
			img.Set(x, y, color.RGBA{clamp(r + glow), clamp(g + glow*0.7), clamp(b + glow*0.5), 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func lerp(a, b uint8, t float64) float64 {
	return float64(a) + (float64(b)-float64(a))*t
}

func clamp(v float64) uint8 {
	return uint8(math.Max(0, math.Min(255, v)))
}
