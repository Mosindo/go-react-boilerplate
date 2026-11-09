// Command seed inserts clearly-labelled demo accounts for local development.
//
// Every demo account uses an @demo.invalid address (a reserved TLD that can never be a real
// mailbox), so demo data cannot be confused with real users. The command refuses to run when
// APP_ENV=production. Re-running it is safe: existing demo accounts are skipped.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"strings"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"golang.org/x/crypto/bcrypt"
)

type demo struct {
	name, gender, birth, city string
	lat, lon                  float64
	seeking                   []string
	bio                       string
	interests                 []string
	hue                       int
}

var people = []demo{
	{"Camille", "woman", "1996-04-12", "Paris", 48.86, 2.35, []string{"man"}, "Fan de cinéma d'auteur et de randonnées le dimanche.", []string{"cinema", "hiking", "reading"}, 10},
	{"Lucas", "man", "1994-09-30", "Paris", 48.85, 2.34, []string{"woman"}, "Cuisine, vélo et longues discussions.", []string{"cooking", "sports", "music"}, 60},
	{"Inès", "woman", "1998-01-21", "Paris", 48.87, 2.30, []string{"man", "woman"}, "Danse, voyages et café noir.", []string{"dancing", "travel", "art"}, 110},
	{"Hugo", "man", "1991-06-05", "Paris", 48.83, 2.37, []string{"woman"}, "Photographe amateur, toujours un appareil sur moi.", []string{"photography", "travel", "nature"}, 160},
	{"Sofia", "woman", "1993-11-17", "Lyon", 45.76, 4.84, []string{"man"}, "Yoga le matin, théâtre le soir.", []string{"yoga", "theatre", "wine"}, 210},
	{"Noah", "man", "1997-03-08", "Lyon", 45.75, 4.85, []string{"woman"}, "Développeur le jour, musicien la nuit.", []string{"tech", "music", "gaming"}, 260},
	{"Léa", "woman", "1995-07-25", "Lyon", 45.77, 4.83, []string{"man"}, "Je cherche quelqu'un pour explorer les marchés.", []string{"cooking", "wine", "pets"}, 310},
	{"Adam", "man", "1990-12-02", "Lyon", 45.74, 4.87, []string{"woman", "man"}, "Trail, montagne et bonne bière.", []string{"hiking", "fitness", "nature"}, 30},
	{"Jade", "woman", "1999-05-14", "Marseille", 43.30, 5.37, []string{"man"}, "Plongée, soleil et musique live.", []string{"music", "festivals", "sports"}, 80},
	{"Eliott", "man", "1992-08-19", "Marseille", 43.29, 5.38, []string{"woman"}, "Passionné de langues et de voyages.", []string{"languages", "travel", "reading"}, 130},
	{"Zoé", "woman", "1994-02-28", "Paris", 48.88, 2.33, []string{"nonbinary", "woman"}, "Illustratrice, chat à la maison.", []string{"art", "pets", "cinema"}, 180},
	{"Sacha", "nonbinary", "1996-10-10", "Paris", 48.84, 2.36, []string{"woman", "man", "nonbinary"}, "Théâtre, festivals et brunchs.", []string{"theatre", "festivals", "wine"}, 230},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "demo-password-123"
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
	storage, err := photos.NewDiskStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	created := 0
	for _, p := range people {
		email := strings.NewReplacer("é", "e", "è", "e", "ï", "i", "ë", "e", "ö", "o").Replace(strings.ToLower(p.name)) + "@demo.invalid"
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
			log.Fatal(err)
		}
		if exists {
			continue
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Fatal(err)
		}
		var id string
		must(tx.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id::text`, email, string(hash)).Scan(&id))
		_, err = tx.Exec(ctx, `INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, p.name, p.birth, p.gender, p.bio, p.city, p.lat, p.lon)
		must(err)
		_, err = tx.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 21, 45, 100)`, id, p.seeking)
		must(err)
		_, err = tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)`, id, p.interests)
		must(err)
		for pos := 0; pos < 2; pos++ {
			data, w, h := gradient(p.hue + pos*40)
			key, err := photos.NewKey()
			must(err)
			must(storage.Save(key, data))
			_, err = tx.Exec(ctx, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1, $2, $3, $4, $5, $6)`,
				id, key, pos, w, h, len(data))
			must(err)
		}
		must(tx.Commit(ctx))
		created++
	}
	fmt.Printf("seed done: %d demo accounts created (%d total defined)\n", created, len(people))
	fmt.Printf("log in with e.g. camille@demo.invalid / %s\n", password)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// gradient renders a simple portrait-format gradient used as a placeholder demo photo.
func gradient(hue int) ([]byte, int, int) {
	const w, h = 480, 640
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			img.Set(x, y, color.RGBA{
				R: uint8(float64(hue%256)*(1-t) + 240*t),
				G: uint8(120 + 80*float64(x)/w),
				B: uint8(255 - float64(hue%256)*(1-t)/2),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes(), w, h
}
