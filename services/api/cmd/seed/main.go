// Command seed loads clearly-labelled demo accounts for local development.
//
// Safety rails: it refuses to run when APP_ENV=production, every account uses the reserved
// ".invalid" domain (never deliverable, never a real person) and is flagged users.is_demo=true so it
// can be wiped with `go run ./cmd/seed -purge`. Demo photos are generated gradients, not people.
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
	"image/jpeg"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"golang.org/x/crypto/bcrypt"
)

const demoPassword = "demo-password-1"

type demoUser struct {
	name, gender string
	birth        string
	interested   []string
	city         string
	lat, lng     float64
	bio          string
	interests    []string
	hue          float64
}

var users = []demoUser{
	{"Camille", "woman", "1995-03-14", []string{"man", "woman"}, "Paris", 48.8566, 2.3522, "Passionnée de cuisine et de randonnées du dimanche.", []string{"cooking", "hiking", "music"}, 0.02},
	{"Louis", "man", "1992-07-02", []string{"woman"}, "Paris", 48.8606, 2.3376, "Photographe amateur, toujours partant pour un café.", []string{"photography", "coffee", "travel"}, 0.58},
	{"Inès", "woman", "1998-11-23", []string{"man"}, "Montreuil", 48.8634, 2.4484, "Danse, cinéma d'auteur et jeux de société.", []string{"dancing", "cinema", "boardgames"}, 0.85},
	{"Hugo", "man", "1990-01-30", []string{"woman"}, "Versailles", 48.8014, 2.1301, "Cycliste le week-end, cuisinier le reste du temps.", []string{"cycling", "cooking", "wine"}, 0.33},
	{"Sacha", "non_binary", "1996-09-09", []string{"man", "woman", "non_binary"}, "Paris", 48.8467, 2.3499, "Lectrice compulsive, yoga le matin.", []string{"reading", "yoga", "art"}, 0.72},
	{"Manon", "woman", "1994-05-05", []string{"man"}, "Saint-Denis", 48.9362, 2.3574, "Festivals l'été, théâtre l'hiver.", []string{"festivals", "theatre", "music"}, 0.12},
	{"Jules", "man", "1997-12-12", []string{"woman", "man"}, "Boulogne-Billancourt", 48.8397, 2.2399, "Dev le jour, joueur de société la nuit.", []string{"tech", "gaming", "boardgames"}, 0.48},
	{"Clara", "woman", "1991-02-18", []string{"man"}, "Lyon", 45.7640, 4.8357, "Course à pied et bons vins.", []string{"running", "wine", "travel"}, 0.95},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo accounts and their data")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 15*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	// Always start clean so the command is idempotent.
	rows, err := pool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id IN (SELECT id FROM users WHERE is_demo)`)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		_ = store.Delete(ctx, k)
	}
	rows.Close()
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`); err != nil {
		log.Fatal(err)
	}
	if *purge {
		fmt.Println("demo accounts removed")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	for i, u := range users {
		email := strings.NewReplacer("è", "e", "é", "e", "ï", "i").Replace(strings.ToLower(u.name)) + "@demo.invalid"
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE) RETURNING id`, email, string(hash)).Scan(&id); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
			VALUES ($1, $2, $3, $4, $5, $6, round($7::numeric, 2), round($8::numeric, 2))`,
			id, u.name, u.birth, u.gender, u.bio, u.city, u.lat, u.lng); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km) VALUES ($1, $2, 21, 45, 100)`, id, u.interested); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO user_interests SELECT $1, id FROM interests WHERE slug = ANY($2::text[])`, id, u.interests); err != nil {
			log.Fatal(err)
		}
		for pos := 0; pos < 2; pos++ {
			key := randomKey()
			data := gradient(u.hue+float64(pos)*0.08, 900, 1200)
			if err := store.Put(ctx, key, data); err != nil {
				log.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1, $2, $3, 900, 1200, $4)`, id, key, pos, len(data)); err != nil {
				log.Fatal(err)
			}
		}
		fmt.Printf("%d. %-8s %s\n", i+1, u.name, email)
	}
	fmt.Printf("\nDemo accounts created. Password for all: %s\n", demoPassword)
}

// gradient renders an abstract placeholder portrait background (no real person).
func gradient(hue float64, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / float64(h)
			cx, cy := float64(x)/float64(w)-0.5, t-0.4
			vignette := 1 - math.Min(1, math.Sqrt(cx*cx+cy*cy)*1.3)
			r, g, b := hsl(math.Mod(hue+t*0.12, 1), 0.55, 0.35+0.3*vignette)
			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func hsl(h, s, l float64) (uint8, uint8, uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h*6, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch int(h * 6) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

func randomKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b) + ".jpg"
}
