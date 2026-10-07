// Command seed fills a DEVELOPMENT database with clearly fake demo accounts so the app
// can be explored with more than one person. Every account uses the reserved
// "@demo.invalid" domain (it can never receive mail or collide with a real user) and the
// command refuses to run when APP_ENV=production.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
)

const demoDomain = "demo.invalid"

var names = []struct{ name, gender string }{
	{"Alice", "woman"}, {"Bastien", "man"}, {"Camille", "nonbinary"}, {"David", "man"}, {"Emma", "woman"},
	{"Farid", "man"}, {"Gaëlle", "woman"}, {"Hugo", "man"}, {"Inès", "woman"}, {"Jules", "man"},
	{"Kenza", "woman"}, {"Louis", "man"}, {"Maya", "woman"}, {"Noah", "man"}, {"Océane", "woman"},
	{"Paul", "man"}, {"Quinn", "nonbinary"}, {"Rania", "woman"}, {"Samuel", "man"}, {"Tess", "woman"},
}

var bios = []string{
	"Sunday markets, long walks and bad puns.",
	"Looking for someone to share good coffee and better stories.",
	"Weekend hiker, weekday dreamer.",
	"I cook for fun and eat for joy.",
	"Always planning the next trip.",
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
	if len(password) < 8 {
		log.Fatal("set SEED_PASSWORD (min 8 chars): the shared password of the demo accounts")
	}
	lat := envFloat("SEED_LAT", 48.8566)
	lng := envFloat("SEED_LNG", 2.3522)

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	signer := media.NewSigner([]byte(cfg.JWTSecret))
	storage, err := photosfeature.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	photos := photosfeature.NewService(photosfeature.NewRepository(pool), storage, signer)
	profiles := profilesfeature.NewService(profilesfeature.NewRepository(pool), signer)
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.LogSender{}, photos)
	interests, err := profiles.Interests(ctx)
	if err != nil {
		log.Fatal(err)
	}

	rng := rand.New(rand.NewSource(42))
	created := 0
	for i, n := range names {
		email := fmt.Sprintf("%s@%s", strings.ToLower(strings.NewReplacer("ë", "e", "è", "e", "ï", "i", "ê", "e", "é", "e", "Océane", "oceane").Replace(n.name))+strconv.Itoa(i), demoDomain)
		_, user, err := authSvc.Register(ctx, email, password, "seed", "")
		if err != nil {
			log.Printf("skip %s: %v", email, err)
			continue
		}
		picks := make([]string, 0, 4)
		for _, idx := range rng.Perm(len(interests))[:3+rng.Intn(3)] {
			picks = append(picks, interests[idx].Slug)
		}
		if _, err := profiles.Upsert(ctx, user.ID, profilesfeature.UpsertProfileRequest{
			FirstName: n.name,
			BirthDate: fmt.Sprintf("%d-%02d-%02d", 1984+rng.Intn(18), 1+rng.Intn(12), 1+rng.Intn(28)),
			Gender:    n.gender,
			Bio:       bios[rng.Intn(len(bios))],
			City:      "Demo City",
			Interests: picks,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if err := profiles.SetLocation(ctx, user.ID, lat+(rng.Float64()-0.5)*0.2, lng+(rng.Float64()-0.5)*0.3); err != nil {
			log.Fatal(err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photos.Upload(ctx, user.ID, gradient(i*37+p*90)); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		created++
	}
	fmt.Printf("seeded %d demo accounts (*@%s) with the password from SEED_PASSWORD\n", created, demoDomain)
}

func envFloat(key string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil {
		return v
	}
	return def
}

// gradient renders a simple abstract placeholder picture (no real faces).
func gradient(hue int) []byte {
	const w, h = 480, 640
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((hue + x*255/w) % 256), G: uint8((hue*2 + y*255/h) % 256), B: uint8(200 - y*120/h), A: 255,
			})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
