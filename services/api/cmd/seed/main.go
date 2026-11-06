// Command seed fills a development database with clearly fake demo accounts.
//
// Safety rules: it refuses to run when APP_ENV=production, every account uses
// the reserved, undeliverable domain @demo.invalid (RFC 2606) so it can never be
// confused with a real user, and `-reset` removes exactly those accounts.
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

const (
	demoDomain   = "@demo.invalid"
	demoPassword = "demo-password-123"
)

type person struct {
	slug, name, gender string
	age                int
	lat, lng           float64
	interests          []string
	bio                string
}

var people = []person{
	{"demo", "Camille", "woman", 29, 48.8566, 2.3522, []string{"travel", "music", "cinema"}, "Compte de démonstration. Curieuse, fan de cinéma et de longues balades."},
	{"lea", "Léa", "woman", 27, 48.8700, 2.3300, []string{"cooking", "travel", "wine"}, "Cheffe amateure. J'échange une recette contre un bon livre."},
	{"manon", "Manon", "woman", 31, 48.8400, 2.3800, []string{"hiking", "nature", "photography"}, "Week-ends en forêt, appareil photo en poche."},
	{"sarah", "Sarah", "woman", 26, 48.8900, 2.3400, []string{"music", "festivals", "dance"}, "Toujours un festival en vue."},
	{"julie", "Julie", "woman", 33, 48.8300, 2.2900, []string{"yoga", "reading", "coffee"}, "Yoga le matin, roman le soir."},
	{"emma", "Emma", "woman", 24, 48.9000, 2.4000, []string{"gaming", "tech", "boardgames"}, "Je gagne à tous les jeux de société, sauf aux échecs."},
	{"clara", "Clara", "woman", 35, 48.7800, 2.4600, []string{"art", "cinema", "coffee"}, "Musées, cafés et conversations interminables."},
	{"ines", "Inès", "woman", 28, 48.8600, 2.3000, []string{"sport", "travel", "animals"}, "Course à pied et chien nommé Biscotte."},
	{"alice", "Alice", "nonbinary", 30, 48.8450, 2.3600, []string{"art", "music", "fashion"}, "Friperies, vinyles et bonne humeur."},
	{"lucas", "Lucas", "man", 30, 48.8600, 2.3500, []string{"travel", "music", "sport"}, "Sac à dos toujours prêt."},
	{"hugo", "Hugo", "man", 28, 48.8800, 2.3200, []string{"cooking", "wine", "cinema"}, "Je cuisine pour les gens que j'aime. Candidats bienvenus."},
	{"theo", "Théo", "man", 32, 48.8350, 2.3900, []string{"hiking", "nature", "photography"}, "Randonneur du dimanche, photographe du lundi."},
	{"nathan", "Nathan", "man", 26, 48.9100, 2.3500, []string{"gaming", "tech", "coffee"}, "Développeur, accro au café et aux jeux indé."},
	{"adam", "Adam", "man", 34, 48.8200, 2.3100, []string{"sport", "yoga", "reading"}, "Coach sportif qui lit trop."},
	{"louis", "Louis", "man", 29, 48.8650, 2.4100, []string{"music", "festivals", "dance"}, "Batteur dans un groupe local."},
	{"jules", "Jules", "man", 37, 48.7900, 2.3700, []string{"art", "boardgames", "wine"}, "Collectionneur de jeux de société et de bonnes bouteilles."},
	{"max", "Max", "man", 25, 48.8500, 2.2800, []string{"animals", "nature", "sport"}, "Vétérinaire, toujours une histoire d'animal à raconter."},
	{"noah", "Noah", "nonbinary", 27, 48.8750, 2.3650, []string{"tech", "art", "music"}, "Créatif·ve numérique, curieux·se de tout."},
}

func main() {
	reset := flag.Bool("reset", false, "delete the demo accounts instead of creating them")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
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

	if *reset {
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo accounts (photo files, if any, stay in %s)\n", tag.RowsAffected(), cfg.UploadDir)
		return
	}

	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), nil, store)
	profRepo := profilesfeature.NewPGRepository(pool)
	profSvc := profilesfeature.NewService(profRepo)
	photoSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), store)

	ids := map[string]string{}
	created := 0
	for i, p := range people {
		email := p.slug + demoDomain
		var existing string
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&existing); err == nil {
			ids[p.slug] = existing
			continue
		}
		_, user, err := authSvc.Register(ctx, email, demoPassword, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		ids[p.slug] = user.ID
		birth := time.Now().AddDate(-p.age, 0, -10).Format("2006-01-02")
		in, err := profSvc.ValidateProfile(profilesfeature.ProfileInput{
			FirstName: p.name, Gender: p.gender, Bio: p.bio, City: "Paris",
			Latitude: p.lat, Longitude: p.lng, Interests: p.interests,
		}, birth)
		if err != nil {
			log.Fatalf("profile %s: %v", p.slug, err)
		}
		if err := profSvc.SaveProfile(ctx, user.ID, in); err != nil {
			log.Fatalf("save profile %s: %v", p.slug, err)
		}
		prefs, _ := profSvc.ValidatePreferences(profilesfeature.Preferences{
			InterestedIn: interestedIn(p), MinAge: 21, MaxAge: 45, MaxDistanceKm: 50,
		})
		if err := profSvc.SavePreferences(ctx, user.ID, prefs); err != nil {
			log.Fatalf("prefs %s: %v", p.slug, err)
		}
		if _, err := photoSvc.Upload(ctx, user.ID, gradient(i)); err != nil {
			log.Fatalf("photo %s: %v", p.slug, err)
		}
		created++
	}

	// A few people already like the main demo account so a match is one swipe away.
	for _, slug := range []string{"lucas", "hugo", "theo"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO swipes (actor_id, target_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING
		`, ids[slug], ids["demo"]); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("seeded %d new demo accounts (%d total)\n", created, len(people))
	fmt.Printf("sign in with demo%s / %s\n", demoDomain, demoPassword)
}

func interestedIn(p person) []string {
	switch p.gender {
	case "woman":
		return []string{"man", "nonbinary"}
	case "man":
		return []string{"woman", "nonbinary"}
	default:
		return []string{"woman", "man", "nonbinary"}
	}
}

// gradient renders a deterministic placeholder portrait (no real people involved).
func gradient(seed int) []byte {
	const w, h = 640, 800
	hue := float64(seed*47%360) / 360
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			s := float64(x) / w
			r, g, b := hsv(hue+0.12*s, 0.55, 0.95-0.45*t)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	// A soft "head and shoulders" silhouette keeps the card readable.
	cx, cy := w/2, h*2/5
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x-cx), float64(y-cy)
			head := dx*dx+dy*dy < 110*110
			body := y > cy+120 && float64((x-cx)*(x-cx))/(230*230)+float64((y-(h+60))*(y-(h+60)))/(500*500) < 1
			if head || body {
				r, g, b, _ := img.At(x, y).RGBA()
				img.Set(x, y, color.RGBA{uint8(r >> 9), uint8(g >> 9), uint8(b >> 9), 255})
			}
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}

func hsv(h, s, v float64) (uint8, uint8, uint8) {
	h -= float64(int(h))
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
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
