// Command seed creates clearly-labelled demonstration profiles for local
// development. Demo accounts use the reserved ".invalid" e-mail domain, carry
// users.is_demo = true (shown as a "Profil démo" badge in the app) and the
// command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed                         # (re)create demo profiles
//	go run ./cmd/seed -like-email me@mail.com # demo profiles pre-like this account
//	go run ./cmd/seed -reset                  # remove every demo profile
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
	"math/rand"
	"os"
	"strings"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.lueur.invalid"

type demo struct {
	name   string
	gender string
	seek   []string
	age    int
	job    string
	bio    string
}

var demos = []demo{
	{"Camille", "woman", []string{"man", "woman"}, 27, "Architecte", "Amoureuse des marchés du dimanche et des longues balades sans but."},
	{"Léa", "woman", []string{"man"}, 31, "Infirmière", "Je cherche quelqu'un avec qui refaire le monde autour d'un café."},
	{"Inès", "woman", []string{"man"}, 25, "Graphiste", "Toujours un carnet de croquis dans le sac."},
	{"Manon", "woman", []string{"woman"}, 29, "Libraire", "Recommandations de lecture garanties, humour discutable."},
	{"Chloé", "woman", []string{"man"}, 34, "Cheffe de projet", "Randonnée le samedi, cinéma le dimanche."},
	{"Sarah", "woman", []string{"man", "woman", "nonbinary"}, 38, "Médecin", "Curieuse de tout, surtout des gens."},
	{"Emma", "woman", []string{"man"}, 23, "Étudiante", "Fan de concerts et de cuisine du monde."},
	{"Julie", "woman", []string{"man"}, 42, "Professeure", "J'aime les gens qui rient fort."},
	{"Lucas", "man", []string{"woman"}, 28, "Développeur", "Je fais un excellent risotto, c'est prouvé."},
	{"Hugo", "man", []string{"woman"}, 33, "Photographe", "Toujours partant pour un week-end improvisé."},
	{"Thomas", "man", []string{"woman", "man"}, 30, "Kiné", "Grimpeur du dimanche, lecteur du soir."},
	{"Nathan", "man", []string{"woman"}, 26, "Ingénieur", "Je cherche une complice pour explorer les musées."},
	{"Karim", "man", []string{"woman"}, 36, "Chef cuisinier", "Je cuisine, tu choisis le vin ?"},
	{"Antoine", "man", []string{"man"}, 32, "Avocat", "Jeux de société et débats passionnés."},
	{"Maxime", "man", []string{"woman"}, 40, "Entrepreneur", "Café serré et grandes idées."},
	{"Yanis", "man", []string{"woman"}, 24, "Musicien", "Guitare, vinyles et couchers de soleil."},
	{"Alex", "nonbinary", []string{"woman", "man", "nonbinary"}, 29, "Designer", "Plantes vertes, podcasts et cuisine végétale."},
	{"Sacha", "nonbinary", []string{"woman", "nonbinary"}, 27, "Danseur·se", "La vie est meilleure en musique."},
}

var palettes = [][2]color.RGBA{
	{{0xE8, 0x7A, 0x5D, 255}, {0x3B, 0x1F, 0x2B, 255}},
	{{0xF2, 0xB4, 0x8C, 255}, {0x5B, 0x2A, 0x3C, 255}},
	{{0xC9, 0x6F, 0x8B, 255}, {0x23, 0x1A, 0x2E, 255}},
	{{0xF4, 0xD3, 0x9C, 255}, {0x8E, 0x3B, 0x46, 255}},
	{{0x9F, 0x86, 0xC0, 255}, {0x2A, 0x1E, 0x38, 255}},
}

