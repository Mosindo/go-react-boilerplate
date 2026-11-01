// Commande de seed: cree des comptes de DEMONSTRATION (emails en @demo.invalid,
// domaine reserve qui ne peut jamais recevoir de courrier reel).
//
//	SEED_CONFIRM=yes go run ./cmd/seed
//
// Refuse de s'executer si APP_ENV=production.
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
	"strings"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
)

type demoUser struct {
	name, gender, birth string
	interestedIn        []string
	city                string
	lat, lng            float64
	interests           []string
	bio                 string
	hue                 float64
}

var demoUsers = []demoUser{
	{"Camille", "woman", "1995-04-12", []string{"man"}, "Paris", 48.8566, 2.3522, []string{"music", "travel", "coffee"}, "Toujours un livre dans le sac et un billet de train pour le week-end.", 12},
	{"Hugo", "man", "1992-09-30", []string{"woman"}, "Paris", 48.8738, 2.2950, []string{"hiking", "cooking", "photography"}, "Cuisinier amateur, randonneur du dimanche.", 210},
	{"Inès", "woman", "1997-01-22", []string{"man", "woman"}, "Montreuil", 48.8634, 2.4411, []string{"art", "dance", "festivals"}, "Danse, expos et concerts en plein air.", 300},
	{"Louis", "man", "1990-06-18", []string{"woman"}, "Versailles", 48.8014, 2.1301, []string{"sport", "tech", "boardgames"}, "Développeur le jour, joueur de jeux de société le soir.", 140},
	{"Sofia", "woman", "1993-11-05", []string{"man"}, "Paris", 48.8499, 2.3792, []string{"yoga", "nature", "reading"}, "Yoga au lever du soleil, polars le soir.", 80},
	{"Malo", "man", "1996-03-14", []string{"woman", "man"}, "Boulogne-Billancourt", 48.8352, 2.2399, []string{"cinema", "music", "gaming"}, "Cinéphile et collectionneur de vinyles.", 250},
	{"Nina", "nonbinary", "1994-08-27", []string{"man", "woman", "nonbinary"}, "Paris", 48.8826, 2.3387, []string{"theatre", "art", "wine"}, "Théâtre, bons vins et longues discussions.", 330},
	{"Adam", "man", "1991-12-02", []string{"woman"}, "Saint-Denis", 48.9362, 2.3574, []string{"sport", "travel", "pets"}, "Fan de course à pied, propriétaire d'un border collie.", 30},
	{"Jade", "woman", "1998-07-09", []string{"man"}, "Paris", 48.8412, 2.3003, []string{"photography", "coffee", "nature"}, "Photographe amatrice, chasseuse de lumière.", 180},
	{"Léo", "man", "1994-02-19", []string{"woman"}, "Créteil", 48.7904, 2.4556, []string{"cooking", "festivals", "music"}, "Toujours partant pour un nouveau resto.", 50},
	{"Maya", "woman", "1992-10-10", []string{"man", "woman"}, "Paris", 48.8606, 2.3376, []string{"reading", "art", "theatre"}, "Libraire, curieuse de tout.", 270},
	{"Noé", "man", "1999-05-25", []string{"woman", "nonbinary"}, "Paris", 48.8530, 2.3499, []string{"gaming", "tech", "boardgames"}, "Étudiant en informatique, amateur de café.", 110},
}

func main() {
	if os.Getenv("SEED_CONFIRM") != "yes" {
		log.Fatal("refusing to seed: set SEED_CONFIRM=yes (creates DEMO accounts @demo.invalid)")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "Demo-Passw0rd"
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

	storage, err := photosfeature.NewDiskStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), storage)
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.Log{}, cfg.AppBaseURL, photosSvc)
	profSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))

	created := 0
	for _, d := range demoUsers {
		email := strings.ToLower(strings.ReplaceAll(d.name, "é", "e")) + "@demo.invalid"
		email = strings.NewReplacer("è", "e", "ï", "i", "ë", "e").Replace(email)
		_, user, err := authSvc.Register(ctx, email, password, "seed", "")
		if errors.Is(err, authfeature.ErrEmailExists) {
			fmt.Printf("skip   %s (exists)\n", email)
			continue
		}
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		if _, err := profSvc.SaveProfile(ctx, user.ID, profilesfeature.ProfileInput{
			FirstName: d.name, BirthDate: d.birth, Gender: d.gender, Bio: d.bio, City: d.city,
			Latitude: d.lat, Longitude: d.lng, Interests: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		s := profilesfeature.DefaultSettings()
		s.InterestedIn = d.interestedIn
		s.MaxDistanceKm = 100
		if _, err := profSvc.SaveSettings(ctx, user.ID, s); err != nil {
			log.Fatalf("settings %s: %v", email, err)
		}
		for i := 0; i < 2; i++ {
			if _, err := photosSvc.Add(ctx, user.ID, gradientPNG(d.hue+float64(i)*35), ""); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		created++
		fmt.Printf("create %s\n", email)
	}
	fmt.Printf("done: %d demo accounts created (password: %s)\n", created, password)
}

// gradientPNG genere une image degradee (aucun asset externe requis).
func gradientPNG(hue float64) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			img.Set(x, y, hsl(hue+float64(x)/w*40, 0.65, 0.78-0.35*t))
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
	h = float64(int(h*100)%36000) / 100
	c := (1 - abs(2*l-1)) * s
	x := c * (1 - abs(mod(h/60, 2)-1))
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
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func mod(a, b float64) float64 {
	for a >= b {
		a -= b
	}
	return a
}
