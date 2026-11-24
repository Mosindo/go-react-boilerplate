// Command seed creates demonstration profiles for local development.
//
// Demo accounts are flagged users.is_demo = TRUE, use the reserved
// demo.lueur.invalid email domain, carry a "Profil de démonstration" bio and
// generated abstract pictures (no real faces). Production discovery never
// shows them, and the command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed                      # create/refresh demo profiles
//	go run ./cmd/seed -like you@mail.com   # demo profiles already like you
//	go run ./cmd/seed -purge               # remove all demo data
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
	"os"
	"strings"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type demoProfile struct {
	Name, Gender, Goal, Job, Bio string
	Age                          int
	InterestedIn                 []string
	Interests                    []string
	Hue                          float64
}

var demoProfiles = []demoProfile{
	{"Camille", "woman", "long_term", "Architecte", "Brunchs, expos et longues balades.", 29, []string{"man", "woman"}, []string{"museums", "coffee", "hiking"}, 12},
	{"Louis", "man", "long_term", "Kiné", "Grimpeur du dimanche, cuisinier du lundi.", 31, []string{"woman"}, []string{"climbing", "cooking", "travel"}, 200},
	{"Inès", "woman", "unsure", "Doctorante en biologie", "Je collectionne les plantes et les mauvais jeux de mots.", 27, []string{"man"}, []string{"science", "gardening", "board_games"}, 330},
	{"Hugo", "man", "short_term", "Développeur", "Synthés, vélo et ramen.", 26, []string{"woman"}, []string{"electro", "cycling", "tech"}, 160},
	{"Sacha", "nonbinary", "friendship", "Illustrateur·rice", "Carnet de croquis toujours dans la poche.", 30, []string{"woman", "man", "nonbinary"}, []string{"painting", "cinema", "coffee"}, 280},
	{"Chloé", "woman", "long_term", "Professeure des écoles", "Fan de jazz et de jeux de société.", 34, []string{"man"}, []string{"jazz", "board_games", "reading"}, 40},
	{"Mehdi", "man", "long_term", "Chef de projet", "Footing au lever du soleil, cinéma le soir.", 33, []string{"woman"}, []string{"running", "cinema", "travel"}, 220},
	{"Léa", "woman", "short_term", "Photographe", "Toujours partante pour un concert improvisé.", 25, []string{"woman"}, []string{"photography", "concerts", "dance"}, 300},
	{"Nina", "woman", "unsure", "Infirmière", "Yoga, podcasts et thé glacé.", 28, []string{"man", "woman"}, []string{"yoga", "podcasts", "meditation"}, 90},
	{"Arthur", "man", "friendship", "Libraire", "Je vous recommanderai un livre, promis.", 37, []string{"woman", "man"}, []string{"reading", "writing", "wine"}, 260},
	{"Zoé", "woman", "long_term", "Ingénieure", "Rando le samedi, plaid le dimanche.", 32, []string{"man"}, []string{"hiking", "video_games", "cooking"}, 140},
	{"Noah", "man", "unsure", "Musicien", "Guitare, voyages et café serré.", 29, []string{"woman", "man"}, []string{"instrument", "travel", "coffee"}, 20},
}

func main() {
	like := flag.String("like", "", "email of an existing account that every demo profile will already like")
	purge := flag.Bool("purge", false, "remove demo accounts and exit")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoLueur2026"
	}
	centerLat, centerLng := envFloat("SEED_LAT", 48.8566), envFloat("SEED_LNG", 2.3522)

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, databaseURL, 20*time.Second, time.Second)
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
		log.Println("demo data removed")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatal(err)
	}
	ids := make([]string, 0, len(demoProfiles))
	for i, p := range demoProfiles {
		id, err := createDemo(ctx, pool, store, i, p, string(hash), centerLat, centerLng)
		if err != nil {
			log.Fatalf("seed %s: %v", p.Name, err)
		}
		ids = append(ids, id)
	}

	if *like != "" {
		var targetID string
		err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(*like))).Scan(&targetID)
		if err != nil {
			log.Fatalf("account %s not found: %v", *like, err)
		}
		for _, id := range ids {
			if _, err := pool.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, decision) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, id, targetID); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("all demo profiles now like %s: like them back to get a match", *like)
	}

	log.Printf("seeded %d demo profiles around %.4f,%.4f", len(ids), centerLat, centerLng)
	log.Printf("demo logins: demo1@demo.lueur.invalid … demo%d@demo.lueur.invalid / password %q", len(ids), password)
}

func envFloat(key string, fallback float64) float64 {
	var v float64
	if _, err := fmt.Sscanf(os.Getenv(key), "%f", &v); err != nil {
		return fallback
	}
	return v
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store) error {
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
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`); err != nil {
		return err
	}
	for _, k := range keys {
		_ = store.Delete(ctx, k)
	}
	return nil
}

func createDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store, index int, p demoProfile, hash string, lat, lng float64) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	email := fmt.Sprintf("demo%d@demo.lueur.invalid", index+1)
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_demo, last_active_at)
		VALUES ($1, $2, TRUE, NOW() - make_interval(hours => $3))
		RETURNING id
	`, email, hash, index*5).Scan(&id); err != nil {
		return "", err
	}
	angle := float64(index) * 2 * math.Pi / float64(len(demoProfiles))
	radius := 0.02 + 0.01*float64(index%4)
	birthdate := time.Now().AddDate(-p.Age, -index%11, -index*2)
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, relationship_goal, city, latitude, longitude, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'Démo', $8, $9, NOW())
	`, id, p.Name, birthdate, p.Gender, "Profil de démonstration — "+p.Bio, p.Job, p.Goal,
		math.Round((lat+radius*math.Sin(angle))*100)/100, math.Round((lng+radius*math.Cos(angle))*100)/100); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 18, 60, 100)`, id, p.InterestedIn); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, id, p.Interests); err != nil {
		return "", err
	}
	for position := 0; position < 2; position++ {
		processed, err := photos.ProcessImage(abstractPortrait(p.Hue+float64(position)*35, index+position))
		if err != nil {
			return "", err
		}
		key, err := storage.NewKey("photos/"+id, ".jpg")
		if err != nil {
			return "", err
		}
		if err := store.Put(ctx, key, processed.Data); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, position, storage_key, mime_type, width, height, size_bytes)
			VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		`, id, position, key, processed.Width, processed.Height, len(processed.Data)); err != nil {
			return "", err
		}
	}
	return id, tx.Commit(ctx)
}

// abstractPortrait renders a soft gradient with circles: clearly synthetic.
func abstractPortrait(hue float64, seed int) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, bottom := hsl(hue, 0.55, 0.62), hsl(hue+40, 0.5, 0.28)
	cx := []float64{float64(w) * 0.3, float64(w) * 0.72, float64(w) * 0.5}
	cy := []float64{float64(h) * 0.3, float64(h) * 0.55, float64(h) * 0.8}
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			r := float64(top.R)*(1-t) + float64(bottom.R)*t
			g := float64(top.G)*(1-t) + float64(bottom.G)*t
			b := float64(top.B)*(1-t) + float64(bottom.B)*t
			for i := range cx {
				d := math.Hypot(float64(x)-cx[i], float64(y)-cy[(i+seed)%3])
				if d < 90+float64(i*30) {
					r, g, b = r*0.85+255*0.15, g*0.85+240*0.15, b*0.85+220*0.15
				}
			}
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360) / 360
	hue := func(p, q, t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	return color.RGBA{uint8(hue(p, q, h+1.0/3) * 255), uint8(hue(p, q, h) * 255), uint8(hue(p, q, h-1.0/3) * 255), 255}
}
