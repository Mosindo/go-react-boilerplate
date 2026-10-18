// Command seed fills a development database with clearly fake demo accounts.
//
// Demo users all use the reserved domain demo.invalid, which can never receive
// mail or belong to a real person, so they are trivial to tell apart from (and
// to delete without touching) real data. The command refuses to run when
// APP_ENV=production.
//
//	SEED_CONFIRM=yes go run ./cmd/seed          # create / refresh demo users
//	SEED_CONFIRM=yes go run ./cmd/seed -reset   # delete demo users only
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.invalid"

type demoUser struct {
	name      string
	gender    string
	seeking   []string
	age       int
	bio       string
	interests []string
	dLat      float64 // offset from the base position, in degrees (~111 km per degree)
	dLon      float64
	likesMe   bool // pre-likes demo+01 so that liking back creates a match
}

var demoUsers = []demoUser{
	{"Camille", "woman", []string{"man"}, 29, "Grimpeuse le week-end, boulangère amatrice la semaine.", []string{"hiking", "cooking", "travel"}, 0, 0, false},
	{"Lucas", "man", []string{"woman"}, 32, "Photographe, toujours un appareil sur moi.", []string{"photography", "cinema", "coffee"}, 0.02, 0.03, true},
	{"Inès", "woman", []string{"man", "woman"}, 27, "Musique live, vin nature et longues balades.", []string{"concerts", "wine", "music"}, -0.03, 0.01, false},
	{"Hugo", "man", []string{"woman"}, 35, "Cuisine du marché et vélo.", []string{"cooking", "sports", "nature"}, 0.05, -0.04, true},
	{"Sarah", "woman", []string{"man"}, 31, "Lectrice compulsive, fan de ciné d'auteur.", []string{"reading", "cinema", "theatre"}, -0.06, 0.05, false},
	{"Nathan", "man", []string{"woman", "man"}, 28, "Développeur le jour, joueur de jeux de société le soir.", []string{"technology", "board-games", "gaming"}, 0.08, 0.02, false},
	{"Léa", "woman", []string{"man"}, 26, "Danse, yoga et gros appétit de voyages.", []string{"dancing", "yoga", "travel"}, -0.02, -0.07, false},
	{"Maxime", "man", []string{"woman"}, 38, "Randonnées, montagne et bon café.", []string{"hiking", "coffee", "nature"}, 0.12, 0.09, true},
	{"Yasmine", "woman", []string{"man"}, 30, "Art contemporain et brunchs interminables.", []string{"art", "coffee", "fashion"}, 0.04, -0.1, false},
	{"Jules", "man", []string{"woman"}, 41, "Course à pied, musique et chiens.", []string{"running", "music", "animals"}, -0.09, -0.03, false},
	{"Zoé", "nonbinary", []string{"man", "woman", "nonbinary"}, 28, "Langues, théâtre et bénévolat.", []string{"languages", "theatre", "volunteering"}, 0.01, 0.06, false},
	{"Arthur", "man", []string{"woman"}, 34, "Je cuisine, vous choisissez le vin.", []string{"cooking", "wine", "reading"}, -0.04, 0.08, false},
}

func main() {
	reset := flag.Bool("reset", false, "delete demo users instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	if os.Getenv("SEED_CONFIRM") != "yes" {
		log.Fatal("seed: set SEED_CONFIRM=yes to confirm you want demo data in this database")
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
	files, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *reset {
		rows, err := pool.Query(ctx, `SELECT p.storage_key, p.thumb_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.email LIKE $1`, "%@"+demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		var keys []string
		for rows.Next() {
			var a, b string
			if err := rows.Scan(&a, &b); err != nil {
				log.Fatal(err)
			}
			keys = append(keys, a, b)
		}
		rows.Close()
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE $1`, "%@"+demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		_ = files.Delete(keys...)
		log.Printf("seed: removed %d demo users", tag.RowsAffected())
		return
	}

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "demo-password-123"
	}
	baseLat := envFloat("SEED_LAT", 48.8566) // Paris by default
	baseLon := envFloat("SEED_LON", 2.3522)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	signer := photos.NewSigner([]byte(cfg.JWTSecret))
	photoSvc := photos.NewService(photos.NewPGRepository(pool), files, signer)
	profileSvc := profiles.NewService(profiles.NewPGRepository(pool), photoSvc)

	ids := make([]string, len(demoUsers))
	for i, d := range demoUsers {
		email := fmt.Sprintf("demo+%02d@%s", i+1, demoDomain)
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash) VALUES ($1, $2)
			ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash
			RETURNING id`, email, string(hash)).Scan(&id); err != nil {
			log.Fatal(err)
		}
		ids[i] = id

		birth := time.Now().AddDate(-d.age, -(i % 11), -(i * 3 % 27)).Format("2006-01-02")
		lat, lon := baseLat+d.dLat, baseLon+d.dLon
		_, err := profileSvc.Upsert(ctx, id, profiles.UpsertRequest{
			FirstName: d.name, BirthDate: birth, Gender: d.gender, Bio: d.bio, City: envOr("SEED_CITY", "Paris"),
			Latitude: &lat, Longitude: &lon, Interests: d.interests,
			Preferences: profiles.Preferences{InterestedIn: d.seeking, AgeMin: 21, AgeMax: 55},
		})
		if err != nil {
			log.Fatalf("seed: profile %s: %v", email, err)
		}

		var have int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, id).Scan(&have); err != nil {
			log.Fatal(err)
		}
		for have < 2 {
			if _, err := photoSvc.Add(ctx, id, demoPortrait(i, have)); err != nil {
				log.Fatalf("seed: photo %s: %v", email, err)
			}
			have++
		}
	}

	// A few demo people already liked demo+01, so liking them back shows a match.
	for i, d := range demoUsers {
		if d.likesMe && i != 0 {
			if _, err := pool.Exec(ctx, `
				INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING
			`, ids[i], ids[0]); err != nil {
				log.Fatal(err)
			}
		}
	}

	log.Printf("seed: %d demo users ready (demo+01@%s … demo+%02d@%s), password %q", len(demoUsers), demoDomain, len(demoUsers), demoDomain, password)
	log.Printf("seed: log in as demo+01@%s: Lucas, Hugo and Maxime already liked you; the others are in your deck", demoDomain)
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64); err == nil {
		return v
	}
	return def
}

// demoPortrait draws an abstract gradient: demo accounts never use real faces.
func demoPortrait(user, n int) []byte {
	const w, h = 720, 960
	hue := math.Mod(float64(user*47+n*140), 360)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			s := float64(x) / w
			r, g, b := hsv(hue+40*s, 0.55, 0.95-0.45*t)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsv(h, s, v float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 360)
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
