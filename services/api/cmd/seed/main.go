// Command seed inserts clearly-labelled DEMO accounts for local development.
//
//	go run ./cmd/seed -confirm          # insert demo users
//	go run ./cmd/seed -purge            # remove them again
//
// Demo accounts use the reserved domain demo.invalid (RFC 2606): they can never
// receive mail or collide with real users. The command refuses to run when
// APP_ENV=production.
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
	"os"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	demoDomain   = "@demo.invalid"
	demoPassword = "demo-password-1"
)

type demoUser struct {
	slug, name, gender string
	age                int
	interestedIn       []string
	city               string
	lat, lng           float64
	bio                string
	interests          []string
	hue                uint8
}

var demoUsers = []demoUser{
	{"lea", "Léa", "woman", 28, []string{"man"}, "Lyon", 45.76, 4.84, "Grimpeuse du dimanche, cuisinière le reste du temps.", []string{"hiking", "cooking", "music"}, 10},
	{"camille", "Camille", "woman", 31, []string{"man", "woman"}, "Lyon", 45.75, 4.85, "Photographe. Toujours un carnet dans le sac.", []string{"photography", "art", "travel"}, 40},
	{"sarah", "Sarah", "woman", 26, []string{"man"}, "Villeurbanne", 45.77, 4.88, "Fan de ciné et de cafés interminables.", []string{"cinema", "coffee", "reading"}, 70},
	{"manon", "Manon", "woman", 34, []string{"man"}, "Lyon", 45.74, 4.82, "Yoga le matin, vin natural le soir.", []string{"yoga", "wine", "nature"}, 100},
	{"julie", "Julie", "woman", 29, []string{"woman", "man"}, "Paris", 48.86, 2.35, "Danse, voyages et bonne humeur.", []string{"dancing", "travel", "music"}, 130},
	{"nora", "Nora", "nonbinary", 30, []string{"man", "woman", "nonbinary"}, "Lyon", 45.76, 4.83, "Dev le jour, théâtre le soir.", []string{"tech", "theatre", "gaming"}, 160},
	{"thomas", "Thomas", "man", 32, []string{"woman"}, "Lyon", 45.76, 4.86, "Randonnées, vieux vinyles et mauvais jeux de mots.", []string{"hiking", "music", "cooking"}, 190},
	{"hugo", "Hugo", "man", 27, []string{"woman"}, "Lyon", 45.78, 4.80, "Foot le week-end, ciné le dimanche.", []string{"sport", "cinema", "pets"}, 220},
	{"karim", "Karim", "man", 35, []string{"woman", "man"}, "Lyon", 45.75, 4.87, "Cuisinier amateur et voyageur curieux.", []string{"cooking", "travel", "wine"}, 250},
	{"antoine", "Antoine", "man", 29, []string{"woman"}, "Grenoble", 45.19, 5.72, "Montagne, café et bouquins.", []string{"hiking", "coffee", "reading"}, 200},
	{"maxime", "Maxime", "man", 38, []string{"woman"}, "Lyon", 45.76, 4.85, "Papa de deux chats, fan de tech et de randos.", []string{"tech", "pets", "nature"}, 215},
	{"yanis", "Yanis", "man", 25, []string{"woman", "man"}, "Lyon", 45.77, 4.84, "Musicien et bénévole dans une asso locale.", []string{"music", "volunteering", "art"}, 235},
}

func main() {
	confirm := flag.Bool("confirm", false, "insert demo accounts")
	purge := flag.Bool("purge", false, "delete demo accounts")
	flag.Parse()

	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("seed refuses to run when APP_ENV=production")
	}
	if *confirm == *purge {
		log.Fatal("pass exactly one of -confirm or -purge")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	if *purge {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo accounts\n", tag.RowsAffected())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	n := 0
	for _, u := range demoUsers {
		ok, err := insert(ctx, pool, u, string(hash))
		if err != nil {
			log.Fatalf("seed %s: %v", u.slug, err)
		}
		if ok {
			n++
		}
	}
	fmt.Printf("inserted %d demo accounts (password: %s, e.g. %s%s)\n", n, demoPassword, demoUsers[0].slug, demoDomain)
}

func insert(ctx context.Context, pool *pgxpool.Pool, u demoUser, hash string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, $2)
		ON CONFLICT (email) DO NOTHING RETURNING id`, u.slug+demoDomain, hash).Scan(&id)
	if err != nil {
		return false, nil // already seeded
	}
	birth := time.Now().AddDate(-u.age, 0, -10)
	if _, err := tx.Exec(ctx, `INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, id, u.name, birth, u.gender, u.bio, u.city, u.lat, u.lng); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1,$2,18,60,200)`, id, u.interestedIn); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)`, id, u.interests); err != nil {
		return false, err
	}
	for pos := 0; pos < 2; pos++ {
		w, h, data := gradient(u.hue + uint8(pos*25))
		if _, err := tx.Exec(ctx, `INSERT INTO photos (user_id, position, content_type, width, height, data)
			VALUES ($1,$2,'image/jpeg',$3,$4,$5)`, id, pos, w, h, data); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}

// gradient renders a placeholder portrait (no real people's photos in the repo).
func gradient(hue uint8) (int, int, []byte) {
	const w, h = 480, 640
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			r := uint8(float64(60+hue/2)*(1-t) + 240*t)
			g := uint8(float64(40+hue/3)*(1-t) + 150*t)
			b := uint8(float64(120+hue/4)*(1-t) + 120*t)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return w, h, buf.Bytes()
}
