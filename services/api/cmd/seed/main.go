// Command seed inserts clearly-labelled demo accounts for local testing.
//
//	go run ./cmd/seed            # refuses to run when APP_ENV=production
//
// Demo accounts use the reserved e-mail domain demo.invalid (RFC 2606), so they
// can never collide with or be mistaken for real users. Remove them with
// `go run ./cmd/seed -purge`.
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
	"math/rand"
	"strings"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	demoDomain   = "@demo.invalid"
	demoPassword = "DemoPassword123"
)

type demoUser struct {
	name, gender string
	interestedIn []string
	age          int
	bio          string
	interests    []string
}

var demoUsers = []demoUser{
	{"Camille", "woman", []string{"man"}, 29, "Fan de randonnée et de bons cafés.", []string{"hiking", "coffee", "photography"}},
	{"Léa", "woman", []string{"man", "woman"}, 26, "Cuisine, cinéma et longues balades.", []string{"cooking", "cinema", "travel"}},
	{"Inès", "woman", []string{"man"}, 31, "Yoga le matin, musique le soir.", []string{"yoga", "music", "nature"}},
	{"Jade", "woman", []string{"woman"}, 28, "Toujours un livre dans le sac.", []string{"reading", "art", "theatre"}},
	{"Chloé", "woman", []string{"man"}, 24, "Danse, voyages et fous rires.", []string{"dancing", "travel", "languages"}},
	{"Hugo", "man", []string{"woman"}, 30, "Vélo, vin et photographie.", []string{"cycling", "wine", "photography"}},
	{"Lucas", "man", []string{"woman"}, 27, "Développeur le jour, cuisinier le soir.", []string{"tech", "cooking", "gaming"}},
	{"Nathan", "man", []string{"woman", "man"}, 33, "Sport, nature et bonne musique.", []string{"sports", "nature", "music"}},
	{"Théo", "man", []string{"man"}, 29, "Cinéma, théâtre et cafés.", []string{"cinema", "theatre", "coffee"}},
	{"Sacha", "nonbinary", []string{"man", "woman", "nonbinary"}, 27, "Curieux·se de tout, surtout des gens.", []string{"art", "languages", "animals"}},
	{"Noa", "nonbinary", []string{"woman", "nonbinary"}, 25, "Jeux vidéo et randonnées.", []string{"gaming", "hiking", "animals"}},
	{"Maël", "man", []string{"woman"}, 35, "Voyages, lecture, et un chien nommé Pixel.", []string{"travel", "reading", "animals"}},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo accounts instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
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
	storage, err := photos.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		rows, err := pool.Query(ctx, `DELETE FROM users WHERE email LIKE '%' || $1 RETURNING id`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		var ids []string
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			_ = storage.RemoveUserFiles(ctx, id)
		}
		fmt.Printf("removed %d demo accounts\n", len(ids))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	rng := rand.New(rand.NewSource(42))
	created := 0
	for i, u := range demoUsers {
		email := strings.ToLower(u.name) + demoDomain
		email = strings.NewReplacer("é", "e", "è", "e", "ë", "e").Replace(email)
		var userID string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash) VALUES ($1, $2)
			ON CONFLICT (email) DO NOTHING RETURNING id`, email, string(hash)).Scan(&userID)
		if err != nil {
			continue // already seeded
		}
		birth := time.Now().AddDate(-u.age, -rng.Intn(11), -rng.Intn(27))
		lat := 48.85 + (rng.Float64()-0.5)*0.2
		lng := 2.35 + (rng.Float64()-0.5)*0.3
		if _, err := pool.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, lat, lng)
			VALUES ($1, $2, $3, $4, $5, 'Paris', ROUND($6::numeric, 2), ROUND($7::numeric, 2))`,
			userID, u.name, birth, u.gender, u.bio, lat, lng); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
			VALUES ($1, $2, 21, 45, 100)`, userID, u.interestedIn); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)`, userID, u.interests); err != nil {
			log.Fatal(err)
		}
		for pos := 0; pos < 2; pos++ {
			processed, err := photos.Process(bytes.NewReader(gradient(i*2 + pos)))
			if err != nil {
				log.Fatal(err)
			}
			key := userID + "/" + uuid.NewString() + ".jpg"
			if err := storage.Save(key, processed.Data); err != nil {
				log.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO photos (user_id, position, storage_key, width, height, bytes)
				VALUES ($1, $2, $3, $4, $5, $6)`, userID, pos, key, processed.Width, processed.Height, len(processed.Data)); err != nil {
				log.Fatal(err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo accounts (password %q, e-mail <first name>%s)\n", created, demoPassword, demoDomain)
	fmt.Println("e.g. camille" + demoDomain + " / hugo" + demoDomain)
}

// gradient renders a deterministic abstract placeholder portrait.
func gradient(seed int) []byte {
	const w, h = 640, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	hue := float64(seed*47%360) / 360
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y)/h*0.6 + float64(x)/w*0.4
			r, g, b := hsv(hue+t*0.12, 0.55, 0.95-t*0.45)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsv(h, s, v float64) (uint8, uint8, uint8) {
	h -= float64(int(h))
	i := int(h * 6)
	f := h*6 - float64(i)
	p, q, t := v*(1-s), v*(1-f*s), v*(1-(1-f)*s)
	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
