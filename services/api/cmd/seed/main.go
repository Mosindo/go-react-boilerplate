// Command seed creates clearly labeled demo profiles for local testing.
//
// Demo accounts use the reserved ".invalid" TLD, are flagged users.is_demo,
// carry a "Profil de démonstration" bio and the command refuses to run when
// APP_ENV=production. Remove them at any time with -purge.
//
//	go run ./cmd/seed                    # 24 demo profiles around Paris
//	go run ./cmd/seed -lat 45.76 -lon 4.84 -count 30
//	go run ./cmd/seed -like-email me@example.org   # demo users like you
//	go run ./cmd/seed -purge
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
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const demoDomain = "demo.hearth.invalid"

var (
	women     = []string{"Camille", "Léa", "Manon", "Chloé", "Inès", "Sarah", "Julie", "Zoé", "Clara", "Louise", "Nina", "Alice"}
	men       = []string{"Lucas", "Hugo", "Louis", "Nathan", "Gabriel", "Arthur", "Jules", "Adam", "Théo", "Paul", "Malik", "Yanis"}
	nonbinary = []string{"Sacha", "Charlie", "Alex", "Noa"}
	bios      = []string{
		"Toujours partant·e pour un café en terrasse et une longue balade.",
		"Je cuisine trop pour une seule personne, avis aux gourmand·es.",
		"Randonnées le week-end, concerts en semaine.",
		"Je cherche quelqu'un avec qui refaire le monde autour d'un verre.",
		"Livres, jeux de société et soirées qui finissent trop tard.",
		"Amoureux·se des musées le dimanche matin.",
	}
	interests = []string{"hiking", "coffee", "cooking", "wine", "travel", "reading", "music", "concerts", "cinema", "yoga", "running", "photography", "board-games", "museums", "dogs", "cats"}
	palettes  = [][2]color.RGBA{
		{{0xE0, 0x7A, 0x5F, 255}, {0x3D, 0x40, 0x5B, 255}},
		{{0xF2, 0xCC, 0x8F, 255}, {0x81, 0xB2, 0x9A, 255}},
		{{0xD6, 0x7D, 0x9E, 255}, {0x2B, 0x2D, 0x42, 255}},
		{{0x8D, 0x99, 0xAE, 255}, {0xEF, 0x23, 0x3C, 255}},
	}
)

func main() {
	count := flag.Int("count", 24, "number of demo profiles to create")
	lat := flag.Float64("lat", 48.8566, "center latitude")
	lon := flag.Float64("lon", 2.3522, "center longitude")
	radius := flag.Float64("radius", 15, "spread radius in km")
	likeEmail := flag.String("like-email", "", "make every demo profile like this existing account")
	purge := flag.Bool("purge", false, "delete every demo account and exit")
	flag.Parse()

	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET (32+ chars) is required to sign photo URLs consistently with the API")
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoPass123"
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
	signer := media.NewSigner([]byte(secret))
	photos := photosfeature.NewService(photosfeature.NewPGRepository(pool), store, signer)

	if *purge {
		purgeDemo(ctx, pool, photos)
		return
	}

	auth := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(secret))
	profiles := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), signer)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	runID := time.Now().Unix()

	created := 0
	for i := 0; i < *count; i++ {
		gender, name := pickIdentity(rng, i)
		email := fmt.Sprintf("demo-%d-%02d@%s", runID, i, demoDomain)
		_, user, err := auth.Register(ctx, email, password, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET is_demo = TRUE WHERE id = $1`, user.ID); err != nil {
			log.Fatal(err)
		}

		birthdate := time.Now().AddDate(-(21 + rng.Intn(25)), -rng.Intn(12), -rng.Intn(28)).Format("2006-01-02")
		bio := "Profil de démonstration — " + bios[rng.Intn(len(bios))]
		picked := pickInterests(rng)
		city := "Démo"
		if _, err := profiles.UpdateMine(ctx, user.ID, profilesfeature.UpdateProfileRequest{
			FirstName: &name, Birthdate: &birthdate, Gender: &gender, Bio: &bio, Interests: &picked,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if _, err := profiles.UpdatePreferences(ctx, user.ID, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: []string{"woman", "man", "nonbinary"}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 100,
		}); err != nil {
			log.Fatal(err)
		}
		dLat, dLon := offset(rng, *lat, *radius)
		if _, err := profiles.UpdateLocation(ctx, user.ID, *lat+dLat, *lon+dLon, &city); err != nil {
			log.Fatal(err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photos.Upload(ctx, user.ID, demoImage(palettes[(i+p)%len(palettes)], i+p)); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		if *likeEmail != "" {
			if _, err := pool.Exec(ctx, `
				INSERT INTO swipes (swiper_id, target_id, action)
				SELECT $1, id, 'like' FROM users WHERE email = lower($2)
				ON CONFLICT DO NOTHING
			`, user.ID, *likeEmail); err != nil {
				log.Fatal(err)
			}
		}
		created++
	}
	log.Printf("created %d demo profiles (password %q, emails *@%s). Remove them with: go run ./cmd/seed -purge", created, password, demoDomain)
}

func pickIdentity(rng *rand.Rand, i int) (string, string) {
	switch i % 5 {
	case 0, 2:
		return "woman", women[rng.Intn(len(women))]
	case 1, 3:
		return "man", men[rng.Intn(len(men))]
	default:
		return "nonbinary", nonbinary[rng.Intn(len(nonbinary))]
	}
}

func pickInterests(rng *rand.Rand) []string {
	perm := rng.Perm(len(interests))
	n := 3 + rng.Intn(4)
	out := make([]string, 0, n)
	for _, idx := range perm[:n] {
		out = append(out, interests[idx])
	}
	return out
}

func offset(rng *rand.Rand, lat, radiusKm float64) (float64, float64) {
	r := radiusKm * math.Sqrt(rng.Float64())
	theta := rng.Float64() * 2 * math.Pi
	dLat := (r * math.Cos(theta)) / 111.0
	dLon := (r * math.Sin(theta)) / (111.0 * math.Cos(lat*math.Pi/180))
	return dLat, dLon
}

// demoImage draws an abstract gradient so demo profiles are visibly not real
// people.
func demoImage(p [2]color.RGBA, seed int) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			wave := 0.5 + 0.5*math.Sin(float64(x+seed*37)/60.0+float64(y)/90.0)
			mix := math.Min(1, t*0.8+wave*0.2)
			img.Set(x, y, color.RGBA{
				R: uint8(float64(p[0].R)*(1-mix) + float64(p[1].R)*mix),
				G: uint8(float64(p[0].G)*(1-mix) + float64(p[1].G)*mix),
				B: uint8(float64(p[0].B)*(1-mix) + float64(p[1].B)*mix),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, photos *photosfeature.Service) {
	rows, err := pool.Query(ctx, `SELECT id FROM users WHERE is_demo = TRUE`)
	if err != nil {
		log.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		keys, err := photos.StorageKeysForUser(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM conversations WHERE id IN (SELECT conversation_id FROM conversation_participants WHERE user_id = $1)`, id); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
			log.Fatal(err)
		}
		photos.PurgeObjects(ctx, keys)
	}
	log.Printf("removed %d demo accounts", len(ids))
}
