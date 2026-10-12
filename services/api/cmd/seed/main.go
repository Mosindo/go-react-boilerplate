// Command seed fills a DEVELOPMENT database with clearly-fake demo accounts.
//
// Every demo account uses the reserved e-mail domain demo.invalid (RFC 2606),
// which can never receive mail nor belong to a real person, so demo rows are
// trivially separable from real data:
//
//	DELETE FROM users WHERE email LIKE '%@demo.invalid';
//
// The command refuses to run when APP_ENV=production.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mediaurl"
)

const (
	demoDomain = "demo.invalid"
	// Demo users are scattered around central Paris.
	centerLat, centerLon = 48.8566, 2.3522
)

type demo struct {
	name, gender, interestedIn string
	age                        int
	bio, job                   string
	interests                  []string
}

var demos = []demo{
	{"Camille", "woman", "man", 28, "Toujours partante pour une randonnée ou un bon café.", "Designer", []string{"hiking", "coffee", "photography"}},
	{"Hugo", "man", "woman", 31, "Cuisinier amateur, fan de vinyles.", "Ingénieur", []string{"cooking", "music", "travel"}},
	{"Léa", "woman", "man", 26, "Entre deux voyages. Je lis beaucoup.", "Journaliste", []string{"reading", "travel", "cinema"}},
	{"Nathan", "man", "woman", 29, "Escalade le week-end, jeux de société le soir.", "Développeur", []string{"sport", "boardgames", "tech"}},
	{"Inès", "woman", "man", 33, "Yoga, marché le dimanche et bon vin.", "Architecte", []string{"yoga", "wine", "art"}},
	{"Théo", "man", "woman", 27, "Photographe, toujours un appareil sur moi.", "Photographe", []string{"photography", "nature", "festivals"}},
	{"Manon", "woman", "man", 30, "Danse, théâtre et longues balades.", "Enseignante", []string{"dance", "theatre", "nature"}},
	{"Louis", "man", "woman", 35, "Je cherche quelqu'un avec qui rire.", "Médecin", []string{"fitness", "cinema", "coffee"}},
	{"Sarah", "woman", "woman", 29, "Gaming, café et chats.", "Product manager", []string{"gaming", "pets", "coffee"}},
	{"Jade", "woman", "woman", 27, "Musique live et festivals.", "Productrice", []string{"music", "festivals", "dance"}},
	{"Adam", "man", "man", 32, "Cuisine du monde et randonnées.", "Chef de projet", []string{"cooking", "hiking", "languages"}},
	{"Noé", "man", "man", 28, "Art contemporain et vélo.", "Graphiste", []string{"art", "fitness", "fashion"}},
	{"Alex", "non_binary", "woman", 30, "Bénévole, lecteur·ice compulsif·ve.", "Libraire", []string{"volunteering", "reading", "art"}},
	{"Charlie", "non_binary", "man", 26, "Mode, musique, bonne humeur.", "Styliste", []string{"fashion", "music", "dance"}},
	{"Emma", "woman", "man", 24, "Voyageuse dans l'âme.", "Étudiante", []string{"travel", "languages", "photography"}},
	{"Lucas", "man", "woman", 38, "Papa de deux chats, fan de cuisine.", "Chef cuisinier", []string{"cooking", "pets", "wine"}},
	{"Chloé", "woman", "man", 34, "Sport, nature, spontanéité.", "Kiné", []string{"sport", "nature", "hiking"}},
	{"Maxime", "man", "woman", 25, "Gamer et passionné de tech.", "Data analyst", []string{"gaming", "tech", "boardgames"}},
}

func main() {
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("seed refuses to run when APP_ENV=production")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	password := os.Getenv("DEMO_PASSWORD")
	if password == "" {
		password = "DemoPass123!"
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

	signer := mediaurl.New([]byte(cfg.JWTSecret))
	storage, err := photosfeature.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret))
	profSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), signer)
	photoSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), storage, signer)

	ids := make([]string, 0, len(demos))
	for i, d := range demos {
		email := fmt.Sprintf("demo.%02d@%s", i+1, demoDomain)
		_, user, err := authSvc.Register(ctx, email, password, "seed", "")
		if err != nil {
			var exists string
			if err2 := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&exists); err2 == nil {
				ids = append(ids, exists)
				continue
			}
			log.Fatalf("register %s: %v", email, err)
		}
		birth := time.Now().AddDate(-d.age, -(i % 11), -(i * 3 % 27)).Format("2006-01-02")
		if err := profSvc.UpdateProfile(ctx, user.ID, profilesfeature.UpdateProfileRequest{
			FirstName: d.name, BirthDate: birth, Gender: d.gender, Bio: d.bio, Occupation: d.job, City: "Paris", Interests: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if err := profSvc.UpdatePreferences(ctx, user.ID, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: []string{d.interestedIn}, AgeMin: 18, AgeMax: 60, MaxDistanceKm: 50,
		}); err != nil {
			log.Fatalf("prefs %s: %v", email, err)
		}
		angle := float64(i) * 2.4
		radius := 0.01 + 0.012*float64(i%7)
		if err := profSvc.UpdateLocation(ctx, user.ID, profilesfeature.UpdateLocationRequest{
			Latitude: centerLat + radius*math.Sin(angle), Longitude: centerLon + radius*math.Cos(angle),
		}); err != nil {
			log.Fatalf("location %s: %v", email, err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photoSvc.Upload(ctx, user.ID, placeholder(i*47+p*110)); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		ids = append(ids, user.ID)
		fmt.Printf("created %s (%s)\n", email, d.name)
	}

	// The first account is the one to log in with. A few others already liked it,
	// so liking them back produces an instant match.
	for _, id := range ids[1:7] {
		if _, err := pool.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action)
			SELECT $1, $2, 'like'
			WHERE EXISTS (SELECT 1 FROM profiles WHERE user_id = $1) AND $1 <> $2
			ON CONFLICT DO NOTHING`, id, ids[0]); err != nil {
			log.Fatalf("seed likes: %v", err)
		}
	}
	fmt.Printf("\nDemo accounts: demo.01@%[1]s ... demo.%02[2]d@%[1]s  password: %[3]s\n", demoDomain, len(demos), password)
}

// placeholder draws an abstract gradient: demo photos are never real people.
func placeholder(hue int) []byte {
	const w, h = 800, 1000
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y)/h*0.6 + float64(x)/w*0.4
			img.Set(x, y, hsv(float64(hue%360)+40*t, 0.55, 0.95-0.35*t))
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func hsv(h, s, v float64) color.RGBA {
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
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}
