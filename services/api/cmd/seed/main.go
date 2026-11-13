// Command seed creates clearly-flagged demo profiles for local testing.
//
// Demo accounts use the reserved .invalid TLD, have a random password nobody
// knows, are marked users.is_demo = TRUE (shown with a "Démo" badge in the
// app) and can be purged with -purge. The command refuses to run when
// APP_ENV=production.
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
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type persona struct {
	name, gender, job, bio string
	interested           []string
	interests            []string
}

var personas = []persona{
	{"Camille", "woman", "Architecte", "Toujours partante pour une expo ou une balade au bord de l'eau.", []string{"man", "woman"}, []string{"museums", "hiking", "coffee", "photography"}},
	{"Lucas", "man", "Infirmier", "Cuisinier du dimanche, grimpeur du samedi.", []string{"woman"}, []string{"climbing", "cooking", "concerts", "travel"}},
	{"Inès", "woman", "Développeuse", "Je cherche quelqu'un avec qui refaire le monde autour d'un jeu de société.", []string{"man"}, []string{"board_games", "tech", "reading", "brunch"}},
	{"Hugo", "man", "Professeur de musique", "Pianiste, amateur de jazz et de longues discussions.", []string{"woman", "man"}, []string{"jazz", "playing_music", "wine", "cinema"}},
	{"Léa", "woman", "Vétérinaire", "Mon chien a validé ce profil.", []string{"man"}, []string{"pets", "running", "gardening", "road_trips"}},
	{"Noah", "man", "Photographe", "Lumière du matin, café serré, carnet de voyage.", []string{"woman"}, []string{"photography", "travel", "coffee", "languages"}},
	{"Sacha", "nonbinary", "Illustrateur·rice", "Peinture, podcasts et pâtisseries ratées.", []string{"woman", "man", "nonbinary"}, []string{"painting", "podcasts", "baking", "theatre"}},
	{"Chloé", "woman", "Juriste", "Danse le mardi, cinéma le jeudi, improvisation le reste du temps.", []string{"man", "woman"}, []string{"dance", "cinema", "wine", "volunteering"}},
	{"Mathis", "man", "Ingénieur", "Vélo, science et bonne humeur.", []string{"woman"}, []string{"cycling", "science", "electro", "cooking"}},
	{"Jade", "woman", "Libraire", "Je vous recommanderai forcément un livre.", []string{"man", "nonbinary"}, []string{"reading", "writing", "coffee", "meditation"}},
	{"Adam", "man", "Chef de projet", "Footballeur du dimanche et fan de brunchs.", []string{"woman"}, []string{"football", "brunch", "travel", "hiphop"}},
	{"Manon", "woman", "Designer", "Mode, yoga et marchés aux puces.", []string{"man"}, []string{"fashion", "yoga", "museums", "vegan"}},
}

var palettes = [][2]color.RGBA{
	{{0xC8, 0x55, 0x3D, 0xFF}, {0xF2, 0xC1, 0x8D, 0xFF}},
	{{0x2E, 0x4A, 0x62, 0xFF}, {0x9C, 0xC5, 0xA1, 0xFF}},
	{{0x6B, 0x3E, 0x75, 0xFF}, {0xF4, 0xA2, 0x59, 0xFF}},
	{{0x1F, 0x6F, 0x6B, 0xFF}, {0xE9, 0xD8, 0xA6, 0xFF}},
	{{0x8E, 0x2C, 0x48, 0xFF}, {0xF7, 0xCA, 0xC9, 0xFF}},
}

