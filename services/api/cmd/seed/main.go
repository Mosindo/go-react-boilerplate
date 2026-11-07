// Command seed creates clearly-labelled DEMO accounts for local development.
// All demo emails end with @demo.invalid (a reserved, undeliverable domain) so they can never be
// confused with real users, and they can be removed with `go run ./cmd/seed -reset`.
// The command refuses to run when APP_ENV=production.
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
	"os"

	authfeature "example.com/api/internal/features/auth"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
)

const demoDomain = "@demo.invalid"

type demo struct {
	name, gender, birth, city string
	lat, lon                  float64
	interestedIn, interests   []string
	bio                       string
}

var demos = []demo{
	{"Camille", "woman", "1994-04-12", "Paris", 48.8566, 2.3522, []string{"man"}, []string{"travel", "cinema", "yoga"}, "Fan de cinéma d'auteur et de longues balades."},
	{"Lucas", "man", "1991-09-03", "Paris", 48.8606, 2.3376, []string{"woman"}, []string{"cooking", "music", "cycling"}, "Cuisinier amateur, vélo le week-end."},
	{"Inès", "woman", "1997-01-25", "Paris", 48.8738, 2.2950, []string{"man", "woman"}, []string{"art", "photography", "travel"}, "Photographe curieuse."},
	{"Hugo", "man", "1989-11-18", "Boulogne-Billancourt", 48.8352, 2.2400, []string{"woman"}, []string{"sport", "tech", "gaming"}, "Dev le jour, joueur le soir."},
	{"Sarah", "woman", "1993-07-07", "Versailles", 48.8049, 2.1204, []string{"man"}, []string{"hiking", "nature", "reading"}, "Grand air et bons livres."},
	{"Yanis", "man", "1995-02-14", "Montreuil", 48.8634, 2.4430, []string{"woman", "man"}, []string{"music", "festivals", "dancing"}, "Toujours un concert en vue."},
	{"Manon", "woman", "1990-12-01", "Lyon", 45.7640, 4.8357, []string{"man"}, []string{"wine", "theatre", "boardgames"}, "Soirées jeux et bon vin."},
	{"Théo", "man", "1992-06-21", "Lyon", 45.7578, 4.8320, []string{"woman"}, []string{"hiking", "photography", "animals"}, "Mon chien choisira pour moi."},
	{"Alex", "nonbinary", "1996-03-30", "Paris", 48.8530, 2.3499, []string{"man", "woman", "nonbinary"}, []string{"art", "gaming", "tech"}, "Curieux·se de tout."},
	{"Emma", "woman", "1999-08-09", "Marseille", 43.2965, 5.3698, []string{"man"}, []string{"dancing", "travel", "festivals"}, "Soleil, mer et musique."},
	{"Noah", "man", "1988-05-05", "Marseille", 43.2800, 5.3800, []string{"woman"}, []string{"sport", "cooking", "cinema"}, "Cuisine du sud et football."},
	{"Léa", "woman", "1998-10-10", "Paris", 48.8450, 2.3700, []string{"man"}, []string{"yoga", "reading", "cinema"}, "Yoga le matin, ciné le soir."},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo accounts and exit")
	flag.Parse()

	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("seed refuses to run when APP_ENV=production")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	files, err := storage.NewDisk(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *reset {
		rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.email LIKE '%`+demoDomain+`'`)
		if err != nil {
			log.Fatal(err)
		}
		for rows.Next() {
			var k string
			_ = rows.Scan(&k)
			files.Delete(k)
		}
		rows.Close()
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%`+demoDomain+`'`)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo accounts\n", tag.RowsAffected())
		return
	}

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "demo-password-123"
	}
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), authfeature.Deps{})
	profSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), files, profilesfeature.NewMediaSigner([]byte(cfg.JWTSecret)))

	created := 0
	for i, d := range demos {
		email := fmt.Sprintf("demo%02d%s", i+1, demoDomain)
		_, user, err := authSvc.Register(ctx, email, password, "seed", "")
		if err != nil {
			fmt.Printf("skip %s: %v\n", email, err)
			continue
		}
		req := profilesfeature.UpsertProfileRequest{
			FirstName: d.name, BirthDate: d.birth, Gender: d.gender, Bio: d.bio, City: d.city,
			Latitude: &d.lat, Longitude: &d.lon, Interests: d.interests,
		}
		req.Preferences.InterestedIn = d.interestedIn
		req.Preferences.AgeMin, req.Preferences.AgeMax, req.Preferences.MaxDistanceKm = 18, 60, 100
		if _, err := profSvc.Upsert(ctx, user.ID, req); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		for j := 0; j < 2; j++ {
			if _, err := profSvc.AddPhoto(ctx, user.ID, gradient(float64(i*47+j*120))); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo accounts (password: %s). Login e.g. demo01%s\n", created, password, demoDomain)
}

// gradient renders a deterministic placeholder portrait so demo cards are not empty.
func gradient(hue float64) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			img.Set(x, y, hsv(math.Mod(hue+t*40, 360), 0.55, 0.95-0.35*t))
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsv(h, s, v float64) color.RGBA {
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
