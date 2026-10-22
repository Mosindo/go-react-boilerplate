// Command seed fills a development database with clearly fake demo accounts.
//
// Every demo account uses the reserved domain "demo.invalid" (never deliverable)
// and the same well-known password, and the command refuses to run when
// APP_ENV=production. Run it with `go run ./cmd/seed`; it is idempotent.
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
	"math"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
)

const (
	demoDomain   = "demo.invalid"
	demoPassword = "demo-password-123"
)

type demoUser struct {
	name, gender string
	interestedIn []string
	age          int
	city         string
	lat, lon     float64
	bio          string
	interests    []int16
}

var demoUsers = []demoUser{
	{"Camille", "woman", []string{"man"}, 29, "Paris", 48.8566, 2.3522, "Chasseuse de bons cafés et de librairies.", []int16{1, 4, 9}},
	{"Lucas", "man", []string{"woman"}, 32, "Paris", 48.8738, 2.2950, "Cuisine le dimanche, court le reste du temps.", []int16{7, 10, 2}},
	{"Inès", "woman", []string{"man", "woman"}, 26, "Montreuil", 48.8610, 2.4430, "Photo argentique et musées.", []int16{15, 17, 1}},
	{"Hugo", "man", []string{"woman"}, 35, "Boulogne-Billancourt", 48.8352, 2.2411, "Vélo, vin et jeux de société.", []int16{13, 8, 20}},
	{"Sofia", "woman", []string{"man"}, 31, "Saint-Denis", 48.9362, 2.3574, "Randonnée dès que possible.", []int16{6, 5, 25}},
	{"Mathis", "man", []string{"woman"}, 28, "Vincennes", 48.8470, 2.4390, "Concerts, cinéma et ciné-clubs.", []int16{3, 1, 2}},
	{"Léa", "woman", []string{"man"}, 27, "Paris", 48.8450, 2.3460, "Yoga le matin, théâtre le soir.", []int16{11, 18, 4}},
	{"Noah", "man", []string{"man", "woman"}, 30, "Versailles", 48.8014, 2.1301, "Développeur curieux, grand marcheur.", []int16{21, 6, 9}},
	{"Jade", "woman", []string{"woman"}, 33, "Paris", 48.8800, 2.3300, "Danse, voyages et langues.", []int16{24, 5, 27}},
	{"Adam", "man", []string{"woman"}, 34, "Nanterre", 48.8924, 2.2071, "Photographe amateur.", []int16{15, 25, 12}},
	{"Manon", "woman", []string{"man"}, 25, "Paris", 48.8650, 2.3800, "Animaux, nature, cuisine.", []int16{22, 25, 7}},
	{"Ethan", "man", []string{"woman"}, 37, "Créteil", 48.7904, 2.4556, "Natation et lecture.", []int16{14, 4, 2}},
	{"Alex", "non_binary", []string{"man", "woman", "non_binary"}, 29, "Paris", 48.8530, 2.3499, "Bénévole, joueur de jeux de société.", []int16{26, 20, 19}},
	{"Chloé", "woman", []string{"man"}, 30, "Lyon", 45.7640, 4.8357, "Lyonnaise, amatrice de bouchons.", []int16{7, 8, 9}},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo accounts when APP_ENV=production")
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

	storage, err := photosfeature.NewStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	authRepo := authfeature.NewPGRepository(pool)
	authService := authfeature.NewService(authRepo, []byte(cfg.JWTSecret), authfeature.Options{})
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(pool), storage)

	created := 0
	for i, d := range demoUsers {
		email := fmt.Sprintf("%s.%02d@%s", strings.ToLower(strings.NewReplacer("é", "e", "è", "e", "ï", "i", "ë", "e").Replace(d.name)), i+1, demoDomain)
		_, user, err := authService.Register(ctx, email, demoPassword, "seed", "")
		if errors.Is(err, authfeature.ErrEmailExists) {
			continue
		}
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		birth := time.Now().AddDate(-d.age, -2, -(i + 3)).Format("2006-01-02")
		if _, err := profilesService.UpdateSelf(ctx, user.ID, profilesfeature.UpdateProfileRequest{
			FirstName: d.name, Birthdate: birth, Gender: d.gender, Bio: d.bio, City: d.city,
			Latitude: d.lat, Longitude: d.lon, InterestIDs: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if _, err := profilesService.UpdatePreferences(ctx, user.ID, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: d.interestedIn, MinAge: 21, MaxAge: 50, MaxDistanceKm: 60,
		}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photosService.Upload(ctx, user.ID, bytes.NewReader(placeholderPhoto(i*2+p))); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		created++
	}
	fmt.Printf("seeded %d demo accounts (%d already existed)\n", created, len(demoUsers)-created)
	fmt.Printf("log in with <name>.<NN>@%s / %s, e.g. camille.01@%s\n", demoDomain, demoPassword, demoDomain)
}

// placeholderPhoto draws an abstract gradient: demo data must not contain real people's pictures.
func placeholderPhoto(seed int) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	hue := math.Mod(float64(seed)*47, 360)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y)/h*0.6 + float64(x)/w*0.4
			r, g, b := hsl(math.Mod(hue+t*60, 360), 0.55, 0.78-0.35*t)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) (uint8, uint8, uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
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
	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}
