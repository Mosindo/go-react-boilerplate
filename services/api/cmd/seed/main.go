// Command seed fills a development database with clearly marked demo accounts
// (users.is_demo = true, email domain demo.invalid). It refuses to run in
// production. Run with -reset to remove the demo data again.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"strings"

	"example.com/api/internal/app"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5"
)

const demoPassword = "DemoPass123"

type demo struct {
	email, name, gender string
	age                 int
	interestedIn        []string
	bio, city           string
	dLat, dLng          float64
	interests           []string
}

// Paris-centred demo cast. Camille and Hugo are the pair to log in with to try matching and chat.
var cast = []demo{
	{"camille@demo.invalid", "Camille", "woman", 29, []string{"man"}, "Fan de randonnées et de cafés trop longs.", "Paris", 0, 0, []string{"hiking", "coffee", "photography"}},
	{"hugo@demo.invalid", "Hugo", "man", 31, []string{"woman"}, "Cuisine le dimanche, court le matin.", "Paris", 0.03, 0.02, []string{"cooking", "running", "music"}},
	{"lea@demo.invalid", "Léa", "woman", 26, []string{"man", "woman"}, "Cinéma d'auteur et pop des années 2000.", "Montreuil", 0.02, 0.06, []string{"cinema", "music", "dancing"}},
	{"yanis@demo.invalid", "Yanis", "man", 34, []string{"woman"}, "Photographe amateur, toujours un appareil sur moi.", "Paris", -0.02, 0.01, []string{"photography", "travel", "coffee"}},
	{"sofia@demo.invalid", "Sofia", "woman", 32, []string{"man"}, "Yoga le matin, vin le soir.", "Vincennes", 0.0, 0.09, []string{"yoga", "wine", "reading"}},
	{"theo@demo.invalid", "Théo", "man", 27, []string{"woman"}, "Développeur, joueur de société invétéré.", "Paris", 0.04, -0.03, []string{"tech", "board_games", "gaming"}},
	{"amel@demo.invalid", "Amel", "woman", 28, []string{"man"}, "Voyageuse, 31 pays et je continue.", "Paris", -0.03, -0.02, []string{"travel", "languages", "cooking"}},
	{"noa@demo.invalid", "Noa", "nonbinary", 30, []string{"man", "woman", "nonbinary"}, "Théâtre, vélo, et bonnes conversations.", "Paris", 0.01, 0.04, []string{"theatre", "cycling", "art"}},
	{"maxime@demo.invalid", "Maxime", "man", 36, []string{"woman"}, "Rando le week-end, animaux toute la semaine.", "Saint-Denis", 0.09, 0.03, []string{"hiking", "animals", "nature"}},
	{"ines@demo.invalid", "Inès", "woman", 25, []string{"man"}, "Festivals l'été, jeux de société l'hiver.", "Paris", -0.01, 0.05, []string{"festivals", "board_games", "music"}},
	{"paul@demo.invalid", "Paul", "man", 29, []string{"woman", "nonbinary"}, "Sport, lecture et un chien nommé Pixel.", "Boulogne-Billancourt", -0.04, -0.07, []string{"sport", "reading", "animals"}},
	{"clara@demo.invalid", "Clara", "woman", 33, []string{"man"}, "Bénévole le samedi, danseuse le dimanche.", "Paris", 0.02, -0.01, []string{"volunteering", "dancing", "wine"}},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo accounts and their photos")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	if cfg.IsProduction() {
		log.Fatal("seed refuses to run when APP_ENV=production")
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 30e9, 1e9)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *reset {
		rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			log.Fatal(err)
		}
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		for _, k := range keys {
			_ = store.Delete(ctx, k)
		}
		fmt.Printf("removed %d demo accounts\n", tag.RowsAffected())
		return
	}

	_, svc := app.New(app.Deps{DB: pool, JWTSecret: []byte(cfg.JWTSecret), Store: store, Mailer: mailer.LogMailer{}})
	ids := map[string]string{}
	for i, d := range cast {
		var existing string
		err := pool.QueryRow(ctx, `SELECT id::text FROM users WHERE email = $1`, d.email).Scan(&existing)
		if err == nil {
			ids[d.email] = existing
			continue
		}
		_, user, err := svc.Auth.Register(ctx, d.email, demoPassword, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", d.email, err)
		}
		ids[d.email] = user.ID
		if _, err := pool.Exec(ctx, `UPDATE users SET is_demo = TRUE WHERE id = $1`, user.ID); err != nil {
			log.Fatal(err)
		}
		birth := nowYear(d.age)
		if err := svc.Profiles.Save(ctx, user.ID, profiles.ProfileInput{
			FirstName: d.name, BirthDate: birth, Gender: d.gender, Bio: d.bio, City: d.city,
			Latitude: 48.8566 + d.dLat, Longitude: 2.3522 + d.dLng, Interests: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", d.email, err)
		}
		if _, err := svc.Profiles.SavePreferences(ctx, user.ID, profiles.Preferences{InterestedIn: d.interestedIn, MinAge: 21, MaxAge: 45}); err != nil {
			log.Fatalf("preferences %s: %v", d.email, err)
		}
		for n := 0; n < 2; n++ {
			if _, err := svc.Photos.Upload(ctx, user.ID, bytes.NewReader(gradient(i*47+n*120))); err != nil {
				log.Fatalf("photo %s: %v", d.email, err)
			}
		}
	}
	// Hugo already liked Camille: logging in as Camille and liking Hugo gives an instant match.
	if _, err := pool.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`,
		ids["hugo@demo.invalid"], ids["camille@demo.invalid"]); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("seeded %d demo accounts (password %q): %s ...\n", len(cast), demoPassword, strings.Join([]string{cast[0].email, cast[1].email}, ", "))
}

func nowYear(age int) string {
	return fmt.Sprintf("%d-01-15", currentYear()-age-1)
}

// gradient draws a soft two-tone placeholder portrait (no real people, no third-party images).
func gradient(hue int) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			r, g, b := hsv(float64((hue+int(t*60))%360), 0.55, 0.95-0.35*t)
			dx, dy := float64(x-w/2), float64(y-h/3)
			if math.Hypot(dx, dy) < 110 { // simple head silhouette
				r, g, b = 250, 240, 235
			}
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsv(h, s, v float64) (uint8, uint8, uint8) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
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
	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}
