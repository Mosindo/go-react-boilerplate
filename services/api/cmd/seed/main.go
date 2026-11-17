// Command seed creates clearly-flagged demonstration accounts for local testing.
//
// Demo accounts use the reserved ".invalid" e-mail domain (RFC 2606) so they can never
// collide with a real address, carry users.is_demo = TRUE and are shown with a "Démo"
// badge in the app. The command refuses to run when APP_ENV=production.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	mrand "math/rand/v2"
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

const demoDomain = "demo.lueur.invalid"

type persona struct {
	name, gender string
	interestedIn []string
	age          int
	job, bio     string
	interests    []string
}

var personas = []persona{
	{"Camille", "woman", []string{"man", "woman"}, 28, "Architecte", "Toujours partante pour un brunch ou une expo photo.", []string{"photography", "museums", "coffee", "travel"}},
	{"Lucas", "man", []string{"woman"}, 31, "Kiné", "Grimpeur du dimanche, cuisinier du lundi.", []string{"climbing", "cooking", "hiking", "board_games"}},
	{"Inès", "woman", []string{"man"}, 26, "Data analyst", "Je collectionne les vinyles et les cafés de quartier.", []string{"music_live", "coffee", "reading", "jazz"}},
	{"Hugo", "man", []string{"woman", "man"}, 34, "Chef de projet", "Vélo le matin, cinéma le soir.", []string{"cycling", "cinema", "wine", "road_trips"}},
	{"Sacha", "nonbinary", []string{"woman", "man", "nonbinary"}, 29, "Illustratrice", "Je dessine dans les trains. Parle-moi de ton dernier livre.", []string{"painting", "reading", "theatre", "animals"}},
	{"Léa", "woman", []string{"man", "nonbinary"}, 30, "Infirmière", "Yoga, randonnées et soirées jeux de société.", []string{"yoga", "hiking", "board_games", "gardening"}},
	{"Nathan", "man", []string{"woman"}, 27, "Développeur", "Je fais du pain au levain et du code propre.", []string{"tech", "baking", "video_games", "running"}},
	{"Chloé", "woman", []string{"woman"}, 32, "Journaliste", "Toujours un carnet dans la poche.", []string{"writing", "history", "travel", "wine"}},
	{"Yanis", "man", []string{"woman"}, 29, "Prof de musique", "Guitare, jazz et longues marches.", []string{"music_play", "jazz", "hiking", "languages"}},
	{"Manon", "woman", []string{"man"}, 25, "Étudiante en médecine", "Danse, plage et séries coréennes.", []string{"dance", "beach", "series", "swimming"}},
	{"Alex", "nonbinary", []string{"woman", "nonbinary"}, 33, "Paysagiste", "Les plantes d'abord, les gens ensuite (je plaisante).", []string{"gardening", "camping", "meditation", "vegetarian"}},
	{"Thomas", "man", []string{"woman", "man"}, 36, "Photographe", "Mon objectif : le prochain coucher de soleil.", []string{"photography", "mountains", "travel", "coffee"}},
	{"Sarah", "woman", []string{"man", "woman"}, 27, "UX designer", "Fan de design, de tennis et de bons restos.", []string{"tennis", "restaurants", "fashion", "museums"}},
	{"Mehdi", "man", []string{"woman"}, 30, "Ingénieur", "Football le samedi, bénévolat le dimanche.", []string{"football", "volunteering", "science", "cooking"}},
	{"Jade", "woman", []string{"man"}, 24, "Barista", "Latte art et électro.", []string{"coffee", "electronic", "nightlife", "dance"}},
	{"Paul", "man", []string{"woman"}, 38, "Libraire", "Je te prête un livre si tu me prêtes ton temps.", []string{"reading", "writing", "cinema", "wine"}},
}

