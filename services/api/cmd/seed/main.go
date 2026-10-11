// Command seed fills a DEVELOPMENT database with clearly-labelled demo accounts.
//
//	SEED_CONFIRM=yes go run ./cmd/seed          # create demo users
//	SEED_CONFIRM=yes go run ./cmd/seed -purge   # remove them again
//
// Demo accounts use the reserved e-mail domain @demo.invalid and a "[DÉMO]" bio prefix so they can never be
// mistaken for (or collide with) real users. The command refuses to run when APP_ENV=production.
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
	"os"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/signedurl"
)

const (
	demoDomain   = "@demo.invalid"
	demoPassword = "demo-password-123"
)

type demoUser struct {
	slug, name, gender, birth, city string
	lat, lng                        float64
	into                            []string
	interests                       []string
	hue                             int
}

var demoUsers = []demoUser{
	{"lea", "Léa", "woman", "1994-03-12", "Paris", 48.8566, 2.3522, []string{"man"}, []string{"voyage", "cinema", "yoga"}, 10},
	{"camille", "Camille", "woman", "1991-07-02", "Paris", 48.8750, 2.3000, []string{"man", "woman"}, []string{"lecture", "cafe", "art"}, 40},
	{"nina", "Nina", "woman", "1997-11-23", "Versailles", 48.8049, 2.1204, []string{"man"}, []string{"musique", "danse", "concerts"}, 80},
	{"sarah", "Sarah", "woman", "1989-05-30", "Lyon", 45.7640, 4.8357, []string{"man"}, []string{"cuisine", "vin", "randonnee"}, 120},
	{"manon", "Manon", "woman", "1999-01-17", "Marseille", 43.2965, 5.3698, []string{"woman", "man"}, []string{"plage", "photo", "running"}, 160},
	{"hugo", "Hugo", "man", "1992-09-08", "Paris", 48.8600, 2.3400, []string{"woman"}, []string{"sport", "cinema", "technologie"}, 200},
	{"theo", "Théo", "man", "1995-02-14", "Paris", 48.8400, 2.3700, []string{"woman", "man"}, []string{"musique", "velo", "cafe"}, 230},
	{"lucas", "Lucas", "man", "1988-12-01", "Lyon", 45.7500, 4.8500, []string{"woman"}, []string{"cuisine", "montagne", "jeux-de-societe"}, 260},
	{"jules", "Jules", "man", "1998-06-21", "Marseille", 43.3000, 5.4000, []string{"woman"}, []string{"plage", "jeux-video", "voyage"}, 290},
	{"alex", "Alex", "non_binary", "1993-04-04", "Paris", 48.8700, 2.3200, []string{"man", "woman", "non_binary"}, []string{"art", "theatre", "meditation"}, 320},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo accounts instead of creating them")
	flag.Parse()

	if os.Getenv("SEED_CONFIRM") != "yes" {
		log.Fatal("refusing to run: set SEED_CONFIRM=yes to confirm you are targeting a development database")
	}
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

	if *purge {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo accounts\n", tag.RowsAffected())
		return
	}

	secret := []byte(cfg.JWTSecret)
	signer := signedurl.New(secret)
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), secret, mail.LogMailer{}, cfg.AppLinkBase)
	usersSvc := usersfeature.NewService(usersfeature.NewPGRepository(pool), signer)
	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), signer)

	interestIDs := map[string]int{}
	list, err := usersSvc.Interests(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, in := range list {
		interestIDs[in.Slug] = in.ID
	}

	created := 0
	for _, u := range demoUsers {
		email := u.slug + demoDomain
		tokens, user, err := authSvc.Register(ctx, email, demoPassword, "seed", "")
		if err != nil {
			if errors.Is(err, authfeature.ErrEmailExists) {
				fmt.Printf("skip %s (already exists)\n", email)
				continue
			}
			log.Fatalf("register %s: %v", email, err)
		}
		_ = tokens
		var ids []int
		for _, slug := range u.interests {
			ids = append(ids, interestIDs[slug])
		}
		if _, err := usersSvc.SaveProfile(ctx, user.ID, usersfeature.ProfileInput{
			FirstName: u.name, BirthDate: u.birth, Gender: u.gender, City: u.city, InterestIDs: ids,
			Bio: "[DÉMO] Compte de démonstration — ne correspond à aucune vraie personne.",
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		d := 100
		if _, err := usersSvc.SavePreferences(ctx, user.ID, usersfeature.Preferences{InterestedIn: u.into, MinAge: 18, MaxAge: 60, MaxDistanceKm: &d}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		if err := usersSvc.SetLocation(ctx, user.ID, u.lat, u.lng); err != nil {
			log.Fatalf("location %s: %v", email, err)
		}
		if _, err := photosSvc.Add(ctx, user.ID, placeholderPNG(u.hue)); err != nil {
			log.Fatalf("photo %s: %v", email, err)
		}
		created++
		fmt.Printf("created %s\n", email)
	}
	fmt.Printf("\n%d demo accounts created. Password for all of them: %s\n", created, demoPassword)
}

// placeholderPNG renders a plain gradient so demo profiles have a photo without shipping any real picture.
func placeholderPNG(hue int) []byte {
	const w, h = 480, 640
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8((hue + x/4) % 256), G: uint8(60 + y/5), B: uint8(200 - x/5), A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
