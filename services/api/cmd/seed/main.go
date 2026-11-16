// Command seed creates clearly-labelled demo accounts for local testing.
//
//	go run ./cmd/seed            # create demo accounts (idempotent)
//	go run ./cmd/seed -purge     # delete every demo account and its photos
//
// Demo accounts are flagged users.is_demo, use the reserved ".invalid" email
// domain, carry a "Profil de démonstration" bio and are hidden from discovery
// when APP_ENV=production. The command refuses to run in production.
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

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/imaging"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.lueur.invalid"

type demoPerson struct {
	Name, Gender, Looking, Job, City string
	InterestedIn                     []string
	Age                              int
	Lat, Lng                         float64
	Interests                        []string
}

var people = []demoPerson{
	{"Camille", "woman", "relationship", "Architecte", "Paris 11e", []string{"man"}, 29, 48.858, 2.379, []string{"hiking", "art", "coffee", "travel"}},
	{"Léa", "woman", "unsure", "Infirmière", "Paris 18e", []string{"man", "woman"}, 26, 48.892, 2.344, []string{"concerts", "yoga", "cooking"}},
	{"Manon", "woman", "relationship", "Professeure", "Montreuil", []string{"man"}, 33, 48.861, 2.443, []string{"reading", "theatre", "wine", "brunch"}},
	{"Inès", "woman", "casual", "Designer", "Paris 3e", []string{"woman"}, 27, 48.863, 2.360, []string{"photography", "fashion", "cinema"}},
	{"Chloé", "woman", "friendship", "Développeuse", "Boulogne", []string{"man", "woman", "nonbinary"}, 31, 48.835, 2.240, []string{"tech", "gaming", "board_games", "climbing"}},
	{"Sarah", "woman", "relationship", "Médecin", "Paris 15e", []string{"man"}, 35, 48.841, 2.300, []string{"running", "travel", "science"}},
	{"Emma", "woman", "unsure", "Journaliste", "Paris 10e", []string{"man"}, 24, 48.876, 2.360, []string{"writing", "podcasts", "music"}},
	{"Zoé", "woman", "relationship", "Cheffe de projet", "Vincennes", []string{"man"}, 30, 48.847, 2.439, []string{"cycling", "nature", "animals"}},
	{"Lucas", "man", "relationship", "Ingénieur", "Paris 12e", []string{"woman"}, 30, 48.840, 2.388, []string{"hiking", "cooking", "travel", "coffee"}},
	{"Hugo", "man", "unsure", "Musicien", "Paris 20e", []string{"woman"}, 28, 48.864, 2.398, []string{"music", "concerts", "cinema"}},
	{"Nathan", "man", "relationship", "Avocat", "Neuilly", []string{"woman"}, 34, 48.884, 2.269, []string{"wine", "reading", "swimming"}},
	{"Théo", "man", "casual", "Barista", "Paris 9e", []string{"woman", "man"}, 25, 48.877, 2.337, []string{"coffee", "photography", "diy"}},
	{"Karim", "man", "relationship", "Kinésithérapeute", "Saint-Denis", []string{"woman"}, 32, 48.936, 2.357, []string{"football", "fitness", "travel"}},
	{"Antoine", "man", "friendship", "Chercheur", "Paris 5e", []string{"woman"}, 36, 48.846, 2.350, []string{"science", "board_games", "languages"}},
	{"Julien", "man", "relationship", "Comptable", "Issy", []string{"woman"}, 29, 48.824, 2.273, []string{"running", "series", "brunch"}},
	{"Maxime", "man", "unsure", "Photographe", "Paris 2e", []string{"man"}, 27, 48.868, 2.345, []string{"photography", "art", "fashion"}},
	{"Alex", "nonbinary", "unsure", "Illustrateur·ice", "Pantin", []string{"woman", "man", "nonbinary"}, 26, 48.894, 2.409, []string{"art", "gaming", "volunteering"}},
	{"Sacha", "nonbinary", "relationship", "Libraire", "Paris 6e", []string{"woman", "nonbinary"}, 31, 48.850, 2.333, []string{"reading", "writing", "coffee"}},
	{"Noah", "man", "relationship", "Pompier", "Clichy", []string{"woman"}, 28, 48.904, 2.306, []string{"fitness", "swimming", "animals"}},
	{"Louise", "woman", "relationship", "Vétérinaire", "Paris 14e", []string{"man"}, 28, 48.833, 2.326, []string{"animals", "nature", "gardening"}},
}

