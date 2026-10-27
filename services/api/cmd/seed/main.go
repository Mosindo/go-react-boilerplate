// Command seed creates clearly-labelled demo accounts for local development.
//
// Demo users use the reserved *.invalid email domain (never deliverable), are
// flagged users.is_demo = TRUE, and can be wiped with `-reset`. The command
// refuses to run when APP_ENV=production.
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

type demoUser struct {
	name, birth, gender, bio, city string
	lat, lng                       float64
	interests                      []string
	interestedIn                   []string
}

var all = []string{"man", "woman", "non_binary"}

var demoUsers = []demoUser{
	{"Camille", "1994-05-02", "woman", "Trail runner and coffee snob. Looking for someone to explore new places with.", "Paris", 48.8566, 2.3522, []string{"running", "coffee", "travel"}, []string{"man", "woman"}},
	{"Lucas", "1991-11-18", "man", "Chef by day, board game addict by night.", "Paris", 48.8738, 2.2950, []string{"cooking", "board_games", "wine"}, []string{"woman"}},
	{"Inès", "1996-03-27", "woman", "Photographer. I collect sunsets and bad puns.", "Montreuil", 48.8610, 2.4430, []string{"photography", "art", "music"}, []string{"man"}},
	{"Mathis", "1990-08-09", "man", "Weekend hiker, weekday engineer.", "Versailles", 48.8049, 2.1204, []string{"hiking", "tech", "nature"}, []string{"woman", "non_binary"}},
	{"Sofia", "1993-01-14", "woman", "Dancing is my love language.", "Paris", 48.8530, 2.3499, []string{"dancing", "festivals", "languages"}, []string{"man", "woman", "non_binary"}},
	{"Noah", "1997-07-21", "man", "Bookworm with a cycling habit.", "Saint-Denis", 48.9362, 2.3574, []string{"books", "cycling", "movies"}, []string{"woman"}},
	{"Alex", "1995-09-30", "non_binary", "Yoga teacher. Plant parent. Always up for a good conversation.", "Paris", 48.8584, 2.2945, []string{"yoga", "nature", "volunteering"}, all},
	{"Léa", "1992-12-05", "woman", "Theatre kid grown up. Wine and stories welcome.", "Boulogne-Billancourt", 48.8352, 2.2410, []string{"theatre", "wine", "books"}, []string{"man"}},
	{"Hugo", "1989-04-16", "man", "Gym, gaming, dogs. In that order. Maybe.", "Paris", 48.8467, 2.3770, []string{"fitness", "gaming", "animals"}, []string{"woman"}},
	{"Jade", "1998-10-11", "woman", "Music festival regular and amateur DJ.", "Paris", 48.8800, 2.3550, []string{"music", "festivals", "dancing"}, []string{"man", "non_binary"}},
	{"Tom", "1993-06-25", "man", "Traveller with a backpack and a bad sense of direction.", "Paris", 48.8420, 2.3220, []string{"travel", "photography", "languages"}, []string{"woman", "man"}},
	{"Manon", "1994-02-08", "woman", "Climbing, cooking and way too many podcasts.", "Nanterre", 48.8924, 2.2069, []string{"fitness", "cooking", "tech"}, []string{"man", "woman"}},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo accounts and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed refuses to run with APP_ENV=production")
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 30e9, 1e9)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.New(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	profiles := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), store)

	// Always start from a clean demo state so the command is idempotent.
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
		cleanup, err := profiles.PrepareUserDeletion(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
			log.Fatal(err)
		}
		cleanup()
	}
	if *reset {
		fmt.Printf("removed %d demo accounts\n", len(ids))
		return
	}

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoPass123"
	}
	authService := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret))

	for i, d := range demoUsers {
		email := fmt.Sprintf("demo%02d%s", i+1, demoDomain)
		_, user, err := authService.Register(ctx, email, password, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET is_demo = TRUE WHERE id = $1`, user.ID); err != nil {
			log.Fatal(err)
		}
		lat, lng := d.lat, d.lng
		if _, err := profiles.Upsert(ctx, user.ID, profilesfeature.UpsertProfileRequest{
			FirstName: d.name, BirthDate: d.birth, Gender: d.gender, Bio: d.bio, City: d.city,
			Latitude: &lat, Longitude: &lng, Interests: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", d.name, err)
		}
		if _, err := profiles.UpdatePreferences(ctx, user.ID, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: d.interestedIn, AgeMin: 18, AgeMax: 60, MaxDistanceKM: 100,
		}); err != nil {
			log.Fatalf("preferences %s: %v", d.name, err)
		}
		for p := 0; p < 2; p++ {
			if _, err := profiles.AddPhoto(ctx, user.ID, avatar(i*2+p)); err != nil {
				log.Fatalf("photo %s: %v", d.name, err)
			}
		}
	}
	fmt.Printf("created %d demo accounts (demoNN%s / %s)\n", len(demoUsers), demoDomain, password)
}

// avatar draws a deterministic abstract gradient: no third-party images, no real faces.
func avatar(seed int) []byte {
	const w, h = 640, 800
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	hue := float64(seed*47%360) / 360
	cx, cy := float64(w)*(0.3+0.4*math.Mod(float64(seed)*0.37, 1)), float64(h)*0.35
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / 700
			t := float64(y) / h
			img.Set(x, y, hsv(math.Mod(hue+0.12*t, 1), 0.55-0.2*d, 0.95-0.45*d-0.25*t))
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsv(h, s, v float64) color.NRGBA {
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
	return color.NRGBA{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255), A: 255}
}
