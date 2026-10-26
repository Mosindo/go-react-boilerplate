// Command seed loads clearly-marked demo accounts for local development.
//
// Every demo account uses the reserved domain "demo.invalid" (it can never be a
// real mailbox) and the command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed          # create or refresh demo data
//	go run ./cmd/seed -reset   # delete all demo accounts
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
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.invalid"

type demoUser struct {
	Email, Name, Gender string
	Interested          []string
	Age                 int
	Bio, Job            string
	Interests           []string
	DLat, DLng          float64 // offset from the seed centre, in degrees
	Hue                 float64
}

var demoUsers = []demoUser{
	{"alex", "Alex", "man", []string{"woman"}, 31, "Fan de randonnée et de cuisine du dimanche.", "Ingénieur", []string{"hiking", "cooking", "music"}, 0.00, 0.00, 210},
	{"camille", "Camille", "woman", []string{"man", "woman"}, 28, "Toujours un livre dans le sac. Je cherche quelqu'un qui aime rire.", "Designer", []string{"reading", "art", "travel", "coffee"}, 0.01, 0.01, 330},
	{"lea", "Léa", "woman", []string{"man"}, 26, "Yoga le matin, concerts le soir.", "Infirmière", []string{"yoga", "concerts", "music", "nature"}, -0.02, 0.01, 20},
	{"nina", "Nina", "woman", []string{"man", "woman"}, 33, "Photographe amatrice, grande fan de cinéma.", "Photographe", []string{"photography", "cinema", "travel"}, 0.03, -0.02, 280},
	{"sam", "Sam", "nonbinary", []string{"man", "woman", "nonbinary"}, 29, "Jeux de société, thé et bonne humeur.", "Développeur·se", []string{"boardgames", "tech", "gaming", "coffee"}, -0.01, -0.03, 160},
	{"hugo", "Hugo", "man", []string{"woman"}, 34, "Cycliste du week-end, cuisinier du dimanche.", "Architecte", []string{"cycling", "cooking", "wine"}, 0.02, 0.03, 40},
	{"ines", "Inès", "woman", []string{"man"}, 27, "Danse, voyages et longues discussions.", "Juriste", []string{"dancing", "travel", "languages"}, -0.03, -0.01, 300},
	{"marc", "Marc", "man", []string{"woman", "man"}, 38, "Course à pied, bons restos, bons films.", "Chef de projet", []string{"running", "cinema", "wine"}, 0.04, 0.00, 190},
	{"julie", "Julie", "woman", []string{"man"}, 30, "Escalade et jardinage : je suis plutôt nature.", "Vétérinaire", []string{"climbing", "gardening", "animals"}, -0.04, 0.02, 120},
	{"theo", "Théo", "man", []string{"woman"}, 25, "Football, séries et pizzas.", "Étudiant", []string{"football", "series", "gaming"}, 0.01, -0.04, 250},
	{"maya", "Maya", "woman", []string{"man", "nonbinary"}, 32, "Théâtre, bénévolat et café allongé.", "Enseignante", []string{"theatre", "volunteering", "coffee"}, -0.02, 0.04, 350},
	{"paul", "Paul", "man", []string{"woman", "nonbinary"}, 36, "Musicien à mes heures perdues.", "Musicien", []string{"music", "concerts", "reading"}, 0.05, 0.02, 90},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo accounts instead of creating them")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("seed refuses to run with APP_ENV=production")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
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

	if *reset {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("deleted %d demo accounts\n", tag.RowsAffected())
		return
	}

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoPass123"
	}
	lat := envFloat("SEED_LAT", 48.8566)
	lng := envFloat("SEED_LNG", 2.3522)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	ids := map[string]string{}
	for _, u := range demoUsers {
		id, err := upsertUser(ctx, pool, u, string(hash), lat, lng)
		if err != nil {
			log.Fatalf("seed %s: %v", u.Name, err)
		}
		ids[u.Email] = id
	}

	// A few people already liked the first demo account, so a like-back shows the match flow.
	for _, from := range []string{"camille", "lea", "nina"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING
		`, ids[from], ids["alex"]); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("seeded %d demo accounts (password %q)\n", len(demoUsers), password)
	fmt.Printf("sign in with e.g. alex@%s or camille@%s\n", demoDomain, demoDomain)
}

func upsertUser(ctx context.Context, pool *pgxpool.Pool, u demoUser, hash string, lat, lng float64) (string, error) {
	email := u.Email + "@" + demoDomain
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id
	`, email, hash).Scan(&id); err != nil {
		return "", err
	}
	birth := time.Now().AddDate(-u.Age, 0, -30)
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, job_title, city, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, 'Démo', $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET first_name = EXCLUDED.first_name, bio = EXCLUDED.bio,
		  job_title = EXCLUDED.job_title, latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude
	`, id, u.Name, birth, u.Gender, u.Bio, u.Job, round2(lat+u.DLat), round2(lng+u.DLng)); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, 18, 60, 100)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in
	`, id, u.Interested); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, id); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2::text[])
	`, id, u.Interests); err != nil {
		return "", err
	}
	if err := replacePhotos(ctx, tx, id, u.Hue); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

func replacePhotos(ctx context.Context, tx pgx.Tx, userID string, hue float64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM photos WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for pos := 0; pos < 3; pos++ {
		data, w, h := gradientJPEG(hue+float64(pos)*25, 720, 960)
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, position, width, height, data) VALUES ($1, $2, $3, $4, $5)
		`, userID, pos, w, h, data); err != nil {
			return err
		}
	}
	return nil
}

// gradientJPEG draws a soft diagonal gradient so demo cards have a distinct look
// without shipping any real person's picture.
func gradientJPEG(hue float64, w, h int) ([]byte, int, int) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := (float64(x)/float64(w) + float64(y)/float64(h)) / 2
			r, g, b := hsl(math.Mod(hue+t*40, 360), 0.55, 0.78-0.28*t)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes(), w, h
}

func hsl(h, s, l float64) (uint8, uint8, uint8) {
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
	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func envFloat(key string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil {
		return v
	}
	return fallback
}