func main() {
	purge := flag.Bool("purge", false, "delete every demo account")
	flag.Parse()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
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
		password = "DemoPassword1"
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, databaseURL, 30*time.Second, time.Second)
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

	if *purge {
		n, err := purgeDemo(ctx, pool, store)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("deleted %d demo accounts", n)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	created := 0
	for i, p := range people {
		ok, err := seedPerson(ctx, pool, store, i, p, string(hash))
		if err != nil {
			log.Fatalf("seed %s: %v", p.Name, err)
		}
		if ok {
			created++
		}
	}
	log.Printf("demo seed done: %d created, %d already present", created, len(people)-created)
	log.Printf("log in with e.g. %s / %s", email(0, people[0]), password)
}

func email(i int, p demoPerson) string {
	return fmt.Sprintf("%s.%02d@%s", strings.ToLower(asciiFold(p.Name)), i+1, demoDomain)
}

func asciiFold(s string) string {
	r := strings.NewReplacer("é", "e", "è", "e", "ê", "e", "ë", "e", "à", "a", "ç", "c", "ï", "i", "î", "i", "ô", "o", "ö", "o", "ù", "u", "û", "u", "É", "E", "È", "E")
	return r.Replace(s)
}

func seedPerson(ctx context.Context, pool *pgxpool.Pool, store storage.Storage, i int, p demoPerson, hash string) (bool, error) {
	var created bool
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var userID string
		err := tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE)
			ON CONFLICT (email) DO NOTHING RETURNING id
		`, email(i, p), hash).Scan(&userID)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		created = true
		birth := time.Now().AddDate(-p.Age, -(i % 11), -(i % 27))
		bio := "Profil de démonstration — " + demoBio(p)
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, looking_for, city, latitude, longitude, location_updated_at, last_active_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW() - ($11 * INTERVAL '1 hour'))
		`, userID, p.Name, birth, p.Gender, bio, p.Job, p.Looking, p.City, round2(p.Lat), round2(p.Lng), i*3); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
			VALUES ($1, $2, 18, 60, 80)
		`, userID, p.InterestedIn); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, p.Interests); err != nil {
			return err
		}
		for pos := 0; pos < 3; pos++ {
			img, err := imaging.Process(demoImage(i*7+pos, p.Name))
			if err != nil {
				return err
			}
			key := fmt.Sprintf("photos/de/demo%02d-%d.jpg", i, pos)
			if err := store.Put(ctx, key, img.Data); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO photos (user_id, storage_key, width, height, size_bytes, position)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (storage_key) DO NOTHING
			`, userID, key, img.Width, img.Height, len(img.Data), pos); err != nil {
				return err
			}
		}
		return nil
	})
	return created, err
}

func demoBio(p demoPerson) string {
	switch p.Looking {
	case "relationship":
		return "Envie d'une belle histoire, de balades et de longues discussions."
	case "casual":
		return "On commence par un verre et on voit ?"
	case "friendship":
		return "Nouveau dans le coin, partant·e pour des sorties."
	default:
		return "Curieux·se, je me laisse surprendre."
	}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// demoImage renders an abstract warm gradient with soft circles: no real
// person is ever depicted in demo data.
func demoImage(seed int, name string) []byte {
	rng := rand.New(rand.NewSource(int64(seed) + int64(len(name))))
	const w, h = 720, 960
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	palette := [][3]float64{{232, 135, 91}, {176, 74, 92}, {72, 52, 92}, {245, 196, 140}, {120, 92, 150}, {214, 104, 120}}
	a := palette[rng.Intn(len(palette))]
	b := palette[rng.Intn(len(palette))]
	type circle struct{ x, y, r float64 }
	circles := make([]circle, 4)
	for i := range circles {
		circles[i] = circle{rng.Float64() * w, rng.Float64() * h, 120 + rng.Float64()*220}
	}
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			r := a[0]*(1-t) + b[0]*t
			g := a[1]*(1-t) + b[1]*t
			bl := a[2]*(1-t) + b[2]*t
			for _, c := range circles {
				d := math.Hypot(float64(x)-c.x, float64(y)-c.y)
				if d < c.r {
					k := 0.18 * (1 - d/c.r)
					r += (255 - r) * k
					g += (255 - g) * k
					bl += (255 - bl) * k
				}
			}
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(bl), 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Storage) (int, error) {
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
	return int(tag.RowsAffected()), nil
}
