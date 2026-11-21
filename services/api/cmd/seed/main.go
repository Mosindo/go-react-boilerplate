// Command seed creates demo members for local development and demos.
//
// Demo data is clearly separated from real data:
//   - it refuses to run when APP_ENV=production;
//   - every demo account uses the reserved ".invalid" email domain, so it can
//     never belong to a real person;
//   - every demo profile is flagged is_demo (shown with a "Démo" badge);
//   - `-purge` removes all demo accounts and their files.
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
	"math/rand"
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const demoDomain = "demo.lueur.invalid"

type persona struct {
	name   string
	gender string
	bio    string
	job    string
}

var personas = []persona{
	{"Camille", "woman", "Toujours partante pour un brunch ou une expo photo.", "Architecte"},
	{"Léa", "woman", "Randonneuse du dimanche, cuisinière du samedi.", "Infirmière"},
	{"Inès", "woman", "Je collectionne les vinyles et les plans de voyage.", "Designer"},
	{"Manon", "woman", "Escalade, café filtre et romans policiers.", "Chercheuse"},
	{"Sarah", "woman", "Fan de théâtre et de longues discussions.", "Avocate"},
	{"Jade", "woman", "Je cherche quelqu'un avec qui découvrir des petites tables.", "Cheffe de projet"},
	{"Chloé", "woman", "Yoga le matin, concerts le soir.", "Professeure"},
	{"Zoé", "woman", "Montagne l'hiver, océan l'été.", "Développeuse"},
	{"Lucas", "man", "Cycliste urbain et amateur de jazz.", "Ingénieur"},
	{"Hugo", "man", "Je fais le meilleur pain maison du quartier.", "Boulanger"},
	{"Nathan", "man", "Voyageur, photographe amateur, curieux de tout.", "Journaliste"},
	{"Louis", "man", "Jeux de société et cuisine du monde.", "Comptable"},
	{"Adam", "man", "Course à pied et podcasts de science.", "Médecin"},
	{"Gabriel", "man", "Je cherche une belle rencontre, simplement.", "Musicien"},
	{"Arthur", "man", "Ski, vin nature et bonne humeur.", "Entrepreneur"},
	{"Jules", "man", "Bénévole le week-end, cinéphile toute la semaine.", "Éducateur"},
	{"Alex", "nonbinary", "Art, mode et longues balades.", "Illustrateur·rice"},
	{"Sacha", "nonbinary", "Tech, jardinage et chats.", "Data analyst"},
}

var palettes = [][2]color.RGBA{
	{{0xC8, 0x55, 0x3D, 255}, {0xF2, 0xC1, 0x8D, 255}},
	{{0x5B, 0x3A, 0x5E, 255}, {0xE8, 0x9F, 0x89, 255}},
	{{0x2F, 0x5D, 0x62, 255}, {0xA7, 0xD7, 0xC5, 255}},
	{{0x8C, 0x4A, 0x2F, 255}, {0xF6, 0xE3, 0xDC, 255}},
	{{0x3D, 0x40, 0x5B, 255}, {0xF4, 0xA2, 0x61, 255}},
}

