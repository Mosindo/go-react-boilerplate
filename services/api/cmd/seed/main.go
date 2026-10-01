// Command seed fills a development database with clearly fake demo accounts.
//
//	go run ./cmd/seed          create demo accounts (idempotent)
//	go run ./cmd/seed -purge   delete every demo account
//
// Demo users live under the reserved @demo.invalid domain, are flagged users.is_demo = true and the
// command refuses to run when APP_ENV=production, so they can never be mistaken for real members.
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
	"math/rand"
	"os"
	"strings"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"golang.org/x/crypto/bcrypt"
)

const demoPassword = "DemoPassword123"

type demo struct {
	name, gender string
	interested   []string
	age          int
	city         string
	lat, lng     float64
	bio          string
	interests    []string
}

var people = []demo{
	{"Camille", "woman", []string{"man"}, 29, "Paris", 48.8566, 2.3522, "Passionnée de cuisine et de randonnées du dimanche.", []string{"cuisine", "randonnee", "cinema"}},
	{"Hugo", "man", []string{"woman"}, 32, "Paris", 48.8700, 2.3800, "Photographe amateur, toujours partant pour un café.", []string{"photographie", "cafe", "voyages"}},
	{"Inès", "woman", []string{"man", "woman"}, 27, "Boulogne-Billancourt", 48.8352, 2.2410, "Danseuse le soir, ingénieure le jour.", []string{"danse", "technologie", "musique"}},
	{"Léo", "man", []string{"woman"}, 35, "Versailles", 48.8014, 2.1301, "Cycliste, lecteur compulsif et mauvais joueur aux jeux de société.", []string{"velo", "lecture", "jeux-de-societe"}},
	{"Manon", "woman", []string{"man"}, 31, "Saint-Denis", 48.9362, 2.3574, "Yoga, plantes vertes et concerts.", []string{"yoga", "jardinage", "concerts"}},
	{"Nathan", "man", []string{"woman", "man"}, 26, "Montreuil", 48.8634, 2.4430, "Musicien, cuisinier du dimanche.", []string{"musique", "cuisine", "concerts"}},
	{"Zoé", "non_binary", []string{"woman", "man", "non_binary"}, 28, "Paris", 48.8600, 2.3400, "Théâtre, art contemporain et longues balades.", []string{"theatre", "art", "nature"}},
	{"Adam", "man", []string{"woman"}, 38, "Créteil", 48.7904, 2.4556, "Running, cinéma et bons restaurants.", []string{"running", "cinema", "gastronomie"}},
	{"Jade", "woman", []string{"man"}, 24, "Paris", 48.8530, 2.3499, "Étudiante, curieuse de tout.", []string{"lecture", "voyages", "mode"}},
	{"Lucas", "man", []string{"woman"}, 30, "Lyon", 45.7640, 4.8357, "Lyonnais, fan de gastronomie et de vélo.", []string{"gastronomie", "velo", "vin"}},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo accounts instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed refuses to run with APP_ENV=production")
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
	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	svc, err := app.NewRouter(app.Deps{Pool: pool, JWTSecret: []byte(cfg.JWTSecret), Store: store, Mailer: mailer.Log{}})
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		rows, err := pool.Query(ctx, `SELECT id FROM users WHERE is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				log.Fatal(err)
			}
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			if err := svc.Photos.PurgeUserFiles(ctx, id); err != nil {
				log.Fatal(err)
			}
		}
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("deleted %d demo accounts\n", tag.RowsAffected())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	created := 0
	for i, p := range people {
		email := fmt.Sprintf("%s@demo.invalid", strings.ToLower(strings.NewReplacer("è", "e", "é", "e", "ë", "e", "ï", "i", "ö", "o").Replace(p.name)))
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE)
			ON CONFLICT (email) DO NOTHING RETURNING id
		`, email, string(hash)).Scan(&id)
		if err != nil {
			continue // already seeded
		}
		birth := time.Now().AddDate(-p.age, 0, -30)
		if _, err := pool.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
			VALUES ($1, $2, $3, $4, $5, $6, round($7::numeric, 2), round($8::numeric, 2))
		`, id, p.name, birth, p.gender, p.bio, p.city, p.lat, p.lng); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 21, 45, 100)`, id, p.interested); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, id, p.interests); err != nil {
			log.Fatal(err)
		}
		for n := 0; n < 2; n++ {
			if _, err := svc.Photos.Add(ctx, id, bytes.NewReader(placeholderPhoto(i*3+n))); err != nil {
				log.Fatal(err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo accounts (password %q, emails *@demo.invalid)\n", created, demoPassword)
	_ = os.Stdout.Sync()
}

// placeholderPhoto renders a soft gradient so demo profiles do not use anyone's real picture.
func placeholderPhoto(seed int) []byte {
	r := rand.New(rand.NewSource(int64(seed) + 7))
	h1, h2 := r.Float64()*360, r.Float64()*360
	img := image.NewRGBA(image.Rect(0, 0, 720, 960))
	for y := 0; y < 960; y++ {
		t := float64(y) / 960
		for x := 0; x < 720; x++ {
			u := float64(x) / 720
			img.Set(x, y, hsl(h1+(h2-h1)*t, 0.55, 0.45+0.2*math.Sin(u*math.Pi)*0.5+0.1*t))
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
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
	return color.RGBA{R: uint8((r + m) * 255), G: uint8((g + m) * 255), B: uint8((b + m) * 255), A: 255}
}
