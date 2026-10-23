// Command seed fills a development database with clearly fake demo accounts.
//
// Demo users live under the reserved, non-routable domain demo.invalid so they can
// never be mistaken for real people, and the command refuses to run unless
// ALLOW_DEMO_SEED=true is set explicitly. Re-running it is safe (existing demo
// users are skipped). Remove them with: DELETE FROM users WHERE email LIKE '%@demo.invalid';
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
)

const demoDomain = "demo.invalid"

type demoUser struct {
	name      string
	gender    string
	seeks     []string
	age       int
	city      string
	lat, lng  float64
	bio       string
	interests []string
	hue       int
}

var demoUsers = []demoUser{
	{"Camille", "woman", []string{"man", "woman"}, 29, "Lyon", 45.764, 4.8357, "Randonnées le week-end, cuisine le soir.", []string{"hiking", "cooking", "cinema"}, 10},
	{"Julien", "man", []string{"woman"}, 32, "Lyon", 45.75, 4.85, "Photographe amateur, toujours partant pour un concert.", []string{"photography", "concerts", "travel"}, 60},
	{"Inès", "woman", []string{"man"}, 27, "Villeurbanne", 45.766, 4.879, "Climbing, séries et café.", []string{"climbing", "series", "restaurants"}, 120},
	{"Hugo", "man", []string{"woman", "non_binary"}, 34, "Lyon", 45.77, 4.83, "Je lis trop et je cours pour compenser.", []string{"reading", "running", "museums"}, 180},
	{"Sacha", "non_binary", []string{"man", "woman", "non_binary"}, 30, "Lyon", 45.758, 4.832, "Théâtre, jeux de société et bons vins.", []string{"theatre", "boardgames", "wine"}, 240},
	{"Léa", "woman", []string{"man"}, 31, "Lyon", 45.745, 4.87, "Yoga au lever du soleil, tennis le dimanche.", []string{"yoga", "tennis", "nature"}, 300},
	{"Thomas", "man", []string{"woman"}, 28, "Caluire", 45.795, 4.84, "Vélo, jardinage et cuisine maison.", []string{"cycling", "gardening", "cooking"}, 30},
	{"Manon", "woman", []string{"man", "woman"}, 26, "Lyon", 45.76, 4.86, "Danse, musées et longues balades.", []string{"dance", "museums", "hiking"}, 90},
	{"Karim", "man", []string{"woman", "man"}, 35, "Lyon", 45.74, 4.82, "Jeux vidéo, tech et football.", []string{"gaming", "technology", "football"}, 150},
	{"Elsa", "woman", []string{"man"}, 33, "Lyon", 45.78, 4.81, "Bénévole dans un refuge, fan de cinéma d'auteur.", []string{"volunteering", "animals", "cinema"}, 210},
}

func main() {
	if os.Getenv("ALLOW_DEMO_SEED") != "true" {
		log.Fatal("refusing to seed: set ALLOW_DEMO_SEED=true to confirm this is a development database")
	}
	password := os.Getenv("DEMO_PASSWORD")
	if password == "" {
		password = "DemoPass123"
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 10*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	authService := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.LogMailer{})
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(pool))
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), photosService)

	created := 0
	for i, u := range demoUsers {
		email := fmt.Sprintf("demo%02d@%s", i+1, demoDomain)
		_, user, err := authService.Register(ctx, email, password, "seed", "")
		if err != nil {
			if appErr, ok := platformerrors.As(err); ok && appErr.Code == "email_taken" {
				continue
			}
			log.Fatalf("register %s: %v", email, err)
		}
		_, err = profilesService.SaveProfile(ctx, user.ID, profilesfeature.ProfileInput{
			FirstName: u.name,
			BirthDate: time.Now().AddDate(-u.age, 0, -3).Format("2006-01-02"),
			Gender:    u.gender,
			Bio:       u.bio + " (compte de démonstration)",
			City:      u.city,
			Latitude:  &u.lat,
			Longitude: &u.lng,
			Interests: u.interests,
		})
		if err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if _, err := profilesService.SetPreferences(ctx, user.ID, profilesfeature.Preferences{
			InterestedIn: u.seeks, MinAge: 18, MaxAge: 60, MaxDistanceKm: 100, Discoverable: true, ShowDistance: true,
		}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		if _, err := photosService.Upload(ctx, user.ID, placeholderPhoto(u.hue)); err != nil {
			log.Fatalf("photo %s: %v", email, err)
		}
		created++
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	log.Printf("seed done: %d demo users created (%d total). Sign in as demo01@%s with password %q", created, len(demoUsers), demoDomain, password)
}

// placeholderPhoto renders a soft diagonal gradient; demo accounts never use real people's photos.
func placeholderPhoto(hue int) []byte {
	const w, h = 480, 640
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / float64(w+h)
			img.Set(x, y, color.RGBA{
				R: uint8(60 + float64(hue)*0.3 + 120*t),
				G: uint8(80 + 100*(1-t)),
				B: uint8(140 + 80*t),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
