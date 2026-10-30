// Command seed fills a development database with clearly labelled demo users.
//
// Demo accounts use the reserved ".invalid" email domain and are flagged
// users.is_demo, so they can never be mistaken for (or collide with) real
// people. The command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed          create demo users (idempotent)
//	go run ./cmd/seed -purge   delete every demo user and its photo files
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.invalid"

type demoUser struct {
	name      string
	gender    string
	birthYear int
	into      []string
	bio       string
	interests []string
	hue       float64
}

var demoUsers = []demoUser{
	{"Camille", "woman", 1994, []string{"man"}, "Passionnée de cuisine et de randonnées du dimanche.", []string{"cuisine", "randonnee", "voyage"}, 12},
	{"Léa", "woman", 1991, []string{"man", "woman"}, "Toujours un livre dans le sac. Café noir, sans sucre.", []string{"lecture", "cafe", "cinema"}, 330},
	{"Inès", "woman", 1997, []string{"man"}, "Danseuse le soir, ingénieure le jour.", []string{"danse", "technologie", "musique"}, 280},
	{"Manon", "woman", 1989, []string{"woman", "man"}, "Photographe amatrice, collectionneuse de vinyles.", []string{"photographie", "musique", "art"}, 200},
	{"Sarah", "woman", 1993, []string{"man"}, "Je cours des semi-marathons et je mange des croissants.", []string{"running", "cuisine", "plage"}, 30},
	{"Alex", "non_binary", 1995, []string{"man", "woman", "non_binary"}, "Jeux de société, thé, et discussions sans fin.", []string{"jeux-de-societe", "cafe", "ecriture"}, 160},
	{"Thomas", "man", 1990, []string{"woman"}, "Cycliste, cuisinier, mauvais joueur de pétanque.", []string{"velo", "cuisine", "vin"}, 220},
	{"Hugo", "man", 1992, []string{"woman"}, "Ingénieur son, fan de concerts et de montagne.", []string{"concerts", "montagne", "musique"}, 100},
	{"Nicolas", "man", 1987, []string{"woman", "man"}, "Je voyage dès que je peux, et je cuisine ce que je rapporte.", []string{"voyage", "cuisine", "photographie"}, 40},
	{"Julien", "man", 1995, []string{"woman"}, "Bénévole dans une asso de quartier, fan de cinéma.", []string{"benevolat", "cinema", "lecture"}, 250},
	{"Maxime", "man", 1993, []string{"woman"}, "Yoga le matin, jeux vidéo le soir.", []string{"yoga", "jeux-video", "technologie"}, 300},
	{"Sam", "non_binary", 1996, []string{"woman", "non_binary"}, "Théâtre, bricolage et plantes en pot.", []string{"theatre", "bricolage", "jardinage"}, 80},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo users instead of creating them")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("seed refuses to run with APP_ENV=production")
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
	pool, err := db.ConnectWithRetry(ctx, databaseURL, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.NewDisk(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		if err := purgeDemo(ctx, pool, store); err != nil {
			log.Fatal(err)
		}
		return
	}

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoPass1234!"
	}
	lat, lon := envFloat("SEED_LAT", 48.8566), envFloat("SEED_LON", 2.3522)
	created, err := seed(ctx, pool, store, password, lat, lon)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d demo users created (%d total available).\n", created, len(demoUsers))
	fmt.Printf("Sign in with demo.<firstname, lowercase, no accents>@%s and the password %q.\n", demoDomain, password)
}

func envFloat(key string, fallback float64) float64 {
	var v float64
	if _, err := fmt.Sscanf(os.Getenv(key), "%f", &v); err != nil {
		return fallback
	}
	return v
}

func demoEmail(name string) string {
	r := strings.NewReplacer("é", "e", "è", "e", "ï", "i", "ë", "e", "à", "a", "ç", "c")
	return "demo." + r.Replace(strings.ToLower(name)) + "@" + demoDomain
}

func seed(ctx context.Context, pool *pgxpool.Pool, store *storage.Disk, password string, lat, lon float64) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	created := 0
	for i, u := range demoUsers {
		var userID string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE)
			ON CONFLICT (email) DO NOTHING RETURNING id::text`, demoEmail(u.name), string(hash)).Scan(&userID)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				continue
			}
			return created, err
		}

		angle := float64(i) / float64(len(demoUsers)) * 2 * math.Pi
		uLat := math.Round((lat+0.08*math.Sin(angle))*100) / 100
		uLon := math.Round((lon+0.12*math.Cos(angle))*100) / 100
		birth := time.Date(u.birthYear, time.Month(1+i%12), 1+i%27, 0, 0, 0, 0, time.UTC)

		if _, err := pool.Exec(ctx, `
			INSERT INTO profiles (user_id, display_name, birth_date, gender, bio, city, latitude, longitude)
			VALUES ($1, $2, $3, $4, $5, 'Démo', $6, $7)`, userID, u.name, birth, u.gender, u.bio, uLat, uLon); err != nil {
			return created, err
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 21, 55, 100)`, userID, u.into); err != nil {
			return created, err
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)`, userID, u.interests); err != nil {
			return created, err
		}

		data, w, h, err := gradientJPEG(u.hue)
		if err != nil {
			return created, err
		}
		random := make([]byte, 17)
		if _, err := rand.Read(random); err != nil {
			return created, err
		}
		key := storage.NewKey(random)
		if err := store.Put(key, data); err != nil {
			return created, err
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, mime_type, width, height, size_bytes, position)
			VALUES ($1, $2, 'image/jpeg', $3, $4, $5, 0)`, userID, key, w, h, len(data)); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, store *storage.Disk) error {
	rows, err := pool.Query(ctx, `SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.is_demo`)
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
		if err := store.Delete(k); err != nil {
			log.Printf("could not delete %s: %v", k, err)
		}
	}
	fmt.Printf("%d demo users deleted.\n", tag.RowsAffected())
	return nil
}

// gradientJPEG draws a soft two-tone portrait-ratio placeholder so demo data
// needs no external image assets.
func gradientJPEG(hue float64) ([]byte, int, int, error) {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, bottom := hsl(hue, 0.65, 0.62), hsl(math.Mod(hue+40, 360), 0.7, 0.32)
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		c := color.RGBA{
			uint8(float64(top.R)*(1-t) + float64(bottom.R)*t),
			uint8(float64(top.G)*(1-t) + float64(bottom.G)*t),
			uint8(float64(top.B)*(1-t) + float64(bottom.B)*t),
			255,
		}
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
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