func main() {
	likeEmail := flag.String("like-email", "", "demo profiles will already like this account (to test matches)")
	reset := flag.Bool("reset", false, "only delete existing demo profiles")
	lat := flag.Float64("lat", 48.8566, "centre latitude for demo profiles")
	lng := flag.Float64("lng", 2.3522, "centre longitude for demo profiles")
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
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoPass123"
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
	store, err := storage.NewLocalStore(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	removed, err := removeDemo(ctx, pool, store)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("removed %d existing demo profiles", removed)
	if *reset {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	var likeTarget string
	if *likeEmail != "" {
		err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(*likeEmail))).Scan(&likeTarget)
		if err != nil {
			log.Fatalf("account %s not found: register it in the app first", *likeEmail)
		}
	}

	rng := rand.New(rand.NewSource(42))
	for i, d := range demos {
		if err := createDemo(ctx, pool, store, rng, i, d, string(hash), *lat, *lng, likeTarget); err != nil {
			log.Fatalf("create %s: %v", d.name, err)
		}
	}
	log.Printf("created %d demo profiles (password: SEED_PASSWORD or the default from README)", len(demos))
	if likeTarget != "" {
		log.Printf("every demo profile already likes %s: like them back to get a match", *likeEmail)
	}
}

func removeDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store) (int, error) {
	rows, err := pool.Query(ctx, `SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.is_demo`)
	if err != nil {
		return 0, err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return 0, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		return 0, err
	}
	for _, k := range keys {
		_ = store.Delete(ctx, k)
	}
	return int(tag.RowsAffected()), nil
}

func createDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store, rng *rand.Rand, i int, d demo, hash string, lat, lng float64, likeTarget string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	email := fmt.Sprintf("%s.%d@%s", strings.ToLower(asciiName(d.name)), i+1, demoDomain)
	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_demo, last_active_at)
		VALUES ($1, $2, TRUE, NOW() - make_interval(hours => $3))
		RETURNING id
	`, email, hash, rng.Intn(24*10)).Scan(&userID); err != nil {
		return err
	}

	birthdate := time.Now().AddDate(-d.age, -rng.Intn(11), -rng.Intn(27))
	// Scatter within ~15 km, rounded like real locations.
	dLat := (rng.Float64()*2 - 1) * 0.13
	dLng := (rng.Float64()*2 - 1) * 0.2
	pLat := math.Round((lat+dLat)*100) / 100
	pLng := math.Round((lng+dLng)*100) / 100
	height := 155 + rng.Intn(40)
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, height_cm, city,
		                      approx_latitude, approx_longitude, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'Paris', $8, $9, NOW())
	`, userID, d.name, birthdate, d.gender, d.bio, d.job, height, pLat, pLng); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, 18, 99, 100)
	`, userID, d.seek); err != nil {
		return err
	}
	interestIDs := rng.Perm(32)[:3+rng.Intn(4)]
	for j := range interestIDs {
		interestIDs[j]++
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id) SELECT $1, unnest($2::smallint[])
	`, userID, interestIDs); err != nil {
		return err
	}

	for pos := 0; pos < 2+rng.Intn(2); pos++ {
		img, err := photos.ProcessImage(demoImage(rng, i+pos))
		if err != nil {
			return err
		}
		key, err := storage.NewKey()
		if err != nil {
			return err
		}
		if err := store.Put(ctx, key, img.Data); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, mime_type, width, height, size_bytes, position)
			VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6)
		`, userID, key, img.Width, img.Height, len(img.Data), pos); err != nil {
			return err
		}
	}

	if likeTarget != "" {
		if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like')`, userID, likeTarget); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// demoImage paints an abstract gradient with soft circles: obviously not a
// real person's photo.
func demoImage(rng *rand.Rand, seed int) []byte {
	const w, h = 720, 960
	p := palettes[seed%len(palettes)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	type circle struct{ x, y, r float64 }
	circles := make([]circle, 4)
	for i := range circles {
		circles[i] = circle{rng.Float64() * w, rng.Float64() * h, 80 + rng.Float64()*200}
	}
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			r := lerp(p[0].R, p[1].R, t)
			g := lerp(p[0].G, p[1].G, t)
			b := lerp(p[0].B, p[1].B, t)
			for _, c := range circles {
				d := math.Hypot(float64(x)-c.x, float64(y)-c.y)
				if d < c.r {
					k := 0.18 * (1 - d/c.r)
					r, g, b = r+(255-r)*k, g+(255-g)*k, b+(255-b)*k
				}
			}
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func lerp(a, b uint8, t float64) float64 { return float64(a) + (float64(b)-float64(a))*t }

func asciiName(name string) string {
	replacer := strings.NewReplacer("é", "e", "è", "e", "ï", "i", "·", "")
	return replacer.Replace(name)
}