func main() {
	count := flag.Int("count", 24, "number of demo profiles to create")
	lat := flag.Float64("lat", 48.8566, "center latitude")
	lon := flag.Float64("lon", 2.3522, "center longitude")
	city := flag.String("city", "Paris", "city label")
	likeUser := flag.String("like-user", "", "e-mail of a real account every demo profile should like (to test matches)")
	purge := flag.Bool("purge", false, "delete every demo account and exit")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
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

	if err := purgeDemo(ctx, pool, store); err != nil {
		log.Fatal(err)
	}
	if *purge {
		log.Println("demo accounts purged")
		return
	}

	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	created := 0
	for i := 0; i < *count; i++ {
		p := personas[i%len(personas)]
		if err := createDemo(ctx, pool, store, rng, i, p, *lat, *lon, *city); err != nil {
			log.Fatalf("create demo %d: %v", i, err)
		}
		created++
	}
	log.Printf("created %d demo profiles around %s (%.4f, %.4f)", created, *city, *lat, *lon)

	if *likeUser != "" {
		tag, err := pool.Exec(ctx, `
			INSERT INTO swipes (swiper_user_id, target_user_id, action)
			SELECT d.id, u.id, 'like'
			FROM users d, users u
			WHERE d.is_demo AND u.email = lower($1)
			ON CONFLICT DO NOTHING
		`, strings.TrimSpace(*likeUser))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%d demo profiles now like %s: like them back to get a match", tag.RowsAffected(), *likeUser)
	}
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store) error {
	rows, err := pool.Query(ctx, `DELETE FROM users WHERE is_demo RETURNING id`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := store.DeletePrefix(ctx, "photos/"+id); err != nil {
			return err
		}
	}
	return rows.Err()
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func createDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store, rng *mrand.Rand, i int, p persona, lat, lon float64, city string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(randomHex(24)), bcrypt.MinCost)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	email := fmt.Sprintf("demo-%02d-%s@demo.invalid", i, randomHex(3))
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, organization_id, is_demo)
		VALUES ($1, $2, (SELECT id FROM organizations WHERE slug = 'default'), TRUE)
		RETURNING id
	`, email, string(hash)).Scan(&userID); err != nil {
		return err
	}

	age := 22 + rng.Intn(20)
	birthdate := time.Now().AddDate(-age, -rng.Intn(11), -rng.Intn(27))
	// Scatter within ~8 km of the center, rounded like real locations.
	angle := rng.Float64() * 2 * math.Pi
	radiusKm := 0.5 + rng.Float64()*8
	dLat := radiusKm / 111 * math.Cos(angle)
	dLon := radiusKm / (111 * math.Cos(lat*math.Pi/180)) * math.Sin(angle)
	height := 155 + rng.Intn(35)
	goals := []string{"long_term", "short_term", "friendship", "unsure"}

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, height_cm, relationship_goal,
		                      city, latitude, longitude, location_updated_at, last_active_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, round(($10)::numeric, 2), round(($11)::numeric, 2), NOW(),
		        NOW() - make_interval(hours => $12))
	`, userID, p.name, birthdate, p.gender, p.bio, p.job, height, goals[rng.Intn(len(goals))],
		city, lat+dLat, lon+dLon, rng.Intn(96)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, 18, 60, 100)
	`, userID, p.interested); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, userID, p.interests); err != nil {
		return err
	}

	for pos := 0; pos < 2; pos++ {
		img, err := photos.ProcessUpload(bytes.NewReader(demoImage(rng, i+pos)))
		if err != nil {
			return err
		}
		key := "photos/" + userID + "/" + randomHex(16) + ".jpg"
		if err := store.Put(ctx, key, bytes.NewReader(img.Data)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, key, pos, img.Width, img.Height, len(img.Data)); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `SELECT refresh_profile_completeness($1)`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// demoImage paints an abstract gradient: demo profiles intentionally have no
// faces so they can never be mistaken for real people.
func demoImage(rng *mrand.Rand, seed int) []byte {
	const w, h = 720, 900
	pal := palettes[seed%len(palettes)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(rng.Intn(w)), float64(rng.Intn(h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / 600
			glow := math.Max(0, 1-d) * 0.35
			mix := func(a, b uint8) uint8 {
				v := float64(a)*(1-t) + float64(b)*t + glow*255
				return uint8(math.Min(255, v))
			}
			img.Set(x, y, color.RGBA{mix(pal[0].R, pal[1].R), mix(pal[0].G, pal[1].G), mix(pal[0].B, pal[1].B), 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
