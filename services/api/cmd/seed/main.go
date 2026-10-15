// Command seed fills a development database with clearly fake demo accounts.
//
// Demo accounts use the reserved ".invalid" email domain, so they can never
// belong to a real person, and the command refuses to run in production.
//
//	go run ./cmd/seed          # create (or complete) the demo accounts
//	go run ./cmd/seed --reset  # delete every demo account and its photos
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
	"time"

	authfeature "example.com/api/internal/features/auth"
	filesfeature "example.com/api/internal/features/files"
	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/blob"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	demoDomain   = "@demo.invalid"
	demoPassword = "DemoPass123!"
)

type demoUser struct {
	Slug, Name, Gender string
	Seeks              []string
	Age                int
	Bio                string
	Interests          []string
	Lat, Lng           float64
	City               string
	Hue                float64
}

// Everyone lives in or near Paris so a single demo login always has people to discover.
var demoUsers = []demoUser{
	{"camille", "Camille", "woman", []string{"man"}, 29, "Illustratrice, toujours un carnet dans le sac. Fan de marchés du dimanche.", []string{"art", "cafe", "velo"}, 48.8566, 2.3522, "Paris", 12},
	{"lucas", "Lucas", "man", []string{"woman"}, 32, "Cuisinier le jour, grimpeur le week-end. Je cherche quelqu'un pour tester mes recettes.", []string{"cuisine", "sport", "randonnee"}, 48.8606, 2.3376, "Paris", 210},
	{"ines", "Inès", "woman", []string{"man", "woman"}, 27, "Photographe. J'aime les trains de nuit et les villes qui s'éveillent.", []string{"photographie", "voyages", "musique"}, 48.8738, 2.2950, "Paris", 320},
	{"hugo", "Hugo", "man", []string{"woman"}, 35, "Développeur, joueur de jeux de société et lecteur compulsif.", []string{"jeux-de-societe", "lecture", "technologie"}, 48.8412, 2.3003, "Paris", 150},
	{"manon", "Manon", "woman", []string{"man"}, 26, "Danseuse amatrice, passionnée de cinéma d'auteur.", []string{"danse", "cinema", "theatre"}, 48.8800, 2.3550, "Paris", 280},
	{"theo", "Théo", "man", []string{"woman", "man"}, 30, "Prof de yoga et jardinier urbain. Calme, curieux, drôle (parait-il).", []string{"yoga", "jardinage", "nature"}, 48.8340, 2.3760, "Paris", 100},
	{"lea", "Léa", "woman", []string{"man"}, 31, "Médecin, marathonienne, grande mangeuse de pâtes.", []string{"course-a-pied", "cuisine", "voyages"}, 48.9000, 2.2500, "Nanterre", 40},
	{"nathan", "Nathan", "man", []string{"woman"}, 28, "Musicien dans un groupe de folk. Concerts, cafés et longues balades.", []string{"musique", "concerts", "cafe"}, 48.8000, 2.4000, "Vincennes", 250},
	{"jade", "Jade", "woman", []string{"man", "woman"}, 24, "Étudiante en archi. Je dessine des maisons et je rêve d'en habiter une.", []string{"art", "mode", "photographie"}, 48.8500, 2.4400, "Montreuil", 340},
	{"adam", "Adam", "man", []string{"woman"}, 38, "Papa de deux chats, amoureux de randonnées et de bon vin.", []string{"randonnee", "vin", "animaux"}, 48.9500, 2.3000, "Saint-Denis", 190},
	{"sasha", "Sasha", "nonbinary", []string{"man", "woman", "nonbinary"}, 29, "Artiste visuel·le, bricoleur·se, fan de brocantes.", []string{"bricolage", "art", "mode"}, 48.8650, 2.3800, "Paris", 60},
	{"eva", "Eva", "woman", []string{"man"}, 33, "Journaliste. Curieuse de tout, surtout des gens.", []string{"lecture", "voyages", "benevolat"}, 48.8450, 2.3500, "Paris", 300},
	{"malo", "Malo", "man", []string{"woman"}, 27, "Skipper breton exilé à Paris. Je cherche un second équipage.", []string{"sport", "nature", "musique"}, 48.8700, 2.3100, "Paris", 220},
	{"zoe", "Zoé", "woman", []string{"man"}, 25, "Graphiste. Playlists, plantes vertes et petits restos.", []string{"musique", "jardinage", "cuisine"}, 48.8300, 2.3200, "Paris", 130},
}

