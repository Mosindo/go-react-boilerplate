// Command seed creates clearly-labelled demo accounts for local testing.
//
// Demo accounts use the reserved ".invalid" email domain, are flagged
// users.is_demo = TRUE, and are never shown in discovery when APP_ENV=production.
// The command itself refuses to run in production.
//
//	go run ./cmd/seed                         # create 24 demo profiles around Paris
//	go run ./cmd/seed -likes you@mail.test    # ...and make half of them like that account
//	go run ./cmd/seed -purge                  # delete every demo account and its photos
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

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type demo struct {
	name, gender, lookingFor, job, bio string
	interestedIn                       []string
	age                                int
}

var demos = []demo{
	{"Camille", "woman", "relationship", "Architecte", "Toujours partante pour un brunch et une expo photo.", []string{"man"}, 29},
	{"Léa", "woman", "unsure", "Infirmière", "Randonneuse du dimanche, lectrice du lundi.", []string{"man", "woman"}, 27},
	{"Inès", "woman", "relationship", "Développeuse", "Je code le jour, je danse la salsa le soir.", []string{"man"}, 31},
	{"Manon", "woman", "friendship", "Étudiante en design", "Nouvelle en ville, je cherche des bons plans café.", []string{"woman", "man"}, 22},
	{"Sarah", "woman", "casual", "Cheffe de projet", "Concerts, vin nature et longues balades.", []string{"man"}, 34},
	{"Chloé", "woman", "relationship", "Professeure", "Fan de jeux de société et de cuisine italienne.", []string{"woman"}, 30},
	{"Jade", "woman", "unsure", "Photographe", "Je cherche la lumière parfaite (et quelqu'un pour la partager).", []string{"man", "nonbinary"}, 26},
	{"Nora", "woman", "relationship", "Avocate", "Escalade le samedi, cinéma le dimanche.", []string{"man"}, 36},
	{"Zoé", "woman", "casual", "Barista", "Latte art et playlists indé.", []string{"man", "woman", "nonbinary"}, 24},
	{"Alice", "woman", "relationship", "Vétérinaire", "Mon chat valide toutes mes rencontres.", []string{"man"}, 33},
	{"Lucas", "man", "relationship", "Ingénieur", "Vélo, voyages et bons restos.", []string{"woman"}, 30},
	{"Hugo", "man", "unsure", "Musicien", "Guitariste, je cherche ma prochaine mélodie.", []string{"woman"}, 28},
	{"Nathan", "man", "relationship", "Médecin", "Course à pied le matin, séries le soir.", []string{"woman"}, 35},
	{"Thomas", "man", "casual", "Chef cuisinier", "Je cuisine mieux que je n'écris des bios.", []string{"woman", "man"}, 32},
	{"Louis", "man", "friendship", "Étudiant", "Toujours partant pour un escape game.", []string{"woman"}, 21},
	{"Gabriel", "man", "relationship", "Journaliste", "Curieux de tout, surtout de vous.", []string{"man"}, 29},
	{"Arthur", "man", "unsure", "Graphiste", "Musées, BD et cafés cachés.", []string{"woman", "nonbinary"}, 27},
	{"Jules", "man", "relationship", "Professeur de yoga", "Respirez, souriez, swipez.", []string{"woman"}, 31},
	{"Adam", "man", "casual", "Commercial", "Football, barbecue et rires garantis.", []string{"woman"}, 26},
	{"Paul", "man", "relationship", "Bibliothécaire", "Je vous recommanderai un livre au premier rendez-vous.", []string{"woman", "man"}, 38},
	{"Sacha", "nonbinary", "unsure", "Illustrateur·rice", "Dessin, plantes et thé glacé.", []string{"woman", "man", "nonbinary"}, 25},
	{"Alex", "nonbinary", "relationship", "Chercheur·se", "Sciences, randonnée et jeux vidéo.", []string{"woman", "nonbinary"}, 30},
	{"Charlie", "nonbinary", "friendship", "Comédien·ne", "Théâtre d'impro et karaoké.", []string{"man", "woman", "nonbinary"}, 28},
	{"Eden", "nonbinary", "casual", "DJ", "Vinyles et couchers de soleil.", []string{"woman", "man", "nonbinary"}, 27},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo accounts")
	likes := flag.String("likes", "", "email of an existing account that half of the demo users will like")
	lat := flag.Float64("lat", 48.8566, "latitude of the demo area")
	lng := flag.Float64("lng", 2.3522, "longitude of the demo area")
	password := flag.String("password", "Demo-passw0rd", "password of every demo account (local use only)")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, dsn, 30*time.Second, time.Second)
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

	if *purge {
		n, err := purgeDemo(ctx, pool, storage)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("removed %d demo accounts", n)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	var target string
	if *likes != "" {
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(*likes))).Scan(&target); err != nil {
			log.Fatalf("account %s not found: %v", *likes, err)
		}
	}

	rng := rand.New(rand.NewPCG(42, 7))
	created := 0
	for i, d := range demos {
		email := fmt.Sprintf("demo-%02d@demo.invalid", i+1)
		id, isNew, err := seedOne(ctx, pool, storage, rng, d, email, string(hash), *lat, *lng, i)
		if err != nil {
			log.Fatalf("seed %s: %v", email, err)
		}
		if isNew {
			created++
		}
		if target != "" && i%2 == 0 && id != target {
			if _, err := pool.Exec(ctx, `
				INSERT INTO swipes (swiper_id, target_id, decision) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING
			`, id, target); err != nil {
				log.Fatal(err)
			}
		}
	}
	log.Printf("demo data ready: %d new accounts (%d total). Sign in with demo-01@demo.invalid / %s", created, len(demos), *password)
}

