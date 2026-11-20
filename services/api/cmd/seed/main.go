// Command seed creates demonstration accounts for local testing.
//
// Demo accounts are flagged with users.is_demo (shown as "Profil démo" in the
// app), use the reserved .invalid e-mail domain, and the command refuses to run
// when APP_ENV=production. Re-running it replaces the previous demo accounts.
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
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.lueur.invalid"

type persona struct {
	name, gender, bio, job string
	interestedIn           []string
	age                    int
}

var personas = []persona{
	{"Camille", "woman", "Toujours partante pour une expo puis un verre en terrasse.", "Architecte", []string{"man"}, 29},
	{"Léa", "woman", "Randos le week-end, romans policiers le soir.", "Infirmière", []string{"man", "woman"}, 27},
	{"Inès", "woman", "Je cuisine mieux que je ne chante, mais je chante quand même.", "Cheffe de projet", []string{"man"}, 32},
	{"Manon", "woman", "Yoga au lever du soleil, jazz au coucher.", "Professeure", []string{"man"}, 34},
	{"Chloé", "woman", "Amoureuse des chats, des vieux films et du café filtre.", "Graphiste", []string{"woman"}, 26},
	{"Sarah", "woman", "On part en road trip ?", "Ingénieure", []string{"man"}, 31},
	{"Julie", "woman", "Bénévole dans une asso, fan de jeux de société.", "Juriste", []string{"man"}, 38},
	{"Nora", "woman", "Escalade, podcasts et brunchs interminables.", "Développeuse", []string{"man", "nonbinary"}, 30},
	{"Zoé", "woman", "Photographe amatrice, je cherche des coins secrets de la ville.", "Pharmacienne", []string{"man"}, 25},
	{"Alice", "woman", "Danse, voyages et longues conversations.", "Journaliste", []string{"man"}, 35},
	{"Thomas", "man", "Cycliste du dimanche et cuisinier du samedi.", "Kiné", []string{"woman"}, 31},
	{"Hugo", "man", "Concerts, vinyles et balades au bord de l'eau.", "Ingénieur son", []string{"woman"}, 28},
	{"Lucas", "man", "Je cherche quelqu'un pour tester tous les restos du quartier.", "Comptable", []string{"woman"}, 33},
	{"Nathan", "man", "Surf l'été, ski l'hiver, lecture toute l'année.", "Médecin", []string{"woman"}, 36},
	{"Karim", "man", "Grand fan de cinéma coréen et de randonnée.", "Enseignant", []string{"woman"}, 30},
	{"Antoine", "man", "Entrepreneur curieux, toujours un livre dans le sac.", "Fondateur", []string{"woman", "man"}, 40},
	{"Maxime", "man", "Chien adorable fourni avec le profil.", "Paysagiste", []string{"woman"}, 29},
	{"Yanis", "man", "Jeux vidéo, basket et pâtisserie (oui oui).", "Data analyst", []string{"woman"}, 27},
	{"Paul", "man", "Musées le matin, marché l'après-midi.", "Historien", []string{"woman"}, 45},
	{"Louis", "man", "J'apprends le japonais, on révise ensemble ?", "Traducteur", []string{"man"}, 32},
	{"Sacha", "nonbinary", "Art, militantisme joyeux et thé glacé.", "Illustrateur·rice", []string{"woman", "man", "nonbinary"}, 28},
	{"Alex", "nonbinary", "Théâtre d'impro et cuisine végétarienne.", "Comédien·ne", []string{"woman", "nonbinary"}, 31},
}