func main() {
	reset := flag.Bool("reset", false, "delete every demo account instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
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

	store, err := blob.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	photos := filesfeature.NewService(filesfeature.NewPGRepository(pool), store, []byte(cfg.JWTSecret))
	profiles := usersfeature.NewService(usersfeature.NewPGRepository(pool), photos)
	authService := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret))
	authService.SetAccountHooks(hooks{photos})

	if *reset {
		if err := deleteDemoUsers(ctx, pool, authService); err != nil {
			log.Fatal(err)
		}
		return
	}

	created := 0
	for _, u := range demoUsers {
		ok, err := seedUser(ctx, authService, profiles, photos, u)
		if err != nil {
			log.Fatalf("seed %s: %v", u.Slug, err)
		}
		if ok {
			created++
		}
	}
	fmt.Printf("Demo data ready: %d new accounts (%d total).\n", created, len(demoUsers))
	fmt.Printf("Sign in with any demo email, e.g. camille%s / %s\n", demoDomain, demoPassword)
}

type hooks struct{ photos *filesfeature.Service }

func (h hooks) BeforeDelete(ctx context.Context, userID string) (func(), error) {
	return h.photos.BeforeDelete(ctx, userID)
}

func seedUser(ctx context.Context, auth *authfeature.Service, profiles *usersfeature.Service, photos *filesfeature.Service, u demoUser) (bool, error) {
	email := u.Slug + demoDomain
	_, user, err := auth.Register(ctx, email, demoPassword, "seed", "")
	if errors.Is(err, authfeature.ErrEmailExists) {
		return false, nil // already seeded
	}
	if err != nil {
		return false, err
	}

	birth := fmt.Sprintf("%d-03-14", time.Now().Year()-u.Age-1)
	if err := profiles.SaveProfile(ctx, user.ID, usersfeature.ProfileInput{
		FirstName: u.Name, BirthDate: birth, Gender: u.Gender, Bio: u.Bio, Interests: u.Interests,
	}); err != nil {
		return false, err
	}
	if err := profiles.SavePreferences(ctx, user.ID, usersfeature.PreferencesInput{
		InterestedIn: u.Seeks, MinAge: 21, MaxAge: 45, MaxDistanceKm: 40,
	}); err != nil {
		return false, err
	}
	lat, lng := u.Lat, u.Lng
	if err := profiles.SetLocation(ctx, user.ID, usersfeature.LocationInput{Latitude: &lat, Longitude: &lng, City: u.City}); err != nil {
		return false, err
	}
	for i := 0; i < 2; i++ {
		if _, err := photos.Upload(ctx, user.ID, avatar(u.Hue+float64(i)*35)); err != nil {
			return false, err
		}
	}
	return true, nil
}

// deleteDemoUsers removes demo accounts through the same path as a user-requested
// deletion, so photo files and every related row go with them.
func deleteDemoUsers(ctx context.Context, pool *pgxpool.Pool, auth *authfeature.Service) error {
	rows, err := pool.Query(ctx, `SELECT id FROM users WHERE email LIKE '%' || $1`, demoDomain)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	for _, id := range ids {
		if err := auth.DeleteAccount(ctx, id, demoPassword); err != nil {
			return fmt.Errorf("delete demo user %s: %w", id, err)
		}
	}
	fmt.Printf("Deleted %d demo accounts.\n", len(ids))
	return nil
}

// avatar draws an abstract gradient portrait placeholder (no real people, no text).
func avatar(hue float64) []byte {
	const w, h = 720, 960
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			r, g, b := hsl(hue+t*40, 0.55, 0.78-0.35*t)
			// A soft disc suggests a head and shoulders.
			dx, dy := float64(x-w/2), float64(y-h*38/100)
			if d := math.Hypot(dx, dy); d < 150 {
				r, g, b = hsl(hue+20, 0.45, 0.9-d/900)
			}
			if y > h*62/100 && math.Abs(float64(x-w/2)) < 130+float64(y-h*62/100)*0.5 {
				r, g, b = hsl(hue+10, 0.5, 0.86)
			}
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 360) / 360
	if h < 0 {
		h++
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	conv := func(t float64) uint8 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		var v float64
		switch {
		case t < 1.0/6:
			v = p + (q-p)*6*t
		case t < 0.5:
			v = q
		case t < 2.0/3:
			v = p + (q-p)*(2.0/3-t)*6
		default:
			v = p
		}
		return uint8(math.Round(v * 255))
	}
	return conv(h + 1.0/3), conv(h), conv(h - 1.0/3)
}
