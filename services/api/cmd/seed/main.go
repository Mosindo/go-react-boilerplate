// Command seed loads clearly-labelled demo accounts for local development.
//
// Demo users use the reserved "@demo.invalid" domain (never deliverable, never a real person),
// share one documented password and are removed with `-purge`. The command refuses to run when
// APP_ENV=production.
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
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
)

const demoDomain = "demo.invalid"

type demoUser struct {
	Name         string
	Gender       profilesfeature.Gender
	Age          int
	Bio          string
	City         string
	Lat, Lon     float64
	InterestedIn []profilesfeature.Gender
	Interests    []int
	Hue          int
}

var demoUsers = []demoUser{
	{"Camille", profilesfeature.GenderWoman, 28, "Trail runner and amateur baker.", "Paris", 48.8566, 2.3522, []profilesfeature.Gender{profilesfeature.GenderMan}, []int{4, 9, 12}, 10},
	{"Hugo", profilesfeature.GenderMan, 31, "Cyclist, coffee snob, board-game host.", "Paris", 48.8700, 2.3300, []profilesfeature.Gender{profilesfeature.GenderWoman}, []int{3, 10, 24}, 200},
	{"Inès", profilesfeature.GenderWoman, 26, "Photographer who travels light.", "Lyon", 45.7640, 4.8357, []profilesfeature.Gender{profilesfeature.GenderMan, profilesfeature.GenderNonBinary}, []int{14, 13, 17}, 320},
	{"Léo", profilesfeature.GenderMan, 34, "Climbing, cinema and bad puns.", "Lyon", 45.7500, 4.8400, []profilesfeature.Gender{profilesfeature.GenderWoman}, []int{6, 19, 22}, 120},
	{"Sacha", profilesfeature.GenderNonBinary, 29, "Museums on Sundays, concerts on Fridays.", "Paris", 48.8500, 2.3600, []profilesfeature.Gender{profilesfeature.GenderWoman, profilesfeature.GenderMan, profilesfeature.GenderNonBinary}, []int{18, 21, 20}, 260},
	{"Nora", profilesfeature.GenderWoman, 32, "Yoga teacher, plant collector.", "Marseille", 43.2965, 5.3698, []profilesfeature.Gender{profilesfeature.GenderMan}, []int{5, 27, 26}, 60},
	{"Malo", profilesfeature.GenderMan, 27, "Developer by day, sailor on weekends.", "Marseille", 43.3000, 5.3800, []profilesfeature.Gender{profilesfeature.GenderWoman}, []int{29, 1, 8}, 180},
	{"Lou", profilesfeature.GenderWoman, 30, "Cooking for friends is my love language.", "Paris", 48.8600, 2.3400, []profilesfeature.Gender{profilesfeature.GenderMan, profilesfeature.GenderWoman}, []int{9, 11, 12}, 340},
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
	password := os.Getenv("DEMO_PASSWORD")
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
	blobs, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	authRepo := authfeature.NewPGRepository(pool)
	authSvc := authfeature.NewService(authRepo, []byte(cfg.JWTSecret), mail.LogMailer{}, blobs)

	if *purge {
		rows, err := pool.Query(ctx, `SELECT id FROM users WHERE email LIKE '%@' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		var ids []string
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			keys, err := authRepo.DeleteUser(ctx, id)
			if err != nil {
				log.Fatal(err)
			}
			for _, k := range keys {
				_ = blobs.Remove(k)
			}
		}
		fmt.Printf("removed %d demo accounts\n", len(ids))
		return
	}

	profilesSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), blobs, profilesSvc)

	created := 0
	for _, u := range demoUsers {
		email := strings.NewReplacer("ï", "i", "é", "e", "è", "e", "ë", "e", "ê", "e").Replace(strings.ToLower(u.Name)) + "@" + demoDomain
		tokens, user, err := authSvc.Register(ctx, email, password, "seed", "")
		if err != nil {
			fmt.Printf("skip %s: %v\n", email, err)
			continue
		}
		_ = tokens
		birth := time.Now().AddDate(-u.Age, 0, -1).Format("2006-01-02")
		if _, err := profilesSvc.Save(ctx, user.ID, profilesfeature.ProfileInput{
			FirstName: u.Name, BirthDate: birth, Gender: u.Gender, Bio: u.Bio, City: u.City, InterestIDs: u.Interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", u.Name, err)
		}
		if _, err := profilesSvc.SaveLocation(ctx, user.ID, profilesfeature.LocationInput{Latitude: u.Lat, Longitude: u.Lon}); err != nil {
			log.Fatalf("location %s: %v", u.Name, err)
		}
		if _, err := profilesSvc.SavePreferences(ctx, user.ID, profilesfeature.Preferences{InterestedIn: u.InterestedIn, MinAge: 18, MaxAge: 60, MaxDistanceKm: 100}); err != nil {
			log.Fatalf("preferences %s: %v", u.Name, err)
		}
		for shift := 0; shift < 2; shift++ {
			if _, err := photosSvc.Upload(ctx, user.ID, placeholderPhoto(u.Hue+shift*40)); err != nil {
				log.Fatalf("photo %s: %v", u.Name, err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo accounts (password %q, emails <firstname>@%s)\n", created, password, demoDomain)
}

// placeholderPhoto draws a soft diagonal gradient; demo data never contains real people's pictures.
func placeholderPhoto(hue int) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / float64(w+h)
			img.Set(x, y, hsv(float64(hue)+t*40, 0.45, 0.95-0.35*t))
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func hsv(h, s, v float64) color.RGBA {
	for h >= 360 {
		h -= 360
	}
	c := v * s
	x := c * (1 - abs(mod(h/60, 2)-1))
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
	return color.RGBA{R: uint8((r + m) * 255), G: uint8((g + m) * 255), B: uint8((b + m) * 255), A: 255}
}

func mod(a, b float64) float64 {
	for a >= b {
		a -= b
	}
	return a
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}