func seedOne(ctx context.Context, pool *pgxpool.Pool, storage media.Storage, rng *rand.Rand, d demo, email, hash string, lat, lng float64, index int) (string, bool, error) {
	var id string
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != pgx.ErrNoRows {
		return "", false, err
	}

	photos := make([][]byte, 0, 2)
	for p := 0; p < 2; p++ {
		processed, err := media.Process(portrait(index, p))
		if err != nil {
			return "", false, err
		}
		photos = append(photos, processed.Data)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE) RETURNING id
	`, email, hash).Scan(&id); err != nil {
		return "", false, err
	}
	birthdate := time.Now().AddDate(-d.age, -rng.IntN(11), -rng.IntN(27))
	// Spread profiles within ~15 km of the centre.
	angle := rng.Float64() * 2 * math.Pi
	dist := 1 + rng.Float64()*14
	pLat := geo.Round(lat + dist/111*math.Cos(angle))
	pLng := geo.Round(lng + dist/(111*math.Cos(lat*math.Pi/180))*math.Sin(angle))
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, looking_for, city, latitude, longitude, location_updated_at, last_active_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'Paris', $8, $9, NOW(), NOW() - make_interval(hours => $10))
	`, id, d.name, birthdate, d.gender, d.bio+" (profil de démonstration)", d.job, d.lookingFor, pLat, pLng, rng.IntN(72)); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km) VALUES ($1, $2, 18, 60, 50)
	`, id, d.interestedIn); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests ORDER BY md5(id::text || $2) LIMIT 5
	`, id, id); err != nil {
		return "", false, err
	}
	keys := make([]string, 0, len(photos))
	for p, data := range photos {
		key, err := storage.Save(data)
		if err != nil {
			return "", false, err
		}
		keys = append(keys, key)
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1, $2, $3, 720, 960, $4)
		`, id, key, p, len(data)); err != nil {
			return "", false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		for _, k := range keys {
			_ = storage.Delete(k)
		}
		return "", false, err
	}
	return id, true, nil
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, storage media.Storage) (int64, error) {
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
		_ = storage.Delete(k)
	}
	return tag.RowsAffected(), nil
}

// portrait renders an abstract placeholder picture: a soft gradient with a
// silhouette-like shape, so demo profiles are obviously not real people.
func portrait(index, variant int) []byte {
	const w, h = 720, 960
	hue := float64((index*37 + variant*90) % 360)
	top := hsl(hue, 0.55, 0.62)
	bottom := hsl(math.Mod(hue+40, 360), 0.6, 0.32)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		row := color.RGBA{
			uint8(float64(top.R)*(1-t) + float64(bottom.R)*t),
			uint8(float64(top.G)*(1-t) + float64(bottom.G)*t),
			uint8(float64(top.B)*(1-t) + float64(bottom.B)*t),
			255,
		}
		for x := 0; x < w; x++ {
			img.Set(x, y, row)
		}
	}
	shape := color.RGBA{255, 255, 255, 60}
	blend := func(x, y int) {
		c := img.RGBAAt(x, y)
		a := float64(shape.A) / 255
		img.Set(x, y, color.RGBA{
			uint8(float64(c.R)*(1-a) + float64(shape.R)*a),
			uint8(float64(c.G)*(1-a) + float64(shape.G)*a),
			uint8(float64(c.B)*(1-a) + float64(shape.B)*a),
			255,
		})
	}
	// Head and shoulders.
	cx, cy, r := w/2, h*2/5, 130
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := x-cx, y-cy
			inHead := dx*dx+dy*dy < r*r
			sy := y - (h*4/5 + 40)
			inBody := y > h*3/5 && float64(dx*dx)/float64(260*260)+float64(sy*sy)/float64(230*230) < 1
			if inHead || inBody {
				blend(x, y)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}
