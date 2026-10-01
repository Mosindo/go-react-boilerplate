// Command seed fills a DEVELOPMENT database with clearly fake demo members.
//
// Demo accounts use the reserved ".invalid" TLD (demo.<name>@demo.invalid), so
// they can never be confused with — or emailed as — real users. The command
// refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed [-city paris|lyon|montreal] [-likes-me you@example.com]
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
)

type demoMember struct {
	Name      string
	Gender    string
	Age       int
	Bio       string
	Interests []string
	Likes     []string // genders
}

var members = []demoMember{
	{"Camille", "woman", 29, "Grimpeuse du dimanche, accro au café filtre. Cherche quelqu'un pour explorer les marchés.", []string{"climbing", "coffee", "travel"}, []string{"man"}},
	{"Léa", "woman", 26, "Illustratrice. J'aime les musées vides et les concerts bruyants.", []string{"art", "museums", "concerts"}, []string{"man", "woman"}},
	{"Inès", "woman", 31, "Cheffe à mi-temps, randonneuse le reste du temps.", []string{"cooking", "hiking", "wine"}, []string{"man"}},
	{"Manon", "woman", 34, "Prof de yoga, lectrice compulsive, maîtresse de deux chats.", []string{"yoga", "reading", "animals"}, []string{"man", "woman"}},
	{"Sarah", "woman", 28, "Développeuse, joueuse de société et cycliste urbaine.", []string{"tech", "board_games", "cycling"}, []string{"man"}},
	{"Jade", "woman", 24, "Photographe amateure, toujours un appareil dans le sac.", []string{"photography", "travel", "music"}, []string{"man", "woman"}},
	{"Hugo", "man", 32, "Ingénieur le jour, cuisinier le soir. Je fais un excellent risotto.", []string{"cooking", "wine", "running"}, []string{"woman"}},
	{"Lucas", "man", 28, "Passionné de cinéma et de vieux vinyles.", []string{"cinema", "music", "reading"}, []string{"woman"}},
	{"Théo", "man", 35, "Randonnée, vélo, bivouac. Je cherche une co-équipière.", []string{"hiking", "cycling", "photography"}, []string{"woman"}},
	{"Nathan", "man", 27, "Danseur et bénévole dans une asso de quartier.", []string{"dancing", "volunteering", "coffee"}, []string{"woman", "man"}},
	{"Adam", "man", 30, "Fan de jeux vidéo et de jardinage (oui, les deux).", []string{"gaming", "gardening", "languages"}, []string{"woman"}},
	{"Louis", "man", 38, "Théâtre amateur, grand marcheur, petit bricoleur.", []string{"theatre", "hiking", "art"}, []string{"woman"}},
	{"Alex", "non_binary", 29, "Musicien·ne, curieux·se de tout. Toujours partant·e pour un concert.", []string{"music", "concerts", "languages"}, []string{"woman", "man", "non_binary"}},
	{"Sam", "non_binary", 33, "Lecteur·rice, cinéphile, amateur·rice de thé.", []string{"reading", "cinema", "coffee"}, []string{"woman", "man", "non_binary", "other"}},
}

var cities = map[string]struct {
	Name     string
	Lat, Lng float64
}{
	"paris":    {"Paris", 48.8566, 2.3522},
	"lyon":     {"Lyon", 45.7640, 4.8357},
	"montreal": {"Montréal", 45.5019, -73.5674},
}

func main() {
	cityKey := flag.String("city", "paris", "city to scatter demo members around (paris, lyon, montreal)")
	likesMe := flag.String("likes-me", "", "email of a real dev account that the first demo members should already like")
	flag.Parse()

	city, ok := cities[*cityKey]
	if !ok {
		log.Fatalf("unknown city %q", *cityKey)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
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
	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	password := os.Getenv("DEMO_PASSWORD")
	if password == "" {
		password = "DemoPassword123"
	}
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret))
	profilesSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), store, profilesSvc)
	rng := rand.New(rand.NewSource(42))

	created := 0
	ids := make([]string, 0, len(members))
	for i, m := range members {
		email := "demo." + strings.ToLower(strings.NewReplacer("é", "e", "è", "e", "ï", "i").Replace(m.Name)) + "@demo.invalid"
		_, user, err := authSvc.Register(ctx, email, password, "seed", "127.0.0.1")
		if errors.Is(err, authfeature.ErrEmailExists) {
			continue
		}
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		birth := time.Now().AddDate(-m.Age, -rng.Intn(10), -rng.Intn(25)).Format("2006-01-02")
		if _, err := profilesSvc.Upsert(ctx, user.ID, profilesfeature.UpsertProfileRequest{
			FirstName: m.Name, BirthDate: birth, Gender: m.Gender, Bio: m.Bio, City: city.Name, Interests: m.Interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", m.Name, err)
		}
		if _, err := profilesSvc.UpdatePreferences(ctx, user.ID, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: m.Likes, AgeMin: 18, AgeMax: 60, MaxDistanceKm: 60,
		}); err != nil {
			log.Fatalf("preferences %s: %v", m.Name, err)
		}
		// scatter within ~8 km of the city centre
		angle, dist := rng.Float64()*2*math.Pi, rng.Float64()*8
		lat, lng := city.Lat+dist/111*math.Sin(angle), city.Lng+dist/(111*math.Cos(city.Lat*math.Pi/180))*math.Cos(angle)
		if err := profilesSvc.SetLocation(ctx, user.ID, profilesfeature.UpdateLocationRequest{Latitude: &lat, Longitude: &lng, City: city.Name}); err != nil {
			log.Fatalf("location %s: %v", m.Name, err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photosSvc.Add(ctx, user.ID, portrait(i*2+p)); err != nil {
				log.Fatalf("photo %s: %v", m.Name, err)
			}
		}
		ids = append(ids, user.ID)
		created++
	}

	if *likesMe != "" {
		var target string
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(email) = lower($1)`, *likesMe).Scan(&target); err != nil {
			log.Fatalf("no account with email %q — register in the app first: %v", *likesMe, err)
		}
		liked := 0
		for _, id := range ids {
			if liked == 5 {
				break
			}
			if _, err := pool.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, id, target); err == nil {
				liked++
			}
		}
		fmt.Printf("%d demo members now like %s — like them back to get matches\n", liked, *likesMe)
	}

	fmt.Printf("Seeded %d new demo members around %s (%d already existed).\n", created, city.Name, len(members)-created)
	fmt.Printf("Log in as demo.camille@demo.invalid / %s (all demo accounts share that password).\n", password)
}

// portrait renders a soft two-tone gradient used as a placeholder photo.
func portrait(seed int) []byte {
	const w, h = 720, 960
	hue := float64(seed*47%360) / 360
	top := hsv(hue, 0.45, 0.95)
	bottom := hsv(math.Mod(hue+0.12, 1), 0.65, 0.55)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		c := color.RGBA{
			R: uint8(float64(top.R)*(1-t) + float64(bottom.R)*t),
			G: uint8(float64(top.G)*(1-t) + float64(bottom.G)*t),
			B: uint8(float64(top.B)*(1-t) + float64(bottom.B)*t),
			A: 255,
		}
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
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
