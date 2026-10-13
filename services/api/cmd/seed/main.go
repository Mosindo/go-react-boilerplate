// Command seed creates clearly-labelled demo accounts for local development.
//
// Every demo account uses an email ending in @seed.invalid (a reserved,
// undeliverable domain) so it can never be mistaken for a real user, and the
// command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed          # create demo accounts (idempotent)
//	go run ./cmd/seed -reset   # delete every @seed.invalid account
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/db"
)

const (
	seedDomain   = "@seed.invalid"
	seedPassword = "DemoPass123"
)

type demoUser struct {
	Name, Gender string
	Age          int
	Interested   []string
	Bio          string
	Interests    []int
	Lat, Lon     float64
	City         string
}

var demoUsers = []demoUser{
	{"Camille", "woman", 29, []string{"man"}, "Chasseuse de bons cafés et de sentiers peu fréquentés.", []int{13, 7, 5}, 48.8566, 2.3522, "Paris"},
	{"Hugo", "man", 32, []string{"woman"}, "Cuisine du dimanche, vélo le reste de la semaine.", []int{2, 20, 3}, 48.8606, 2.3376, "Paris"},
	{"Inès", "woman", 27, []string{"man", "woman"}, "Photographe amateure, toujours un carnet dans le sac.", []int{9, 8, 1}, 48.8738, 2.2950, "Paris"},
	{"Léo", "man", 35, []string{"woman"}, "Ingénieur le jour, joueur de jeux de société la nuit.", []int{22, 17, 14}, 48.8412, 2.3003, "Paris"},
	{"Zoé", "woman", 31, []string{"man"}, "Yoga au lever du soleil, festivals l'été.", []int{12, 19, 16}, 48.8920, 2.3440, "Paris"},
	{"Nathan", "man", 28, []string{"woman"}, "Grimpeur, lecteur, amateur de cinéma d'auteur.", []int{6, 5, 4}, 48.8300, 2.3700, "Paris"},
	{"Lina", "woman", 34, []string{"man"}, "Je cherche quelqu'un avec qui partir sans destination.", []int{1, 24, 7}, 48.8650, 2.3800, "Paris"},
	{"Adam", "man", 30, []string{"woman", "man"}, "Musicien, danseur maladroit mais enthousiaste.", []int{3, 11, 18}, 48.8500, 2.3200, "Paris"},
	{"Maëlle", "woman", 26, []string{"man"}, "Bénévole, amoureuse des chiens et des vieux livres.", []int{23, 15, 5}, 48.8700, 2.3100, "Paris"},
	{"Sacha", "nonbinary", 29, []string{"woman", "man", "nonbinary"}, "Théâtre, randonnée et bons vins.", []int{18, 7, 14}, 48.8800, 2.3600, "Paris"},
	{"Théo", "man", 38, []string{"woman"}, "Course à pied, cuisine italienne, humour douteux.", []int{21, 2, 4}, 48.8450, 2.3900, "Paris"},
	{"Jade", "woman", 33, []string{"man"}, "Nature, photo et cafés qui s'éternisent.", []int{16, 9, 13}, 48.8780, 2.4000, "Paris"},
}

func main() {
	reset := flag.Bool("reset", false, "delete every demo account instead of creating them")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, url, 20*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	if *reset {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%' || $1`, seedDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("deleted %d demo accounts\n", tag.RowsAffected())
		return
	}

	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte("seed-command-does-not-issue-sessions-0000"), nil, nil)
	profilesSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), profilesSvc)

	created := 0
	for i, d := range demoUsers {
		email := fmt.Sprintf("demo%02d%s", i+1, seedDomain)
		_, user, err := authSvc.Register(ctx, email, seedPassword, "seed", "")
		if errors.Is(err, authfeature.ErrEmailExists) {
			continue
		}
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		city := d.City
		birth := time.Now().AddDate(-d.Age, 0, -10).Format("2006-01-02")
		if _, err := profilesSvc.Save(ctx, user.ID, profilesfeature.ProfileRequest{
			FirstName: d.Name, BirthDate: birth, Gender: d.Gender, Bio: d.Bio, City: &city, InterestIDs: d.Interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		dist := 50
		if _, err := profilesSvc.SavePreferences(ctx, user.ID, profilesfeature.PreferencesRequest{
			InterestedIn: d.Interested, MinAge: 18, MaxAge: 60, MaxDistanceKm: &dist,
		}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		lat, lon := d.Lat, d.Lon
		if _, err := profilesSvc.SetLocation(ctx, user.ID, profilesfeature.LocationRequest{Latitude: &lat, Longitude: &lon}); err != nil {
			log.Fatalf("location %s: %v", email, err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photosSvc.Upload(ctx, user.ID, bytes.NewReader(placeholderPhoto(i*2+p))); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		created++
	}

	// A few demo people already like demo01, so liking them back shows a match.
	_, err = pool.Exec(ctx, `
		INSERT INTO likes (from_user_id, to_user_id, action)
		SELECT f.id, t.id, 'like'
		FROM users t JOIN users f ON f.email IN ('demo02`+seedDomain+`', 'demo04`+seedDomain+`', 'demo06`+seedDomain+`')
		WHERE t.email = 'demo01`+seedDomain+`'
		ON CONFLICT DO NOTHING`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("created %d demo accounts (%d total available)\n", created, len(demoUsers))
	fmt.Printf("sign in with demo01%s … demo%02d%s, password %q\n", seedDomain, len(demoUsers), seedDomain, seedPassword)
	fmt.Println("demo02, demo04 and demo06 already like demo01: like them back to see a match")
}

// placeholderPhoto draws a soft two-colour gradient (no real people's images).
func placeholderPhoto(seed int) []byte {
	const w, h = 600, 800
	hue := float64(seed*47%360) / 360
	c1 := hsv(hue, 0.55, 0.95)
	c2 := hsv(math.Mod(hue+0.12, 1), 0.7, 0.55)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			u := t*0.8 + float64(x)/w*0.2
			img.Set(x, y, color.RGBA{
				uint8(float64(c1.R)*(1-u) + float64(c2.R)*u),
				uint8(float64(c1.G)*(1-u) + float64(c2.G)*u),
				uint8(float64(c1.B)*(1-u) + float64(c2.B)*u), 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsv(h, s, v float64) color.RGBA {
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
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}
