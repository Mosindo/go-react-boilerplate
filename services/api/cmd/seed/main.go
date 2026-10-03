// Command seed fills a development database with clearly fake demo members.
//
// Every demo account uses an @demo.invalid address (a reserved, undeliverable domain), so
// they can never be mistaken for real users, and `seed -purge` removes exactly those rows.
// It refuses to run when APP_ENV=production.
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
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/db"
)

const demoDomain = "@demo.invalid"

type member struct {
	name         string
	gender       string
	interestedIn []string
	age          int
	bio          string
	interests    []string
	dLat, dLng   float64 // offset from the seed center, in degrees
}

var members = []member{
	{"Camille", "woman", []string{"man"}, 29, "Trail runner and amateur baker.", []string{"running", "cooking", "travel"}, 0.02, 0.01},
	{"Louis", "man", []string{"woman"}, 32, "Vinyl, long walks and bad puns.", []string{"music", "hiking", "coffee"}, -0.03, 0.02},
	{"Inès", "woman", []string{"man", "woman"}, 27, "Photographer. Always chasing golden hour.", []string{"photography", "art", "travel"}, 0.05, -0.02},
	{"Hugo", "man", []string{"woman"}, 35, "Board games host, coffee snob.", []string{"board-games", "coffee", "cinema"}, 0.01, 0.06},
	{"Zoé", "woman", []string{"man"}, 31, "Climbing, yoga, and sourdough experiments.", []string{"yoga", "sports", "cooking"}, -0.04, -0.03},
	{"Malo", "man", []string{"woman", "nonbinary"}, 28, "Cyclist. Will beat you at Mario Kart.", []string{"cycling", "gaming", "music"}, 0.07, 0.04},
	{"Alix", "nonbinary", []string{"man", "woman", "nonbinary"}, 30, "Theatre kid grown up. Plant parent.", []string{"theatre", "nature", "reading"}, -0.01, -0.05},
	{"Jade", "woman", []string{"woman"}, 26, "Festival season is my favorite season.", []string{"festivals", "dancing", "music"}, 0.03, 0.08},
	{"Noah", "man", []string{"man"}, 33, "Cooking for friends every Sunday.", []string{"cooking", "wine", "pets"}, -0.06, 0.01},
	{"Maëlle", "woman", []string{"man", "woman"}, 34, "Volunteering, hiking, and bad karaoke.", []string{"volunteering", "hiking", "music"}, 0.04, -0.07},
	{"Théo", "man", []string{"woman"}, 30, "Software engineer who actually goes outside.", []string{"technology", "running", "coffee"}, -0.02, 0.07},
	{"Lina", "woman", []string{"man"}, 28, "Dog person. Museum regular.", []string{"pets", "art", "cinema"}, 0.06, 0.0},
}

// gradientJPEG makes a plain synthetic image: demo data never ships anybody's real photo.
func gradientJPEG(seed int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 640, 800))
	r0, g0, b0 := uint8(60+seed*37%160), uint8(80+seed*53%140), uint8(100+seed*29%120)
	for y := 0; y < 800; y++ {
		for x := 0; x < 640; x++ {
			t := float64(y) / 800
			img.Set(x, y, color.RGBA{R: uint8(float64(r0)*(1-t) + 240*t), G: uint8(float64(g0)*(1-t) + 200*t), B: uint8(float64(b0)*(1-t) + 190*t), A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func main() {
	purge := flag.Bool("purge", false, "delete every demo member instead of creating them")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("seed refuses to run with APP_ENV=production")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoPassword1!"
	}
	uploads := os.Getenv("UPLOADS_DIR")
	if uploads == "" {
		uploads = "./data/uploads"
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, databaseURL, 10*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	storage, err := photosfeature.NewLocalStorage(uploads)
	if err != nil {
		log.Fatal(err)
	}
	photos := photosfeature.NewService(photosfeature.NewPGRepository(pool), storage)

	if *purge {
		rows, err := pool.Query(ctx, `SELECT id FROM users WHERE email LIKE '%' || $1`, demoDomain)
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
			if err := photos.PurgeUserFiles(ctx, id); err != nil {
				log.Printf("purge files %s: %v", id, err)
			}
			if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
				log.Fatal(err)
			}
		}
		fmt.Printf("removed %d demo members\n", len(ids))
		return
	}

	centerLat, centerLng := envFloat("SEED_LAT", 48.8566), envFloat("SEED_LNG", 2.3522)
	secret := []byte(strings.Repeat("s", 32)) // sessions are not used by the seeder
	auth := authfeature.NewService(authfeature.NewPGRepository(pool), secret)
	profiles := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))

	created := 0
	for i, m := range members {
		email := fmt.Sprintf("%s%s", strings.ToLower(asciiFold(m.name)), demoDomain)
		_, user, err := auth.Register(ctx, email, password, "seed", "")
		if errors.Is(err, authfeature.ErrEmailExists) {
			continue
		}
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		lat, lng := centerLat+m.dLat, centerLng+m.dLng
		birth := time.Now().AddDate(-m.age, 0, -15).Format("2006-01-02")
		if _, err := profiles.Upsert(ctx, user.ID, profilesfeature.UpsertProfileRequest{
			FirstName: m.name, BirthDate: birth, Gender: m.gender, Bio: m.bio, City: "Demo City",
			Interests: m.interests, Latitude: &lat, Longitude: &lng,
		}); err != nil {
			log.Fatalf("profile %s: %v", m.name, err)
		}
		maxDist := 50
		if _, err := profiles.SetPreferences(ctx, user.ID, profilesfeature.PreferencesRequest{
			InterestedIn: m.interestedIn, MinAge: 18, MaxAge: 60, MaxDistanceKm: &maxDist,
		}); err != nil {
			log.Fatalf("preferences %s: %v", m.name, err)
		}
		if _, err := photos.Upload(ctx, user.ID, "", bytes.NewReader(gradientJPEG(i))); err != nil {
			log.Fatalf("photo %s: %v", m.name, err)
		}
		created++
	}
	fmt.Printf("created %d demo members (%d already existed)\n", created, len(members)-created)
	fmt.Printf("sign in with any <firstname>%s and password %q (set SEED_PASSWORD to change it)\n", demoDomain, password)
}

func envFloat(key string, fallback float64) float64 {
	var v float64
	if _, err := fmt.Sscanf(os.Getenv(key), "%f", &v); err != nil {
		return fallback
	}
	return v
}

// asciiFold keeps seeded emails ASCII-only.
func asciiFold(s string) string {
	r := strings.NewReplacer("é", "e", "è", "e", "ë", "e", "ï", "i", "î", "i", "ö", "o", "ô", "o", "à", "a", "ç", "c", "ü", "u")
	return r.Replace(strings.ToLower(s))
}
