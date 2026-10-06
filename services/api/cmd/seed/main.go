// Command seed fills a development database with clearly fake demo accounts.
//
// Safety rails: it refuses to run when APP_ENV=production, every account uses
// the reserved, undeliverable domain @demo.invalid, and the "demo" accounts can
// be removed again with `seed -purge`. Real users can never be confused with
// them.
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
	matchingfeature "example.com/api/internal/features/matching"
	"example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
)

const (
	demoDomain   = "@demo.invalid"
	demoPassword = "Demo-password-123"
)

type demoUser struct {
	slug         string
	name         string
	gender       string
	interestedIn []string
	age          int
	bio          string
	city         string
	lat, lng     float64
	interests    []string
	likesDemo    bool
}

var demoUsers = []demoUser{
	{"demo", "Démo", "woman", []string{"man", "woman", "non_binary"}, 30, "Compte de démonstration : connectez-vous avec celui-ci pour tester l'application.", "Paris", 48.8566, 2.3522, []string{"voyage", "cuisine", "musique"}, false},
	{"camille", "Camille", "woman", []string{"man"}, 28, "Randonnée le week-end, ciné le mardi.", "Paris", 48.8647, 2.3490, []string{"randonnee", "cinema", "cafe"}, true},
	{"julien", "Julien", "man", []string{"woman"}, 32, "Cuisinier amateur, grand mangeur de fromage.", "Paris", 48.8530, 2.3499, []string{"cuisine", "vin", "voyage"}, true},
	{"sarah", "Sarah", "woman", []string{"man", "woman"}, 26, "Photographe, toujours un appareil dans le sac.", "Montreuil", 48.8610, 2.4430, []string{"photographie", "art", "festivals"}, true},
	{"thomas", "Thomas", "man", []string{"woman", "man"}, 35, "Développeur le jour, joueur de société la nuit.", "Paris", 48.8700, 2.3300, []string{"jeux-de-societe", "technologie", "lecture"}, false},
	{"lea", "Léa", "woman", []string{"man"}, 24, "Danse, yoga et thé vert.", "Boulogne-Billancourt", 48.8352, 2.2410, []string{"danse", "yoga", "nature"}, false},
	{"nina", "Nina", "non_binary", []string{"woman", "man", "non_binary"}, 29, "Théâtre, bénévolat, balades à vélo.", "Paris", 48.8800, 2.3550, []string{"theatre", "benevolat", "nature"}, false},
	{"hugo", "Hugo", "man", []string{"woman"}, 27, "Sport, musique et bonne humeur.", "Saint-Denis", 48.9362, 2.3574, []string{"sport", "musique", "festivals"}, false},
	{"emma", "Emma", "woman", []string{"man"}, 31, "Lectrice compulsive, amoureuse des chats.", "Versailles", 48.8014, 2.1301, []string{"lecture", "animaux", "jardinage"}, false},
	{"lucas", "Lucas", "man", []string{"woman", "man"}, 33, "Voyageur, toujours à la recherche du prochain vol pas cher.", "Paris", 48.8420, 2.3220, []string{"voyage", "photographie", "cafe"}, false},
	{"manon", "Manon", "woman", []string{"man", "woman"}, 25, "Jeux vidéo, cuisine japonaise.", "Nanterre", 48.8924, 2.2071, []string{"jeux-video", "cuisine", "art"}, false},
	{"antoine", "Antoine", "man", []string{"woman"}, 29, "Bricolage, vin et vieux vinyles.", "Vincennes", 48.8473, 2.4393, []string{"bricolage", "vin", "musique"}, false},
}

func main() {
	purge := flag.Bool("purge", false, "delete every demo account instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed refuses to run when APP_ENV=production")
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 15*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	if *purge {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo accounts (photo files are not purged: delete %s if you wish)\n", tag.RowsAffected(), cfg.UploadDir)
		return
	}

	blobs, err := storage.NewDisk(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	hub := realtime.NewHub()
	authService := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.LogMailer{})
	profileRepo := profilesfeature.NewPGRepository(pool)
	profileService := profilesfeature.NewService(profileRepo)
	photoService := photosfeature.NewService(photosfeature.NewPGRepository(pool), blobs, profileService)
	notifier := notifications.NewService(notifications.NewPGRepository(pool), hub)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(pool), profileRepo, matchingfeature.HeuristicRanker{}, notifier, hub)

	catalog, err := profileService.Catalog(ctx)
	if err != nil {
		log.Fatal(err)
	}
	interestID := map[string]int16{}
	for _, in := range catalog {
		interestID[in.Slug] = in.ID
	}

	ids := map[string]string{}
	for i, d := range demoUsers {
		email := d.slug + demoDomain
		_, user, err := authService.Register(ctx, email, demoPassword, "seed", "127.0.0.1")
		if err != nil {
			// Already seeded: look the id up instead.
			if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&user.ID); err != nil {
				log.Fatalf("seed %s: %v", email, err)
			}
			ids[d.slug] = user.ID
			continue
		}
		ids[d.slug] = user.ID

		birth := time.Now().AddDate(-d.age, 0, -7).Format("2006-01-02")
		must(profileService.SaveProfile(ctx, user.ID, profilesfeature.UpsertProfileRequest{FirstName: d.name, Birthdate: birth, Gender: d.gender, Bio: d.bio}))
		must(profileService.SaveLocation(ctx, user.ID, d.lat, d.lng, d.city))
		must(profileService.SavePreferences(ctx, user.ID, profilesfeature.PreferencesRequest{InterestedIn: d.interestedIn, MinAge: 18, MaxAge: 60, MaxDistanceKm: 50}))
		var picked []int16
		for _, slug := range d.interests {
			picked = append(picked, interestID[slug])
		}
		must(profileService.SaveInterests(ctx, user.ID, picked))
		for n := 0; n < 2; n++ {
			if _, err := photoService.Upload(ctx, user.ID, demoPhoto(i*2+n), ""); err != nil {
				log.Fatalf("seed photo: %v", err)
			}
		}
	}

	for _, d := range demoUsers {
		if d.likesDemo {
			if _, err := matchingService.Swipe(ctx, ids[d.slug], ids["demo"], matchingfeature.ActionLike); err != nil {
				log.Printf("like %s -> demo: %v", d.slug, err)
			}
		}
	}

	fmt.Printf("\nDemo data ready (%d accounts, all %s).\n", len(demoUsers), demoDomain)
	fmt.Printf("Sign in with  demo%s  /  %s\n", demoDomain, demoPassword)
	fmt.Println("Remove them any time with:  go run ./cmd/seed -purge")
	_ = os.Stdout.Sync()
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// demoPhoto draws a soft two-colour gradient: clearly synthetic, no real
// person's image is ever used.
func demoPhoto(n int) []byte {
	const w, h = 600, 800
	hue := float64(n*47%360) / 360
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			s := float64(x) / w
			r, g, b := hsv(math.Mod(hue+0.12*s, 1), 0.45+0.2*t, 0.95-0.35*t)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82})
	return buf.Bytes()
}

func hsv(h, s, v float64) (uint8, uint8, uint8) {
	i := math.Floor(h * 6)
	f := h*6 - i
	p, q, t := v*(1-s), v*(1-f*s), v*(1-(1-f)*s)
	var r, g, b float64
	switch int(i) % 6 {
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
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