func main() {
	count := flag.Int("count", len(personas), "number of demo members to create")
	purge := flag.Bool("purge", false, "delete every demo member and exit")
	lat := flag.Float64("lat", 48.8566, "center latitude for demo members")
	lng := flag.Float64("lng", 2.3522, "center longitude for demo members")
	likeEmail := flag.String("like-email", "", "make every demo member like this account (instant matches when it likes back)")
	password := flag.String("password", "DemoPassword1", "password of demo accounts")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
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
	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		rows, err := pool.Query(ctx, `
			SELECT ph.storage_key, ph.thumb_storage_key FROM photos ph
			JOIN users u ON u.id = ph.user_id WHERE u.email LIKE '%@' || $1
		`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		for rows.Next() {
			var a, b string
			if err := rows.Scan(&a, &b); err == nil {
				_ = store.Delete(ctx, a)
				_ = store.Delete(ctx, b)
			}
		}
		rows.Close()
		tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo members\n", tag.RowsAffected())
		return
	}

	mediaSecret := cfg.MediaSigningSecret
	if mediaSecret == "" {
		mediaSecret = cfg.JWTSecret
	}
	signer := media.NewSigner([]byte(mediaSecret))
	authService := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret))
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(pool), store, signer, profilesService)

	var likeTarget string
	if *likeEmail != "" {
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(*likeEmail))).Scan(&likeTarget); err != nil {
			log.Fatalf("like-email: account not found (%v)", err)
		}
	}

	rng := rand.New(rand.NewSource(42))
	created := 0
	for i := 0; i < *count; i++ {
		p := personas[i%len(personas)]
		email := fmt.Sprintf("demo%02d@%s", i+1, demoDomain)
		_, user, err := authService.Register(ctx, email, *password, "seed", "")
		if err != nil {
			if err == authfeature.ErrEmailExists {
				continue
			}
			log.Fatalf("register %s: %v", email, err)
		}
		age := 22 + rng.Intn(20)
		birth := time.Now().AddDate(-age, -rng.Intn(11), -rng.Intn(27)).Format("2006-01-02")
		if err := profilesService.UpsertProfile(ctx, user.ID, profilesfeature.UpsertProfileRequest{
			FirstName: p.name, Birthdate: birth, Gender: p.gender, Bio: p.bio, City: "Paris", JobTitle: p.job,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		interested := []string{"woman", "man", "nonbinary"}
		if err := profilesService.UpdatePreferences(ctx, user.ID, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: interested, MinAge: 18, MaxAge: 60, MaxDistanceKm: 100,
		}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		ids := rng.Perm(40)[:4+rng.Intn(4)]
		for k := range ids {
			ids[k]++
		}
		if err := profilesService.UpdateInterests(ctx, user.ID, ids); err != nil {
			log.Fatalf("interests %s: %v", email, err)
		}
		// Random point within ~15 km of the center.
		r := 15.0 * math.Sqrt(rng.Float64())
		theta := rng.Float64() * 2 * math.Pi
		dLat := r / 111.0 * math.Cos(theta)
		dLng := r / (111.0 * math.Cos(*lat*math.Pi/180)) * math.Sin(theta)
		if err := profilesService.UpdateLocation(ctx, user.ID, *lat+dLat, *lng+dLng); err != nil {
			log.Fatalf("location %s: %v", email, err)
		}
		for photo := 0; photo < 2; photo++ {
			if _, err := photosService.Upload(ctx, user.ID, portrait(p.name, palettes[(i+photo)%len(palettes)], photo)); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		if _, err := pool.Exec(ctx, `UPDATE profiles SET is_demo = TRUE WHERE user_id = $1`, user.ID); err != nil {
			log.Fatal(err)
		}
		if likeTarget != "" {
			if _, err := pool.Exec(ctx, `
				INSERT INTO swipes (actor_user_id, target_user_id, action) VALUES ($1, $2, 'like')
				ON CONFLICT DO NOTHING
			`, user.ID, likeTarget); err != nil {
				log.Fatal(err)
			}
		}
		created++
	}
	fmt.Printf("created %d demo members (password %q, emails demoNN@%s)\n", created, *password, demoDomain)
}

// portrait renders an abstract placeholder picture with the member's initial.
func portrait(name string, palette [2]color.RGBA, variant int) []byte {
	const w, h = 720, 900
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, bottom := palette[0], palette[1]
	if variant == 1 {
		top, bottom = bottom, top
	}
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		row := color.RGBA{
			R: uint8(float64(top.R)*(1-t) + float64(bottom.R)*t),
			G: uint8(float64(top.G)*(1-t) + float64(bottom.G)*t),
			B: uint8(float64(top.B)*(1-t) + float64(bottom.B)*t),
			A: 255,
		}
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, row)
		}
	}
	// Soft circle.
	cx, cy, radius := w/2, h/2-40, 230.0
	for y := cy - 240; y < cy+240; y++ {
		for x := cx - 240; x < cx+240; x++ {
			d := math.Hypot(float64(x-cx), float64(y-cy))
			if d < radius {
				c := img.RGBAAt(x, y)
				alpha := 0.18 * (1 - d/radius)
				c.R = uint8(float64(c.R)*(1-alpha) + 255*alpha)
				c.G = uint8(float64(c.G)*(1-alpha) + 255*alpha)
				c.B = uint8(float64(c.B)*(1-alpha) + 255*alpha)
				img.SetRGBA(x, y, c)
			}
		}
	}
	// Initial, drawn small then upscaled.
	small := image.NewRGBA(image.Rect(0, 0, 9, 14))
	d := &font.Drawer{Dst: small, Src: image.NewUniform(color.RGBA{255, 255, 255, 230}), Face: basicfont.Face7x13, Dot: fixed.P(1, 11)}
	initial := strings.ToUpper(string([]rune(name)[0]))
	d.DrawString(initial)
	scale := 22
	ox, oy := cx-9*scale/2, cy-14*scale/2
	for y := 0; y < 14; y++ {
		for x := 0; x < 9; x++ {
			if small.RGBAAt(x, y).A > 100 {
				for yy := 0; yy < scale; yy++ {
					for xx := 0; xx < scale; xx++ {
						img.SetRGBA(ox+x*scale+xx, oy+y*scale+yy, color.RGBA{255, 255, 255, 255})
					}
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func init() {
	if os.Getenv("APP_ENV") == "" {
		// The seed is a development tool: default to development rather than
		// the API's production default.
		_ = os.Setenv("APP_ENV", config.EnvDevelopment)
	}
}
