// Command seed fills a development database with clearly labelled demo members.
//
// Demo users are flagged users.is_demo = TRUE, use the reserved e-mail domain
// "seed.lumo.invalid" (never deliverable) and go through the same services as
// real sign-ups, so every validation rule applies. It refuses to run when
// APP_ENV=production. Run `go run ./cmd/seed -reset` to remove them again.
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
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
)

const demoDomain = "seed.lumo.invalid"

type demo struct {
	name      string
	gender    profilesfeature.Gender
	age       int
	interests []string
	bio       string
	into      []profilesfeature.Gender
}

var (
	women = []profilesfeature.Gender{profilesfeature.GenderWoman}
	men   = []profilesfeature.Gender{profilesfeature.GenderMan}
	all   = []profilesfeature.Gender{profilesfeature.GenderMan, profilesfeature.GenderWoman, profilesfeature.GenderNonBinary}
)

var demos = []demo{
	{"Camille", "woman", 28, []string{"travel", "photography", "coffee"}, "Toujours un appareil photo dans le sac. Je cherche quelqu'un pour explorer la ville.", men},
	{"Inès", "woman", 31, []string{"cooking", "wine", "reading"}, "Cheffe amateur, lectrice compulsive. Je cuisine, tu choisis le vin ?", men},
	{"Sofia", "woman", 26, []string{"dancing", "music", "concerts"}, "Salsa le jeudi, concerts le week-end.", men},
	{"Léa", "woman", 34, []string{"hiking", "nature", "yoga"}, "Grand air et bonne humeur. Randonnée dès que la météo le permet.", men},
	{"Nora", "woman", 29, []string{"movies", "series", "board-games"}, "Cinéphile et mauvaise perdante aux jeux de société.", men},
	{"Eva", "woman", 37, []string{"art", "museums", "theatre"}, "Un musée, un café, une discussion qui dure.", all},
	{"Jade", "woman", 24, []string{"running", "gym", "podcasts"}, "Marathon en préparation, podcasts en boucle.", all},
	{"Mila", "woman", 41, []string{"gardening", "animals", "cooking"}, "Un jardin, deux chats, beaucoup de projets.", men},
	{"Lucas", "man", 30, []string{"cycling", "coffee", "technology"}, "Développeur le jour, cycliste le week-end.", women},
	{"Hugo", "man", 27, []string{"football", "gaming", "music"}, "Foot, jeux vidéo et playlists partagées.", women},
	{"Samir", "man", 33, []string{"cooking", "travel", "languages"}, "J'apprends une langue par voyage. Prochaine étape : le japonais.", women},
	{"Adrien", "man", 36, []string{"climbing", "hiking", "nature"}, "Escalade en salle ou en falaise, au choix.", women},
	{"Théo", "man", 25, []string{"surfing", "photography", "travel"}, "À la recherche de la prochaine vague.", women},
	{"Nathan", "man", 39, []string{"reading", "writing", "wine"}, "Écrivain du dimanche, amateur de bons vins.", women},
	{"Malik", "man", 29, []string{"dancing", "concerts", "fashion"}, "Danse, mode et bonne humeur.", all},
	{"Paul", "man", 44, []string{"diy", "gardening", "board-games"}, "Je répare tout, sauf mon sens de l'orientation.", women},
	{"Alex", "nonbinary", 27, []string{"art", "technology", "gaming"}, "Illustrateur·rice et joueur·se. Curieux·se de tout.", all},
	{"Sacha", "nonbinary", 32, []string{"meditation", "yoga", "volunteering"}, "Calme, curieux·se, engagé·e.", all},
}

func main() {
	reset := flag.Bool("reset", false, "delete every demo account (and its photos) instead of creating them")
	password := flag.String("password", envOr("SEED_PASSWORD", "DemoPass123!"), "password shared by all demo accounts")
	lat := flag.Float64("lat", 48.8566, "latitude of the city centre demo members live around")
	lon := flag.Float64("lon", 2.3522, "longitude of the city centre")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production (set APP_ENV=development)")
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
	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	photos := photosfeature.NewService(photosfeature.NewPGRepository(pool), store)

	if *reset {
		rows, err := pool.Query(ctx, `SELECT id::text FROM users WHERE is_demo`)
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
			if err := photos.PurgeUser(ctx, id); err != nil {
				log.Printf("purge photos of %s: %v", id, err)
			}
		}
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo accounts\n", tag.RowsAffected())
		return
	}

	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mail.DiscardMailer{})
	profilesSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))

	created := 0
	for i, d := range demos {
		email := fmt.Sprintf("demo%02d@%s", i+1, demoDomain)
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
			log.Fatal(err)
		}
		if exists {
			continue
		}
		_, user, err := authSvc.Register(ctx, email, *password, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET is_demo = TRUE WHERE id = $1`, user.ID); err != nil {
			log.Fatal(err)
		}

		// Spread members over ~12 km around the centre, deterministically.
		angle := float64(i) * 2.399963 // golden angle
		radius := 0.01 * math.Sqrt(float64(i+1)) * 4
		pLat, pLon := *lat+radius*math.Sin(angle), *lon+radius*math.Cos(angle)

		birth := time.Now().UTC().AddDate(-d.age, -(i%11 + 1), 0).Format("2006-01-02")
		if _, err := profilesSvc.UpdateProfile(ctx, user.ID, profilesfeature.UpdateProfileRequest{
			FirstName: d.name, BirthDate: birth, Gender: d.gender, Bio: d.bio, City: "Paris",
			Latitude: &pLat, Longitude: &pLon, Interests: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", d.name, err)
		}
		if _, err := profilesSvc.UpdatePreferences(ctx, user.ID, profilesfeature.Preferences{
			InterestedIn: d.into, MinAge: 20, MaxAge: 55, MaxDistanceKm: 50,
		}); err != nil {
			log.Fatalf("preferences %s: %v", d.name, err)
		}
		for n := 0; n < 2+i%2; n++ {
			if _, err := photos.Upload(ctx, user.ID, bytes.NewReader(portrait(i, n))); err != nil {
				log.Fatalf("photo %s: %v", d.name, err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo accounts (%d already existed)\n", created, len(demos)-created)
	fmt.Printf("log in with demoNN@%s (NN = 01..%02d), password: %s\n", demoDomain, len(demos), *password)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// portrait renders an abstract gradient placeholder; no real person's picture is ever bundled.
func portrait(seed, variant int) []byte {
	const w, h = 600, 800
	hue := math.Mod(float64(seed)*47+float64(variant)*25, 360)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		c := hsl(hue+40*t, 0.55, 0.62-0.25*t)
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	// A soft circle stands in for a head-and-shoulders silhouette.
	cx, cy, r := float64(w)/2, float64(h)*0.38, 120.0
	for y := int(cy - r); y < int(cy+r); y++ {
		for x := int(cx - r); x < int(cx+r); x++ {
			if math.Hypot(float64(x)-cx, float64(y)-cy) < r {
				img.Set(x, y, hsl(hue+20, 0.45, 0.85))
			}
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360) / 360
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	f := func(t float64) float64 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return color.RGBA{uint8(f(h+1.0/3) * 255), uint8(f(h) * 255), uint8(f(h-1.0/3) * 255), 255}
}
