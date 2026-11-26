// Command seed creates clearly-labelled demo members for local testing.
//
// Demo accounts are flagged users.is_demo = true, use the reserved ".invalid" e-mail domain
// and are shown with a "Démo" badge in the app. The command refuses to run when
// APP_ENV=production. Re-running it replaces previous demo data.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type demo struct {
	name, gender, job, city, bio string
	age                          int
	interestedIn                 []string
	interests                    []string
}

var demos = []demo{
	{"Camille", "woman", "Architecte", "Paris", "Amoureuse des marchés du dimanche et des longues balades sans but.", 29, []string{"man"}, []string{"museums", "coffee", "hiking", "photography"}},
	{"Hugo", "man", "Ingénieur son", "Paris", "Je cherche le meilleur concert de jazz de la ville. Tu viens ?", 31, []string{"woman"}, []string{"jazz", "concerts", "cooking", "cycling"}},
	{"Inès", "woman", "Médecin", "Montreuil", "Entre deux gardes : escalade, romans policiers et brunchs.", 33, []string{"man", "woman"}, []string{"climbing", "reading", "brunch", "travel"}},
	{"Lucas", "man", "Prof de maths", "Vincennes", "Jeux de société, randonnées et un faible pour les crêpes.", 28, []string{"woman"}, []string{"board_games", "hiking", "cooking", "podcasts"}},
	{"Sarah", "woman", "Designer", "Paris", "Je dessine, je voyage, je goûte tout ce qui se mange.", 27, []string{"man"}, []string{"drawing", "travel", "wine", "cinema"}},
	{"Thomas", "man", "Kiné", "Boulogne", "Course à pied le matin, cinéma le soir.", 34, []string{"woman"}, []string{"running", "cinema", "series", "coffee"}},
	{"Alex", "nonbinary", "Développeur·se", "Paris", "Tech le jour, synthés modulaires la nuit.", 30, []string{"woman", "man", "nonbinary"}, []string{"tech", "electro", "video_games", "vegetarian"}},
	{"Manon", "woman", "Journaliste", "Saint-Denis", "Curieuse de tout, surtout des gens.", 32, []string{"man", "nonbinary"}, []string{"writing", "podcasts", "theatre", "languages"}},
	{"Nathan", "man", "Cuisinier", "Paris", "Je te prépare un risotto si tu choisis le vin.", 29, []string{"woman"}, []string{"cooking", "wine", "festivals", "football"}},
	{"Chloé", "woman", "Vétérinaire", "Issy-les-Moulineaux", "Deux chats, un vélo, zéro sens de l'orientation.", 26, []string{"man"}, []string{"animals", "cycling", "yoga", "gardening"}},
	{"Karim", "man", "Avocat", "Paris", "Tennis le samedi, expos le dimanche.", 36, []string{"woman"}, []string{"tennis", "museums", "travel", "reading"}},
	{"Jade", "woman", "Photographe", "Pantin", "Toujours un appareil à la main. Lumière dorée obligatoire.", 30, []string{"man", "woman"}, []string{"photography", "roadtrips", "concerts", "coffee"}},
	{"Léo", "man", "Étudiant en master", "Paris", "Danse salsa, bénévolat, et beaucoup trop de podcasts.", 24, []string{"woman"}, []string{"dance", "volunteering", "podcasts", "languages"}},
	{"Emma", "woman", "Chargée de projet", "Levallois", "Yoga, jardinage sur balcon et thé glacé maison.", 35, []string{"man"}, []string{"yoga", "gardening", "meditation", "brunch"}},
	{"Sacha", "nonbinary", "Comédien·ne", "Paris", "Théâtre, improvisation et rires garantis.", 27, []string{"man", "woman", "nonbinary"}, []string{"theatre", "cinema", "dance", "festivals"}},
	{"Maxime", "man", "Pompier", "Créteil", "Natation, montagne et barbecues entre amis.", 33, []string{"woman"}, []string{"swimming", "hiking", "gym", "roadtrips"}},
}

