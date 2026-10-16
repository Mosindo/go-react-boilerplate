// Command seed fills a development database with clearly labelled demo accounts so the
// app can be explored without inviting friends first.
//
// Demo users all use the reserved domain demo.lien.test, share one well-known password,
// and the command refuses to run when APP_ENV=production. `-reset` removes them again.
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

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/mediaurl"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	demoDomain   = "demo.lien.test"
	demoPassword = "DemoPass123!"
)

type demoUser struct {
	name      string
	gender    string
	seeking   []string
	birthYear int
	bio       string
	job       string
	city      string
	lat, lon  float64
	interests []string
	hue       int
}

var demoUsers = []demoUser{
	{"Camille", "woman", []string{"man"}, 1996, "Toujours partante pour une randonnée ou un bon film.", "Architecte", "Paris", 48.8566, 2.3522, []string{"hiking", "cinema", "travel"}, 10},
	{"Léa", "woman", []string{"man"}, 1993, "Café, livres et longues balades.", "Éditrice", "Paris", 48.8738, 2.2950, []string{"reading", "coffee", "museums"}, 40},
	{"Inès", "woman", []string{"man", "woman"}, 1998, "Danse, cuisine et bonne humeur.", "Designer", "Montreuil", 48.8610, 2.4430, []string{"dancing", "cooking", "music"}, 80},
	{"Chloé", "woman", []string{"man"}, 1990, "Yoga le matin, vin le soir.", "Médecin", "Boulogne-Billancourt", 48.8397, 2.2399, []string{"yoga", "wine", "restaurants"}, 120},
	{"Sarah", "woman", []string{"woman"}, 1995, "Photographe, curieuse de tout.", "Photographe", "Paris", 48.8530, 2.3499, []string{"photography", "art", "travel"}, 160},
	{"Thomas", "man", []string{"woman"}, 1992, "Cycliste du dimanche et cuisinier du samedi.", "Ingénieur", "Paris", 48.8600, 2.3400, []string{"cycling", "cooking", "technology"}, 200},
	{"Hugo", "man", []string{"woman"}, 1994, "Football, séries et pizzas.", "Développeur", "Saint-Denis", 48.9362, 2.3574, []string{"football", "series", "gaming"}, 230},
	{"Nathan", "man", []string{"woman", "man"}, 1997, "Musique live avant tout.", "Musicien", "Paris", 48.8708, 2.3770, []string{"concerts", "music", "dancing"}, 260},
	{"Maxime", "man", []string{"woman"}, 1989, "Je cours, je lis, je voyage.", "Consultant", "Versailles", 48.8014, 2.1301, []string{"running", "reading", "travel"}, 290},
	{"Yanis", "man", []string{"man"}, 1996, "Escalade et jeux de société.", "Enseignant", "Paris", 48.8467, 2.3720, []string{"board_games", "fitness", "cinema"}, 320},
	{"Alex", "nonbinary", []string{"woman", "man", "nonbinary"}, 1995, "Nature, art et discussions sans fin.", "Illustrateur·rice", "Paris", 48.8584, 2.2945, []string{"nature", "art", "languages"}, 350},
	{"Jade", "woman", []string{"man", "nonbinary"}, 1999, "Étudiante en lettres, fan de théâtre.", "Étudiante", "Paris", 48.8462, 2.3441, []string{"theatre", "reading", "fashion"}, 100},
}

func main() {
	reset := flag.Bool("reset", false, "delete the demo accounts instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed: refusing to run with APP_ENV=production")
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

	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	if *reset {
		removeDemoAccounts(ctx, pool, store, cfg.UploadDir)
		return
	}
	signer := mediaurl.New([]byte(cfg.JWTSecret))
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.LogMailer{}, cfg.AppName)
	profileSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), signer.URL)
	photoSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), store)

	created := 0
	for _, d := range demoUsers {
		email := fmt.Sprintf("%s@%s", ascii(d.name), demoDomain)
		_, user, err := authSvc.Register(ctx, email, demoPassword, "seed", "")
		if err != nil {
			fmt.Printf("skip %s: %v\n", email, err)
			continue
		}
		err = profileSvc.SaveProfile(ctx, user.ID, profilesfeature.ProfileInput{
			FirstName: d.name, BirthDate: fmt.Sprintf("%d-06-15", d.birthYear), Gender: d.gender, Bio: d.bio,
			Occupation: d.job, LookingFor: "open", City: d.city, Latitude: d.lat, Longitude: d.lon, Interests: d.interests,
		})
		if err != nil {
			log.Fatalf("seed profile %s: %v", d.name, err)
		}
		if err := profileSvc.SavePreferences(ctx, user.ID, profilesfeature.PreferencesInput{
			InterestedIn: d.seeking, MinAge: 18, MaxAge: 60, MaxDistanceKm: 60,
		}); err != nil {
			log.Fatalf("seed preferences %s: %v", d.name, err)
		}
		for i := 0; i < 2; i++ {
			if _, err := photoSvc.Upload(ctx, user.ID, bytes.NewReader(placeholder(d.hue+i*25))); err != nil {
				log.Fatalf("seed photo %s: %v", d.name, err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo accounts\n  email:    <firstname>@%s (e.g. camille@%s)\n  password: %s\n", created, demoDomain, demoDomain, demoPassword)
	fmt.Println("placeholder portraits are generated gradients, not real people")
}

func ascii(name string) string {
	repl := map[rune]rune{'é': 'e', 'è': 'e', 'ï': 'i', 'î': 'i', 'ç': 'c', 'à': 'a'}
	out := []rune{}
	for _, r := range []rune(name) {
		if m, ok := repl[r]; ok {
			r = m
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}

// placeholder draws a 800x1000 soft gradient: obviously synthetic, deterministic.
func placeholder(hue int) []byte {
	const w, h = 800, 1000
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			img.Set(x, y, color.RGBA{
				R: uint8(60 + float64(hue%256)*0.3 + 120*t),
				G: uint8(90 + 100*(1-t) + float64(x)/w*40),
				B: uint8(150 + float64(hue%256)*0.2 - 60*t),
				A: 255,
			})
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, &jpeg.Options{Quality: 80})
	return b.Bytes()
}

func removeDemoAccounts(ctx context.Context, pool *pgxpool.Pool, store storage.Store, uploadDir string) {
	rows, err := pool.Query(ctx, `
		SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.email LIKE $1`, "%@"+demoDomain)
	if err != nil {
		log.Fatal(err)
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			log.Fatal(err)
		}
		keys = append(keys, k)
	}
	rows.Close()

	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE $1`, "%@"+demoDomain)
	if err != nil {
		log.Fatal(err)
	}
	for _, k := range keys {
		if err := store.Delete(ctx, k); err != nil {
			log.Printf("could not remove %s from %s: %v", k, uploadDir, err)
		}
	}
	fmt.Printf("removed %d demo accounts and %d photo files\n", tag.RowsAffected(), len(keys))
}
