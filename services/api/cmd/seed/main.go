// Command seed fills a development database with clearly fake demo accounts.
//
// Every demo account uses an address ending in @demo.invalid (a reserved,
// undeliverable domain), so it can never be mistaken for a real user, and the
// command refuses to run when APP_ENV=production. It is idempotent.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
)

const demoDomain = "demo.invalid"

type demoUser struct {
	name      string
	gender    string
	age       int
	city      string
	lat, lng  float64
	bio       string
	interests []int16
	into      []string
	hue       int
}

var demoUsers = []demoUser{
	{"Camille", "woman", 28, "Paris", 48.8566, 2.3522, "Grimpeuse du dimanche, fan de cuisine de rue.", []int16{1, 2, 6}, []string{"man"}, 10},
	{"Lucas", "man", 31, "Paris", 48.8738, 2.2950, "Photographe, café noir et vieux vinyles.", []int16{3, 9, 15}, []string{"woman"}, 40},
	{"Inès", "woman", 26, "Montreuil", 48.8634, 2.4430, "Danseuse et lectrice compulsive.", []int16{5, 13, 14}, []string{"man", "non_binary"}, 70},
	{"Hugo", "man", 34, "Boulogne-Billancourt", 48.8352, 2.2400, "Je cuisine pour mes amis, je cours pour compenser.", []int16{2, 22, 16}, []string{"woman"}, 100},
	{"Sofia", "woman", 30, "Paris", 48.8530, 2.3499, "Cinéma d'auteur et randonnées improvisées.", []int16{4, 6, 18}, []string{"man"}, 130},
	{"Yanis", "man", 29, "Saint-Denis", 48.9362, 2.3574, "Musicien amateur, toujours un jeu de société dans le sac.", []int16{3, 12, 14}, []string{"woman"}, 160},
	{"Maëlle", "woman", 33, "Versailles", 48.8049, 2.1204, "Yoga le matin, théâtre le soir.", []int16{8, 21, 10}, []string{"man", "woman"}, 190},
	{"Théo", "man", 27, "Paris", 48.8606, 2.3376, "Vélo, bouquins et petites expositions.", []int16{23, 5, 10}, []string{"woman", "non_binary"}, 220},
	{"Nour", "non_binary", 29, "Paris", 48.8467, 2.3708, "Voyageur·se curieux·se, photographe à mes heures.", []int16{1, 9, 17}, []string{"man", "woman", "non_binary"}, 250},
	{"Clara", "woman", 36, "Créteil", 48.7904, 2.4556, "Bénévole, amoureuse des animaux et du bon vin.", []int16{24, 17, 16}, []string{"man"}, 280},
	{"Antoine", "man", 38, "Paris", 48.8584, 2.2945, "Ingénieur le jour, joueur de société la nuit.", []int16{19, 12, 11}, []string{"woman"}, 310},
	{"Jade", "woman", 24, "Paris", 48.8719, 2.3317, "Concerts, mode seconde main et longues balades.", []int16{14, 20, 3}, []string{"man", "woman"}, 340},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "demo-password-123"
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
	store, err := storage.NewDisk(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.Log{})
	profSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	photoSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), store)

	created := 0
	for i, u := range demoUsers {
		email := fmt.Sprintf("%s@%s", strings.ToLower(asciiName(u.name)), demoDomain)
		_, user, err := authSvc.Register(ctx, email, password, "seed", "")
		if err != nil {
			log.Printf("skip %s: %v", email, err)
			continue
		}
		birth := time.Now().AddDate(-u.age, 0, -(i*13)%300).Format("2006-01-02")
		if _, err := profSvc.Upsert(ctx, user.ID, profilesfeature.UpdateProfileRequest{
			FirstName: u.name, BirthDate: birth, Gender: u.gender, Bio: u.bio, City: u.city,
			Latitude: &u.lat, Longitude: &u.lng, InterestIDs: u.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if _, err := profSvc.SetPreferences(ctx, user.ID, profilesfeature.PreferencesRequest{InterestedIn: u.into, MinAge: 22, MaxAge: 45, MaxDistanceKm: 60}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		if _, err := photoSvc.Upload(ctx, user.ID, bytes.NewReader(placeholderPhoto(u.hue))); err != nil {
			log.Fatalf("photo %s: %v", email, err)
		}
		created++
	}
	fmt.Printf("seed done: %d demo accounts created (log in with <name>@%s / %s)\n", created, demoDomain, password)
}

func asciiName(s string) string {
	return strings.NewReplacer("è", "e", "é", "e", "ë", "e", "ï", "i", "î", "i", "ô", "o", "ç", "c", "à", "a", "·", "").Replace(s)
}

// placeholderPhoto draws a soft diagonal gradient; demo accounts have no real photos.
func placeholderPhoto(hue int) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / float64(w+h)
			img.Set(x, y, color.RGBA{
				R: uint8(float64(hue%256)*0.3 + 120*(1-t) + 40),
				G: uint8(90 + 110*t),
				B: uint8(200 - float64(hue%256)*0.2 - 90*t),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
