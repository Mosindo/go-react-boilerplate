// Command seed creates clearly-labelled demo accounts for local development.
//
// Demo users are flagged users.is_demo = TRUE and use the reserved
// "@demo.invalid" domain, so they can never be confused with (or collide with)
// real accounts. The command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed                      # create demo users
//	go run ./cmd/seed -like-me me@x.com    # also make demo users like that account
//	go run ./cmd/seed -reset               # delete every demo user
package main

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type demoUser struct {
	Name         string
	Gender       string
	InterestedIn []string
	BirthYear    int
	Bio          string
	Lat, Lng     float64
	City         string
	Interests    []string
	Hue          float64
}

var demoUsers = []demoUser{
	{"Camille", "woman", []string{"man"}, 1994, "Lectrice compulsive et randonneuse du dimanche.", 45.76, 4.84, "Lyon", []string{"reading", "hiking", "coffee"}, 12},
	{"Julien", "man", []string{"woman"}, 1990, "Cuisinier amateur, fan de cinéma d'auteur.", 45.77, 4.85, "Lyon", []string{"cooking", "cinema", "wine"}, 210},
	{"Léa", "woman", []string{"man", "woman"}, 1997, "Danse, festivals et bons restos.", 45.75, 4.83, "Villeurbanne", []string{"dancing", "festivals", "music"}, 320},
	{"Hugo", "man", []string{"woman"}, 1992, "Développeur le jour, photographe le week-end.", 45.78, 4.87, "Lyon", []string{"tech", "photography", "travel"}, 160},
	{"Inès", "woman", []string{"man"}, 1995, "Yoga au lever du soleil, café ensuite.", 45.74, 4.86, "Lyon", []string{"yoga", "coffee", "nature"}, 45},
	{"Théo", "man", []string{"woman", "man"}, 1989, "Toujours un livre et un sac de rando prêts.", 45.80, 4.80, "Caluire", []string{"hiking", "reading", "animals"}, 95},
	{"Sarah", "woman", []string{"man"}, 1993, "Voyageuse, curieuse, un peu bavarde.", 45.76, 4.89, "Lyon", []string{"travel", "art", "theatre"}, 280},
	{"Nathan", "man", []string{"woman"}, 1996, "Sport, jeux vidéo et pizzas.", 45.71, 4.81, "Vénissieux", []string{"sport", "gaming", "cooking"}, 190},
	{"Alex", "nonbinary", []string{"man", "woman", "nonbinary"}, 1995, "Musique live et expositions.", 45.76, 4.83, "Lyon", []string{"music", "art", "festivals"}, 260},
	{"Manon", "woman", []string{"man"}, 1991, "Bénévole et amoureuse des animaux.", 45.79, 4.84, "Lyon", []string{"volunteering", "animals", "nature"}, 340},
	{"Maxime", "man", []string{"woman"}, 1988, "Vin, théâtre et longues balades.", 45.73, 4.82, "Lyon", []string{"wine", "theatre", "hiking"}, 30},
	{"Clara", "woman", []string{"woman", "man"}, 1998, "Photographe, passionnée de cinéma.", 45.77, 4.78, "Écully", []string{"photography", "cinema", "coffee"}, 130},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo users and exit")
	likeMe := flag.String("like-me", "", "email of an existing account that every compatible demo user should like")
	flag.Parse()

	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	url := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if url == "" {
		log.Fatal("seed: DATABASE_URL is required")
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

	if *reset {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("deleted %d demo users\n", tag.RowsAffected())
		return
	}

	password := os.Getenv("DEMO_PASSWORD")
	if password == "" {
		password = "demo-password-123"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	created := 0
	for _, u := range demoUsers {
		ok, err := createUser(ctx, pool, u, string(hash))
		if err != nil {
			log.Fatalf("seed %s: %v", u.Name, err)
		}
		if ok {
			created++
		}
	}
	fmt.Printf("demo users ready (%d created, %d already present)\n", created, len(demoUsers)-created)
	fmt.Printf("login with <name>@demo.invalid, e.g. camille@demo.invalid / %s\n", password)

	if *likeMe != "" {
		n, err := likeUser(ctx, pool, strings.ToLower(strings.TrimSpace(*likeMe)))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d demo users now like %s: like them back to get matches\n", n, *likeMe)
	}
}

func email(name string) string {
	r := strings.NewReplacer("é", "e", "è", "e", "ï", "i", "ë", "e")
	return strings.ToLower(r.Replace(name)) + "@demo.invalid"
}

func createUser(ctx context.Context, pool *pgxpool.Pool, u demoUser, hash string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, birth_date, is_demo)
		VALUES ($1, $2, make_date($3, 6, 15), TRUE)
		ON CONFLICT (email) DO NOTHING RETURNING id`, email(u.Name), hash, u.BirthYear).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, gender, bio, city, latitude, longitude)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, u.Name, u.Gender, u.Bio, u.City, u.Lat, u.Lng); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km) VALUES ($1,$2,18,60,100)`, id, u.InterestedIn); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)`, id, u.Interests); err != nil {
		return false, err
	}
	for pos := 0; pos < 2; pos++ {
		data, err := portrait(u.Hue+float64(pos)*35, pos)
		if err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO photos (user_id, position, mime_type, size_bytes, data) VALUES ($1,$2,'image/jpeg',$3,$4)`, id, pos, len(data), data); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}

func likeUser(ctx context.Context, pool *pgxpool.Pool, targetEmail string) (int64, error) {
	var targetID string
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND NOT is_demo`, targetEmail).Scan(&targetID); err != nil {
		return 0, fmt.Errorf("no real account with email %q (register in the app first): %w", targetEmail, err)
	}
	tag, err := pool.Exec(ctx, `
		INSERT INTO likes (swiper_id, target_id, kind, created_at)
		SELECT u.id, $1, 'like', $2 FROM users u WHERE u.is_demo
		ON CONFLICT DO NOTHING`, targetID, time.Now())
	return tag.RowsAffected(), err
}

// portrait renders a soft abstract avatar (no real person's image is used).
func portrait(hue float64, variant int) ([]byte, error) {
	const w, h = 480, 640
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top := hsv(hue, 0.45, 0.95)
	bottom := hsv(hue+30, 0.65, 0.65)
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		c := color.RGBA{lerp(top.R, bottom.R, t), lerp(top.G, bottom.G, t), lerp(top.B, bottom.B, t), 255}
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	skin := hsv(hue+15, 0.25, 0.98)
	cx, cy := float64(w)/2, float64(h)*0.42+float64(variant*20)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			head := dx*dx/(90*90)+dy*dy/(110*110) <= 1
			sx, sy := float64(x)-cx, float64(y)-(cy+330)
			body := sx*sx/(190*190)+sy*sy/(210*210) <= 1
			if head || body {
				img.SetRGBA(x, y, skin)
			}
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func lerp(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t) }

func hsv(h, s, v float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
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
