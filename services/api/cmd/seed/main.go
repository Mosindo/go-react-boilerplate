// Command seed fills a development database with clearly marked demo
// profiles so the app can be tried end to end without real users.
//
// Demo accounts use the reserved ".invalid" e-mail domain, carry
// profiles.is_demo = TRUE and a "Profil de démonstration" bio prefix. The
// command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed                      # create 24 demo profiles around Paris
//	go run ./cmd/seed -like-email me@x.com # demo profiles pre-like this account
//	go run ./cmd/seed -reset               # remove every demo account
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
	"image/png"
	"log"
	"math"
	mrand "math/rand"
	"os"
	"strings"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	demoDomain   = "demo.seren.invalid"
	demoPassword = "DemoPassword1"
)

var (
	womenNames = []string{"Camille", "Léa", "Manon", "Chloé", "Inès", "Jade", "Louise", "Zoé", "Alice", "Lina"}
	menNames   = []string{"Hugo", "Lucas", "Nathan", "Théo", "Louis", "Jules", "Arthur", "Adam", "Gabriel", "Noah"}
	nbNames    = []string{"Sacha", "Charlie", "Alex", "Eden", "Noa", "Andrea"}
	bios       = []string{
		"Toujours partant·e pour un café en terrasse et une longue balade.",
		"Je cuisine mieux que je ne chante, mais je chante quand même.",
		"Amoureux·se des musées le dimanche et des concerts le vendredi.",
		"Je cherche quelqu'un avec qui refaire le monde autour d'un brunch.",
		"Rando le week-end, séries le soir, curiosité tout le temps.",
	}
	jobs = []string{"Architecte", "Infirmière", "Développeur", "Professeur", "Designer", "Libraire", "Chef de projet", ""}
)

func main() {
	count := flag.Int("count", 24, "number of demo profiles to create")
	lat := flag.Float64("lat", 48.8566, "latitude around which demo profiles are placed")
	lng := flag.Float64("lng", 2.3522, "longitude around which demo profiles are placed")
	likeEmail := flag.String("like-email", "", "make every demo profile like this account (instant match when you like back)")
	reset := flag.Bool("reset", false, "delete all demo accounts and exit")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}
	store, err := storage.NewLocal(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if err := resetDemo(ctx, pool, store); err != nil {
		log.Fatal(err)
	}
	if *reset {
		log.Println("demo accounts removed")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), 12)
	if err != nil {
		log.Fatal(err)
	}
	interests, err := interestSlugs(ctx, pool)
	if err != nil {
		log.Fatal(err)
	}

	var likeTarget string
	if *likeEmail != "" {
		err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(*likeEmail))).Scan(&likeTarget)
		if err != nil {
			log.Fatalf("account %s not found: register it in the app first (%v)", *likeEmail, err)
		}
	}

	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	for i := 0; i < *count; i++ {
		gender, name := pick(rng, i)
		email := fmt.Sprintf("demo%02d@%s", i+1, demoDomain)
		if err := createDemo(ctx, pool, store, rng, demoUser{
			email: email, hash: string(hash), name: name, gender: gender,
			lat: *lat, lng: *lng, interests: interests, likeTarget: likeTarget,
		}); err != nil {
			log.Fatalf("create %s: %v", email, err)
		}
	}
	log.Printf("created %d demo profiles (password: %s, e-mails demoNN@%s)", *count, demoPassword, demoDomain)
}

func pick(rng *mrand.Rand, i int) (string, string) {
	switch i % 5 {
	case 0, 2:
		return "woman", womenNames[rng.Intn(len(womenNames))]
	case 1, 3:
		return "man", menNames[rng.Intn(len(menNames))]
	default:
		return "nonbinary", nbNames[rng.Intn(len(nbNames))]
	}
}

type demoUser struct {
	email, hash, name, gender string
	lat, lng                  float64
	interests                 []string
	likeTarget                string
}

func createDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Storage, rng *mrand.Rand, u demoUser) error {
	age := 21 + rng.Intn(25)
	birth := time.Now().AddDate(-age, -rng.Intn(11), -rng.Intn(27))
	// Spread within ~10 km.
	dLat := (rng.Float64()*2 - 1) * 0.09
	dLng := (rng.Float64()*2 - 1) * 0.09 / math.Max(0.2, math.Cos(u.lat*math.Pi/180))
	interestedIn := map[string][]string{"woman": {"man"}, "man": {"woman"}, "nonbinary": {"woman", "man", "nonbinary"}}[u.gender]
	if rng.Intn(4) == 0 {
		interestedIn = []string{"woman", "man", "nonbinary"}
	}

	type stored struct {
		key string
		img photos.Processed
	}
	var files []stored
	for p := 0; p < 2+rng.Intn(2); p++ {
		img, err := photos.Process(abstractImage(rng))
		if err != nil {
			return err
		}
		key := randomKey()
		if err := store.Put(key, img.Data); err != nil {
			return err
		}
		files = append(files, stored{key: key, img: img})
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`, u.email, u.hash).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, latitude, longitude, location_updated_at, is_demo)
			VALUES ($1, $2, $3, $4, $5, $6, 'Démo', $7, $8, NOW(), TRUE)
		`, id, u.name, birth, u.gender, "Profil de démonstration · "+bios[rng.Intn(len(bios))], jobs[rng.Intn(len(jobs))],
			profiles.RoundCoordinate(u.lat+dLat), profiles.RoundCoordinate(u.lng+dLng)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km) VALUES ($1, $2, 18, 99, 100)
		`, id, interestedIn); err != nil {
			return err
		}
		rng.Shuffle(len(u.interests), func(a, b int) { u.interests[a], u.interests[b] = u.interests[b], u.interests[a] })
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, id, u.interests[:3+rng.Intn(3)]); err != nil {
			return err
		}
		for pos, f := range files {
			if _, err := tx.Exec(ctx, `
				INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1, $2, $3, $4, $5, $6)
			`, id, f.key, pos, f.img.Width, f.img.Height, len(f.img.Data)); err != nil {
				return err
			}
		}
		if u.likeTarget != "" {
			if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, direction) VALUES ($1, $2, 'like')`, id, u.likeTarget); err != nil {
				return err
			}
		}
		return nil
	})
}

func resetDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Storage) error {
	rows, err := pool.Query(ctx, `
		SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.email LIKE '%@' || $1
	`, demoDomain)
	if err != nil {
		return err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain); err != nil {
		return err
	}
	for _, k := range keys {
		_ = store.Delete(k)
	}
	return nil
}

func interestSlugs(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT slug FROM interests`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func randomKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b) + ".jpg"
}

// abstractImage draws a warm gradient with soft circles: obviously not a
// photo of a real person.
func abstractImage(rng *mrand.Rand) []byte {
	const w, h = 900, 1200
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	palettes := [][2]color.RGBA{
		{{0xE8, 0x6A, 0x4F, 255}, {0x3B, 0x1F, 0x3A, 255}},
		{{0xF2, 0xB2, 0x6B, 255}, {0x6B, 0x2D, 0x5C, 255}},
		{{0xD9, 0x7B, 0x93, 255}, {0x2E, 0x2A, 0x4F, 255}},
		{{0x9C, 0xC5, 0xA1, 255}, {0x2F, 0x4B, 0x5A, 255}},
	}
	pal := palettes[rng.Intn(len(palettes))]
	for y := 0; y < h; y++ {
		t := float64(y) / h
		c := color.RGBA{
			uint8(float64(pal[0].R)*(1-t) + float64(pal[1].R)*t),
			uint8(float64(pal[0].G)*(1-t) + float64(pal[1].G)*t),
			uint8(float64(pal[0].B)*(1-t) + float64(pal[1].B)*t), 255,
		}
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	for n := 0; n < 4; n++ {
		cx, cy, r := rng.Intn(w), rng.Intn(h), 80+rng.Intn(220)
		for y := max(0, cy-r); y < min(h, cy+r); y++ {
			for x := max(0, cx-r); x < min(w, cx+r); x++ {
				if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r {
					o := img.RGBAAt(x, y)
					img.SetRGBA(x, y, color.RGBA{uint8(min(255, int(o.R)+28)), uint8(min(255, int(o.G)+22)), uint8(min(255, int(o.B)+18)), 255})
				}
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
