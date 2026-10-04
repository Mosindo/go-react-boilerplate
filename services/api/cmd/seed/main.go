// Command seed fills a DEVELOPMENT database with clearly fake demo accounts.
//
//	go run ./cmd/seed          # create demo accounts (idempotent)
//	go run ./cmd/seed -purge   # delete every demo account again
//
// Demo accounts use the reserved, undeliverable domain demo.lumen.invalid so they can never be mistaken
// for real users, and the command refuses to run when APP_ENV=production.
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
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
)

const demoDomain = "demo.lumen.invalid"

type demo struct {
	name, gender, birth, city, bio string
	lat, lng                       float64
	into                           []string
	interests                      []string
}

var demos = []demo{
	{"Camille", "woman", "1994-03-12", "Paris", "Sunday markets, bad puns and long walks along the Seine.", 48.8566, 2.3522, []string{"man", "woman"}, []string{"cooking", "travel", "cinema", "coffee"}},
	{"Lucas", "man", "1991-07-02", "Paris", "Climbing on weekends, cooking on weeknights.", 48.8600, 2.3400, []string{"woman"}, []string{"climbing", "cooking", "music"}},
	{"Inès", "woman", "1996-11-23", "Lyon", "Photographer chasing golden hour. Ask me about my cat.", 45.7640, 4.8357, []string{"man"}, []string{"photography", "animals", "hiking"}},
	{"Hugo", "man", "1989-01-30", "Lyon", "Runner, reader, terrible dancer (working on it).", 45.7500, 4.8500, []string{"woman"}, []string{"running", "reading", "dancing"}},
	{"Sofia", "woman", "1993-05-09", "Marseille", "Sea, sun, and a good book. Looking for someone curious.", 43.2965, 5.3698, []string{"man", "woman"}, []string{"swimming", "reading", "travel"}},
	{"Noah", "man", "1995-09-14", "Marseille", "Musician by night, developer by day.", 43.3000, 5.4000, []string{"woman", "nonbinary"}, []string{"music", "tech", "concerts"}},
	{"Léa", "woman", "1998-02-18", "Bordeaux", "Wine nerd, trail runner, board-game champion.", 44.8378, -0.5792, []string{"man"}, []string{"wine", "running", "board-games"}},
	{"Jules", "man", "1992-12-05", "Bordeaux", "Always planning the next road trip.", 44.8400, -0.5700, []string{"woman"}, []string{"road-trips", "photography", "foodie"}},
	{"Alex", "nonbinary", "1997-04-27", "Paris", "Theatre kid grown up. Yoga, plants, and strong opinions on coffee.", 48.8700, 2.3300, []string{"man", "woman", "nonbinary"}, []string{"theatre", "yoga", "gardening", "coffee"}},
	{"Manon", "woman", "1990-08-08", "Toulouse", "Scientist who laughs at her own experiments.", 43.6047, 1.4442, []string{"man"}, []string{"science", "cycling", "festivals"}},
	{"Théo", "man", "1994-10-21", "Toulouse", "Football on Saturdays, lazy brunches on Sundays.", 43.6100, 1.4500, []string{"woman"}, []string{"football", "foodie", "series"}},
	{"Zoé", "woman", "1999-06-16", "Paris", "Volunteering, languages, and spontaneous trips.", 48.8500, 2.3600, []string{"man"}, []string{"volunteering", "languages", "travel"}},
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
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 15*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	files, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	authRepo := authfeature.NewPGRepository(pool)
	auth := authfeature.NewService(authRepo, []byte(cfg.JWTSecret), authfeature.Deps{Mailer: mail.Log{}, Files: files})
	profiles := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	photos := photosfeature.NewService(photosfeature.NewPGRepository(pool), files)

	password := os.Getenv("DEMO_PASSWORD")
	if password == "" {
		password = "lumen-demo-pass"
	}

	if *purge {
		rows, err := pool.Query(ctx, `SELECT id FROM users WHERE email LIKE '%@' || $1`, demoDomain)
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
			keys, err := authRepo.DeleteUser(ctx, id)
			if err != nil {
				log.Fatal(err)
			}
			for _, k := range keys {
				_ = files.Delete(ctx, k)
			}
		}
		fmt.Printf("removed %d demo accounts\n", len(ids))
		return
	}

	created := 0
	for i, d := range demos {
		email := fmt.Sprintf("%s.%02d@%s", asciiLower(d.name), i+1, demoDomain)
		tokens, user, err := auth.Register(ctx, email, password, "seed", "")
		if err != nil {
			if appErr, ok := platformerrors.As(err); ok && appErr.Code == "conflict" {
				continue // already seeded
			}
			log.Fatalf("register %s: %v", email, err)
		}
		_ = tokens
		if _, err := profiles.Save(ctx, user.ID, profilesfeature.SaveProfileRequest{
			DisplayName: d.name, BirthDate: d.birth, Gender: d.gender, Bio: d.bio, City: d.city, Interests: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if _, err := profiles.SavePreferences(ctx, user.ID, profilesfeature.Preferences{InterestedIn: d.into, MinAge: 18, MaxAge: 60, MaxDistanceKm: 100}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		lat, lng := d.lat, d.lng
		if err := profiles.SetLocation(ctx, user.ID, profilesfeature.LocationRequest{Latitude: &lat, Longitude: &lng}); err != nil {
			log.Fatalf("location %s: %v", email, err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photos.Upload(ctx, user.ID, gradient(float64(i)*31+float64(p)*47)); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo accounts (%d total defined)\nlogin: <name>.<nn>@%s / %s   e.g. camille.01@%s\n", created, len(demos), demoDomain, password, demoDomain)
}

func asciiLower(s string) string {
	repl := map[rune]rune{'è': 'e', 'é': 'e', 'ï': 'i', 'ô': 'o', 'à': 'a'}
	out := []rune{}
	for _, r := range s {
		if v, ok := repl[r]; ok {
			r = v
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}

// gradient renders a soft abstract portrait-format placeholder (no real people involved).
func gradient(hue float64) []byte {
	const w, h = 800, 1000
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y)/h*0.7 + float64(x)/w*0.3
			r, g, b := hsl(math.Mod(hue+t*60, 360), 0.65, 0.35+0.35*t)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
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