func main() {
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
		password = "DemoPassword1"
	}
	lat := envFloat("SEED_LATITUDE", 48.8566)
	lng := envFloat("SEED_LONGITUDE", 2.3522)

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	storage, err := media.NewLocalStorage(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if err := purgeDemo(ctx, pool, storage); err != nil {
		log.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatal(err)
	}
	rng := rand.New(rand.NewSource(42))
	ids := make([]string, 0, len(demos))
	for i, d := range demos {
		id, err := createDemo(ctx, pool, storage, rng, i, d, string(hash), lat, lng)
		if err != nil {
			log.Fatalf("create %s: %v", d.name, err)
		}
		ids = append(ids, id)
	}

	// Demo members like a few of each other so likes/matches exist between them.
	for i, id := range ids {
		for j := 1; j <= 2; j++ {
			target := ids[(i+j*3)%len(ids)]
			if target != id {
				_, _ = pool.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, id, target)
			}
		}
	}

	// Optionally make every demo member like an existing test account, so liking a demo
	// profile from that account produces an instant match.
	if likeEmail := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_LIKE_EMAIL"))); likeEmail != "" {
		tag, err := pool.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action)
			SELECT d.id, u.id, 'like' FROM users d CROSS JOIN users u
			WHERE d.is_demo AND u.email = $1 AND NOT u.is_demo
			ON CONFLICT DO NOTHING
		`, likeEmail)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("demo members liked %s (%d likes)", likeEmail, tag.RowsAffected())
	}

	log.Printf("seeded %d demo members (e-mail demo-<n>@demo.lueur.invalid, password from SEED_PASSWORD or the documented default)", len(ids))
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, storage media.Storage) error {
	rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.is_demo`)
	if err != nil {
		return err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, k := range keys {
		_ = storage.Delete(k)
	}
	_, err = pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	return err
}

func createDemo(ctx context.Context, pool *pgxpool.Pool, storage media.Storage, rng *rand.Rand, i int, d demo, hash string, lat, lng float64) (string, error) {
	var id string
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		email := fmt.Sprintf("demo-%d@demo.lueur.invalid", i+1)
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, is_demo, last_active_at)
			VALUES ($1, $2, TRUE, NOW() - make_interval(hours => $3)) RETURNING id
		`, email, hash, rng.Intn(72)).Scan(&id); err != nil {
			return err
		}
		birth := time.Now().UTC().AddDate(-d.age, -rng.Intn(11), -rng.Intn(27))
		// Scatter members within ~15 km, stored coarsened like real locations.
		pLat := math.Round((lat+(rng.Float64()-0.5)*0.25)*100) / 100
		pLng := math.Round((lng+(rng.Float64()-0.5)*0.35)*100) / 100
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, latitude, longitude, location_updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		`, id, d.name, birth, d.gender, d.bio, d.job, d.city, pLat, pLng); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 18, 60, 80)
		`, id, d.interestedIn); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, id, d.interests); err != nil {
			return err
		}
		for pos := 0; pos < 2; pos++ {
			img, err := media.Process(artwork(rng, i*7+pos))
			if err != nil {
				return err
			}
			key, err := media.NewKey()
			if err != nil {
				return err
			}
			if err := storage.Save(key, img.Data); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO photos (user_id, storage_key, width, height, size_bytes, position) VALUES ($1, $2, $3, $4, $5, $6)
			`, id, key, img.Width, img.Height, len(img.Data), pos); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

// artwork renders an abstract placeholder portrait (warm gradient + soft circles):
// obviously not a photo of a real person.
func artwork(rng *rand.Rand, seed int) []byte {
	const w, h = 600, 800
	palettes := [][2]color.RGBA{
		{{232, 96, 76, 255}, {245, 196, 120, 255}},
		{{92, 61, 125, 255}, {232, 140, 120, 255}},
		{{38, 70, 83, 255}, {233, 196, 106, 255}},
		{{180, 80, 110, 255}, {250, 214, 165, 255}},
		{{60, 90, 140, 255}, {240, 170, 150, 255}},
	}
	p := palettes[seed%len(palettes)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	type circle struct{ x, y, r float64 }
	circles := []circle{
		{float64(rng.Intn(w)), float64(rng.Intn(h)), 120 + float64(rng.Intn(140))},
		{float64(rng.Intn(w)), float64(rng.Intn(h)), 80 + float64(rng.Intn(120))},
	}
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			r := float64(p[0].R)*(1-t) + float64(p[1].R)*t
			g := float64(p[0].G)*(1-t) + float64(p[1].G)*t
			b := float64(p[0].B)*(1-t) + float64(p[1].B)*t
			for _, c := range circles {
				if d := math.Hypot(float64(x)-c.x, float64(y)-c.y); d < c.r {
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

func envFloat(key string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64); err == nil {
		return v
	}
	return fallback
}