func main() {
	confirm := flag.Bool("yes-demo", false, "required: confirm that demo accounts should be created")
	reset := flag.Bool("reset", false, "delete existing demo accounts first")
	lat := flag.Float64("lat", 48.8566, "latitude of the demo area")
	lng := flag.Float64("lng", 2.3522, "longitude of the demo area")
	radiusKm := flag.Float64("radius-km", 8, "spread of demo profiles around the centre")
	likeEmail := flag.String("like-email", "", "make every demo account like this real account (to test matches quickly)")
	flag.Parse()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}
	if !*confirm {
		log.Fatal("pass --yes-demo to create demonstration accounts")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
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
	store, err := storage.NewLocalStore(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *reset {
		if err := deleteDemo(ctx, pool, store); err != nil {
			log.Fatal(err)
		}
	}

	password := os.Getenv("SEED_DEMO_PASSWORD")
	if password == "" {
		password = randomPassword()
	}
	if err := validatePassword(password); err != nil {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	rng := mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), 7))
	created := 0
	for i, p := range personas {
		email := fmt.Sprintf("demo-%02d@%s", i+1, demoDomain)
		pLat, pLng := scatter(rng, *lat, *lng, *radiusKm)
		userID, err := createPersona(ctx, pool, store, email, string(hash), p, pLat, pLng, i)
		if err != nil {
			log.Fatalf("create %s: %v", email, err)
		}
		if userID != "" {
			created++
		}
	}

	if *likeEmail != "" {
		// Every demo account (new or existing) likes the given real account.
		tag, err := pool.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action)
			SELECT d.id, u.id, 'like'
			FROM users d, users u
			WHERE d.is_demo AND u.email = lower(btrim($1)) AND u.is_demo = FALSE
			ON CONFLICT DO NOTHING
		`, *likeEmail)
		if err != nil {
			log.Fatalf("like %s: %v", *likeEmail, err)
		}
		fmt.Printf("%d demo likes sent to %s.\n", tag.RowsAffected(), *likeEmail)
	}

	fmt.Printf("Created %d demo accounts (domain %s).\n", created, demoDomain)
	fmt.Printf("Sign in with demo-01@%s … demo-%02d@%s\n", demoDomain, len(personas), demoDomain)
	if os.Getenv("SEED_DEMO_PASSWORD") == "" {
		fmt.Printf("Generated demo password: %s\n", password)
	}
}

func createPersona(ctx context.Context, pool *pgxpool.Pool, store storage.Store, email, hash string, p persona, lat, lng float64, index int) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_demo, last_active_at)
		VALUES ($1, $2, TRUE, NOW() - make_interval(hours => $3))
		ON CONFLICT (email) DO NOTHING
		RETURNING id
	`, email, hash, index*3).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	birthdate := time.Now().AddDate(-p.age, -index%11, -index*3)
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, latitude, longitude, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'Paris', $7, $8, NOW())
	`, userID, p.name, birthdate, p.gender, p.bio, p.job, profiles.RoundCoordinate(lat), profiles.RoundCoordinate(lng)); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, 18, 60, 100)
	`, userID, p.interestedIn); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, userID, p.interests); err != nil {
		return "", err
	}
	for position := 0; position < 2; position++ {
		img, err := photos.ProcessImage(demoPicture(index*2 + position))
		if err != nil {
			return "", err
		}
		key := "photos/demo-" + randomHex(12) + ".jpg"
		if err := store.Put(ctx, key, img.Data); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes, mime_type)
			VALUES ($1, $2, $3, $4, $5, $6, 'image/jpeg')
		`, userID, key, position, img.Width, img.Height, len(img.Data)); err != nil {
			return "", err
		}
	}
	return userID, tx.Commit(ctx)
}

func deleteDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store) error {
	rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.is_demo`)
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
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		return err
	}
	for _, k := range keys {
		_ = store.Delete(ctx, k)
	}
	fmt.Printf("Deleted %d demo accounts.\n", tag.RowsAffected())
	return nil
}

// demoPicture renders an abstract warm gradient portrait placeholder (no real faces).
func demoPicture(seed int) []byte {
	const w, h = 720, 960
	palettes := [][2]color.RGBA{
		{{0xF2, 0x6B, 0x4F, 255}, {0x3A, 0x1C, 0x32, 255}},
		{{0xE9, 0xA8, 0x5C, 255}, {0x2B, 0x1B, 0x3D, 255}},
		{{0xD9, 0x5D, 0x7A, 255}, {0x1F, 0x2A, 0x44, 255}},
		{{0x8C, 0xB3, 0x9A, 255}, {0x26, 0x1E, 0x2F, 255}},
		{{0x7A, 0x9C, 0xE0, 255}, {0x2A, 0x17, 0x2B, 255}},
	}
	pal := palettes[seed%len(palettes)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)*(0.3+0.4*float64(seed%3)/2), float64(h)*0.38
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / 420
			glow := math.Max(0, 1-d)
			mix := func(a, b uint8) uint8 {
				v := float64(a)*(1-t) + float64(b)*t + glow*60
				return uint8(math.Min(255, v))
			}
			img.SetRGBA(x, y, color.RGBA{mix(pal[0].R, pal[1].R), mix(pal[0].G, pal[1].G), mix(pal[0].B, pal[1].B), 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func scatter(rng *mrand.Rand, lat, lng, radiusKm float64) (float64, float64) {
	r := radiusKm * math.Sqrt(rng.Float64())
	theta := rng.Float64() * 2 * math.Pi
	dLat := r * math.Cos(theta) / 111.0
	dLng := r * math.Sin(theta) / (111.0 * math.Cos(lat*math.Pi/180))
	return lat + dLat, lng + dLng
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func randomPassword() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "Demo" + base64.RawURLEncoding.EncodeToString(b) + "1"
}

func validatePassword(p string) error {
	if len(p) < 8 || !strings.ContainsAny(p, "0123456789") {
		return fmt.Errorf("SEED_DEMO_PASSWORD must have at least 8 characters including a digit")
	}
	return nil
}