func main() {
	lat := flag.Float64("lat", 48.8566, "latitude around which demo members are placed")
	lng := flag.Float64("lng", 2.3522, "longitude around which demo members are placed")
	password := flag.String("password", envOr("SEED_DEMO_PASSWORD", "Demo12345"), "password of every demo account")
	likesFor := flag.String("likes-for", "", "e-mail of an existing account that half of the demo members will like (instant matches)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 30*time.Second, time.Second)
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

	if err := removeDemoAccounts(ctx, pool, store); err != nil {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	interestIDs, err := loadInterestIDs(ctx, pool)
	if err != nil {
		log.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(42, 7))
	created := make([]string, 0, len(personas))
	for i, p := range personas {
		id, err := createDemo(ctx, pool, store, rng, i, p, string(hash), interestIDs, *lat, *lng)
		if err != nil {
			log.Fatalf("create %s: %v", p.name, err)
		}
		created = append(created, id)
	}

	if *likesFor != "" {
		n, err := likeTarget(ctx, pool, strings.ToLower(strings.TrimSpace(*likesFor)), created)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d demo members now like %s: like them back to get instant matches.\n", n, *likesFor)
	}

	fmt.Printf("Created %d demo accounts (demo01@%s ... password %q).\n", len(created), demoDomain, *password)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func removeDemoAccounts(ctx context.Context, pool *pgxpool.Pool, store storage.Store) error {
	rows, err := pool.Query(ctx, `SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.is_demo`)
	if err != nil {
		return err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`); err != nil {
		return err
	}
	for _, key := range keys {
		_ = store.Delete(ctx, key)
	}
	return nil
}

func loadInterestIDs(ctx context.Context, pool *pgxpool.Pool) ([]int16, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM interests ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int16])
}

func createDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store, rng *rand.Rand, index int, p persona,
	hash string, interestIDs []int16, centerLat, centerLng float64) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	email := fmt.Sprintf("demo%02d@%s", index+1, demoDomain)
	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_demo, last_active_at)
		VALUES ($1, $2, TRUE, NOW() - make_interval(hours => $3))
		RETURNING id
	`, email, hash, rng.IntN(72)).Scan(&userID); err != nil {
		return "", err
	}

	birthdate := time.Now().AddDate(-p.age, -rng.IntN(11), -rng.IntN(27))
	// Spread members up to ~25 km around the center, snapped like real data.
	distance := rng.Float64() * 25
	bearing := rng.Float64() * 2 * math.Pi
	lat := centerLat + distance/111*math.Cos(bearing)
	lng := centerLng + distance/(111*math.Cos(centerLat*math.Pi/180))*math.Sin(bearing)
	height := 155 + rng.IntN(40)
	goals := []string{"long_term", "short_term", "friendship", "undecided"}
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, height_cm, relationship_goal,
		                      location_lat, location_lng, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'Paris', $7, $8, round($9::numeric, 2), round($10::numeric, 2), NOW())
	`, userID, p.name, birthdate, p.gender, p.bio, p.job, height, goals[rng.IntN(len(goals))], lat, lng); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km) VALUES ($1, $2, 18, 60, 100)
	`, userID, p.interestedIn); err != nil {
		return "", err
	}
	picked := map[int16]bool{}
	for len(picked) < 4+rng.IntN(4) && len(interestIDs) > 0 {
		picked[interestIDs[rng.IntN(len(interestIDs))]] = true
	}
	for id := range picked {
		if _, err := tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) VALUES ($1, $2)`, userID, id); err != nil {
			return "", err
		}
	}
	for position := 0; position < 2+rng.IntN(2); position++ {
		data, err := demoImage(rng)
		if err != nil {
			return "", err
		}
		key, err := storage.NewKey("jpg")
		if err != nil {
			return "", err
		}
		if err := store.Put(ctx, key, bytes.NewReader(data)); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, position, storage_key, mime_type, width, height, size_bytes)
			VALUES ($1, $2, $3, 'image/jpeg', 900, 1200, $4)
		`, userID, position, key, len(data)); err != nil {
			return "", err
		}
	}
	return userID, tx.Commit(ctx)
}

// demoImage paints an abstract warm gradient: demo profiles deliberately use
// no real faces so they cannot be mistaken for real people.
func demoImage(rng *rand.Rand) ([]byte, error) {
	const w, h = 900, 1200
	palette := [][2]color.RGBA{
		{{0xE8, 0x77, 0x5A, 255}, {0x3B, 0x1F, 0x2B, 255}},
		{{0xF2, 0xB8, 0x8C, 255}, {0x6B, 0x2D, 0x3E, 255}},
		{{0xD9, 0x8F, 0x6B, 255}, {0x24, 0x1A, 0x2E, 255}},
		{{0xF5, 0xD0, 0xA9, 255}, {0x8A, 0x3B, 0x4A, 255}},
	}
	pair := palette[rng.IntN(len(palette))]
	cx, cy := float64(rng.IntN(w)), float64(rng.IntN(h/2))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / float64(h)
			t := math.Min(1, d*1.4)
			img.Set(x, y, color.RGBA{
				R: uint8(float64(pair[0].R)*(1-t) + float64(pair[1].R)*t),
				G: uint8(float64(pair[0].G)*(1-t) + float64(pair[1].G)*t),
				B: uint8(float64(pair[0].B)*(1-t) + float64(pair[1].B)*t),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes(), err
}

func likeTarget(ctx context.Context, pool *pgxpool.Pool, email string, demoIDs []string) (int, error) {
	var targetID string
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND NOT is_demo`, email).Scan(&targetID); err != nil {
		return 0, fmt.Errorf("account %s not found: %w", email, err)
	}
	count := 0
	for i, id := range demoIDs {
		if i%2 != 0 {
			continue
		}
		tag, err := pool.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING
		`, id, targetID)
		if err != nil {
			return count, err
		}
		count += int(tag.RowsAffected())
	}
	return count, nil
}
