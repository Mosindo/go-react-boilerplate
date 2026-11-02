// Command seed fills a development database with clearly labelled demo users.
//
// Every demo account uses the e-mail domain demo.tandem.local, so it can never
// be mistaken for (or collide with) a real user. The command refuses to run
// when APP_ENV=production. Run `go run ./cmd/seed -reset` to remove them.
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
	"example.com/api/internal/platform/db"
)

const (
	demoDomain   = "demo.tandem.local"
	demoPassword = "DemoPass123"
)

type demo struct {
	name, gender string
	age          int
	interested   []string
	bio          string
	interests    []int
	dLat, dLng   float64
}

var people = []demo{
	{"Camille", "woman", 28, []string{"man"}, "Fan de randonnées et de cafés lents.", []int{5, 14, 17}, 0.01, 0.02},
	{"Louis", "man", 31, []string{"woman"}, "Cuisine le dimanche, court le lundi.", []int{4, 6, 2}, -0.02, 0.01},
	{"Inès", "woman", 26, []string{"man", "woman"}, "Photographe amateure, voyageuse.", []int{8, 1, 9}, 0.03, -0.02},
	{"Hugo", "man", 34, []string{"woman"}, "Guitare, ciné et bonne humeur.", []int{2, 3, 7}, 0.02, 0.04},
	{"Léa", "woman", 24, []string{"man"}, "Yoga le matin, festivals l'été.", []int{12, 19, 11}, -0.03, -0.01},
	{"Nathan", "man", 29, []string{"woman", "man"}, "Gamer et amoureux de la montagne.", []int{10, 5, 16}, 0.04, 0.03},
	{"Manon", "woman", 32, []string{"man"}, "Théâtre, lecture, vin naturel.", []int{18, 7, 15}, -0.01, 0.05},
	{"Théo", "man", 27, []string{"woman"}, "Technophile, cuisinier du dimanche.", []int{16, 4, 20}, 0.05, -0.03},
	{"Sarah", "woman", 30, []string{"man"}, "Danse, voyages et chiens.", []int{11, 1, 13}, -0.04, 0.02},
	{"Yanis", "man", 33, []string{"woman"}, "Sport, musique et cafés.", []int{6, 2, 14}, 0.0, -0.05},
	{"Alex", "non_binary", 29, []string{"woman", "man", "non_binary"}, "Curieux·se de tout.", []int{9, 8, 20}, 0.02, 0.0},
	{"Julie", "woman", 35, []string{"man"}, "Nature, animaux, bons restos.", []int{17, 13, 4}, -0.02, -0.04},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo users and exit")
	lat := flag.Float64("lat", 48.8566, "centre latitude of the demo area")
	lng := flag.Float64("lng", 2.3522, "centre longitude of the demo area")
	flag.Parse()

	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("seed: DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, url, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	if *reset {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo users\n", tag.RowsAffected())
		return
	}

	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte("seed-only-not-used-for-signing-0123456789"))
	profSvc := profilesfeature.NewService(profilesfeature.NewRepository(pool))
	photoRepo := photosfeature.NewRepository(pool)

	ids := make([]string, len(people))
	for i, p := range people {
		email := fmt.Sprintf("demo%d@%s", i+1, demoDomain)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
		_, user, err := authSvc.Register(ctx, email, demoPassword, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		ids[i] = user.ID
		birth := time.Now().AddDate(-p.age, -2, -3).Format("2006-01-02")
		la, lo := *lat+p.dLat, *lng+p.dLng
		if _, err := profSvc.Upsert(ctx, user.ID, profilesfeature.UpsertProfileRequest{
			FirstName: p.name, BirthDate: birth, Gender: p.gender, Bio: p.bio, City: "Démo-Ville",
			Latitude: &la, Longitude: &lo, InterestIDs: p.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if _, err := profSvc.UpdatePreferences(ctx, user.ID, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: p.interested, MinAge: 18, MaxAge: 60, MaxDistanceKm: 50,
		}); err != nil {
			log.Fatalf("prefs %s: %v", email, err)
		}
		for k := 0; k < 2; k++ {
			processed, err := photosfeature.Process(bytes.NewReader(gradient(i*37+k*90, 600, 800)))
			if err != nil {
				log.Fatal(err)
			}
			if _, err := photoRepo.Add(ctx, user.ID, processed, photosfeature.MaxPhotos); err != nil {
				log.Fatal(err)
			}
		}
	}
	// Some demo profiles already like demo1 (Camille) and demo2 (Louis) so that
	// a fresh test account matches as soon as it likes them back.
	for _, pair := range [][2]int{{1, 0}, {3, 0}, {7, 0}, {9, 0}, {0, 1}, {2, 1}, {4, 1}, {6, 1}} {
		_, err := pool.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, ids[pair[0]], ids[pair[1]])
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("seeded %d demo users around (%.4f, %.4f)\n", len(people), *lat, *lng)
	fmt.Printf("log in with demoN@%s / %s (N = 1..%d)\n", demoDomain, demoPassword, len(people))
}

// gradient renders a deterministic abstract picture so demo photos need no assets.
func gradient(hue, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / float64(h)
			a := float64(hue%360) * math.Pi / 180
			r := 130 + 100*math.Sin(a+t*2)
			g := 130 + 100*math.Sin(a+2.1+t*2+float64(x)/float64(w))
			b := 130 + 100*math.Sin(a+4.2+float64(x)/float64(w)*2)
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}
