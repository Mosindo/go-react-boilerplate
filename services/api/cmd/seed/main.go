// Command seed creates clearly-labelled demo profiles for local testing.
//
// Demo accounts are flagged users.is_demo = true, use the reserved
// ".invalid" e-mail domain, carry a "[Démo]" bio prefix and abstract
// generated pictures (never photos of real people). The command refuses to
// run when APP_ENV=production.
//
//	go run ./cmd/seed                         # create demo profiles
//	go run ./cmd/seed -like you@example.com   # demo profiles already like you → instant matches
//	go run ./cmd/seed -reset                  # delete every demo account
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"strings"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const demoDomain = "demo.lueur.invalid"

type demo struct {
	name, gender, birth, bio, job string
	interestedIn                  []string
	interests                     []string
	dLat, dLng                    float64
}

var demos = []demo{
	{"Camille", "woman", "1994-04-12", "Toujours partante pour une expo puis un verre en terrasse.", "Architecte", []string{"man", "woman"}, []string{"museums", "wine", "photography"}, 0.01, 0.02},
	{"Louise", "woman", "1997-09-03", "Randonneuse du dimanche, lectrice tous les soirs.", "Infirmière", []string{"man"}, []string{"hiking", "reading", "coffee"}, -0.02, 0.01},
	{"Inès", "woman", "1991-01-22", "Je cuisine mieux que je ne chante. Ce qui n'est pas difficile.", "Cheffe de projet", []string{"man"}, []string{"cooking", "travel", "concerts"}, 0.03, -0.01},
	{"Manon", "woman", "1999-06-15", "Escalade, jeux de société et longues discussions.", "Étudiante", []string{"woman", "nonbinary"}, []string{"climbing", "board_games", "series"}, -0.01, -0.03},
	{"Chloé", "woman", "1988-11-30", "Amoureuse de la mer et des petits matins calmes.", "Professeure", []string{"man"}, []string{"sea", "yoga", "brunch"}, 0.05, 0.04},
	{"Sarah", "woman", "1995-02-08", "Développeuse le jour, danseuse de salsa le soir.", "Développeuse", []string{"man", "nonbinary"}, []string{"dance", "tech", "music"}, -0.04, 0.02},
	{"Lucas", "man", "1993-07-19", "Vélo, café serré et cinéma d'auteur.", "Graphiste", []string{"woman"}, []string{"cycling", "coffee", "cinema"}, 0.02, -0.02},
	{"Hugo", "man", "1990-03-27", "Je cherche quelqu'un avec qui partager de bons restos.", "Ingénieur", []string{"woman"}, []string{"cooking", "wine", "travel"}, -0.03, -0.01},
	{"Thomas", "man", "1996-12-05", "Musicien amateur, voyageur professionnel (presque).", "Commercial", []string{"woman", "man"}, []string{"music", "concerts", "languages"}, 0.04, 0.01},
	{"Nathan", "man", "1998-05-21", "Footballeur du jeudi, gamer du vendredi.", "Kiné", []string{"woman"}, []string{"football", "video_games", "series"}, 0.01, 0.05},
	{"Julien", "man", "1987-08-14", "Jardinier urbain et papa d'un chat très bavard.", "Paysagiste", []string{"woman"}, []string{"gardening", "animals", "nature"}, -0.05, -0.04},
	{"Karim", "man", "1992-10-02", "Photographe de rue, toujours un appareil sur moi.", "Photographe", []string{"man"}, []string{"photography", "art", "museums"}, 0.02, 0.03},
	{"Alex", "nonbinary", "1996-04-28", "Théâtre, écriture et thé au jasmin.", "Comédien·ne", []string{"woman", "man", "nonbinary"}, []string{"theatre", "writing", "meditation"}, -0.02, 0.04},
	{"Sacha", "nonbinary", "1994-01-17", "Montagne l'hiver, mer l'été, livres toute l'année.", "Bibliothécaire", []string{"woman", "nonbinary"}, []string{"mountain", "reading", "sea"}, 0.03, -0.04},
	{"Élodie", "woman", "1985-06-09", "Bénévole, curieuse et un peu trop compétitive aux quiz.", "Juriste", []string{"man"}, []string{"volunteering", "board_games", "theatre"}, -0.06, 0.00},
	{"Maxime", "man", "1989-09-25", "Course à pied le matin, jazz le soir.", "Médecin", []string{"woman"}, []string{"running", "music", "science"}, 0.00, -0.06},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo accounts and exit")
	likeEmail := flag.String("like", "", "e-mail of a real account that every demo profile will like")
	password := flag.String("password", "", "password for demo accounts (random if empty)")
	lat := flag.Float64("lat", 48.8566, "centre latitude for demo profiles")
	lng := flag.Float64("lng", 2.3522, "centre longitude for demo profiles")
	flag.Parse()
	log.SetFlags(0)

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 10*time.Second, time.Second)
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
	signer := media.NewSigner([]byte(cfg.JWTSecret))
	photoSvc := photos.NewService(photos.NewPGRepository(pool), store, signer)

	if *reset {
		n, err := resetDemo(ctx, pool, photoSvc)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("deleted %d demo accounts\n", n)
		return
	}

	pw := *password
	if pw == "" {
		b := make([]byte, 6)
		_, _ = rand.Read(b)
		pw = "Demo" + hex.EncodeToString(b) + "9"
	}

	authSvc := auth.NewService(auth.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.LogMailer{}, auth.AccountHooks{})
	profileSvc := profiles.NewService(profiles.NewPGRepository(pool), signer)

	created := 0
	ids := make([]string, 0, len(demos))
	for i, d := range demos {
		email := fmt.Sprintf("%s@%s", strings.ToLower(asciiName(d.name)), demoDomain)
		var existing string
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&existing); err == nil {
			ids = append(ids, existing)
			continue
		}
		_, user, err := authSvc.Register(ctx, email, pw, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET is_demo = TRUE WHERE id = $1`, user.ID); err != nil {
			log.Fatal(err)
		}
		if _, err := profileSvc.UpdateProfile(ctx, user.ID, profiles.UpdateProfileRequest{
			FirstName: d.name, Birthdate: d.birth, Gender: d.gender, Bio: "[Démo] " + d.bio, JobTitle: d.job,
			LookingFor: []string{"relationship", "not_sure", "casual", "friendship"}[i%4], Interests: d.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", d.name, err)
		}
		if _, err := profileSvc.UpdatePreferences(ctx, user.ID, profiles.UpdatePreferencesRequest{
			InterestedIn: d.interestedIn, MinAge: 18, MaxAge: 99, MaxDistanceKm: 200,
		}); err != nil {
			log.Fatal(err)
		}
		la, ln := *lat+d.dLat, *lng+d.dLng
		if err := profileSvc.UpdateLocation(ctx, user.ID, profiles.UpdateLocationRequest{Latitude: &la, Longitude: &ln, City: "Démo"}); err != nil {
			log.Fatal(err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photoSvc.Upload(ctx, user.ID, artwork(i*7+p*3)); err != nil {
				log.Fatalf("photo %s: %v", d.name, err)
			}
		}
		ids = append(ids, user.ID)
		created++
	}
	fmt.Printf("demo profiles: %d created, %d already present (e-mail domain @%s)\n", created, len(ids)-created, demoDomain)
	if *password != "" || created > 0 {
		fmt.Printf("demo password: %s\n", pw)
	}

	if *likeEmail != "" {
		var target string
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND NOT is_demo`, strings.ToLower(strings.TrimSpace(*likeEmail))).Scan(&target); err != nil {
			log.Fatalf("account %s not found: register it in the app first", *likeEmail)
		}
		tag, err := pool.Exec(ctx, `
			INSERT INTO swipes (actor_id, target_id, action)
			SELECT unnest($1::UUID[]), $2, 'like'
			ON CONFLICT DO NOTHING
		`, ids, target)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d demo profiles now like %s: liking them back creates a match\n", tag.RowsAffected(), *likeEmail)
	}
}

func resetDemo(ctx context.Context, pool *pgxpool.Pool, photoSvc *photos.Service) (int, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM users WHERE is_demo`)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		cleanup, err := photoSvc.PrepareAccountDeletion(ctx, id)
		if err != nil {
			return 0, err
		}
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1 AND is_demo`, id); err != nil {
			return 0, err
		}
		cleanup()
	}
	return len(ids), nil
}

func asciiName(name string) string {
	r := strings.NewReplacer("é", "e", "É", "E", "è", "e", "ï", "i", "î", "i", "ô", "o", "ç", "c")
	return r.Replace(name)
}

// artwork renders an abstract gradient picture so demo profiles can never
// be mistaken for real people.
func artwork(seed int) []byte {
	const w, h = 720, 900
	palette := [][2]color.RGBA{
		{{0xE0, 0x6D, 0x4F, 255}, {0x4A, 0x1E, 0x3A, 255}},
		{{0xF2, 0xB8, 0x8C, 255}, {0x7A, 0x2E, 0x4D, 255}},
		{{0x9C, 0xB8, 0xA0, 255}, {0x2E, 0x3F, 0x4F, 255}},
		{{0xF4, 0xD3, 0x8B, 255}, {0xB0, 0x4A, 0x3A, 255}},
		{{0xB8, 0xA9, 0xD9, 255}, {0x3A, 0x2E, 0x5C, 255}},
	}
	pair := palette[seed%len(palette)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)*(0.3+0.4*math.Mod(float64(seed)*0.37, 1)), float64(h)*(0.35+0.3*math.Mod(float64(seed)*0.61, 1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / 260
			glow := math.Max(0, 1-d) * 0.55
			mix := func(a, b uint8) uint8 {
				v := float64(a)*(1-t) + float64(b)*t
				v = v*(1-glow) + 255*glow
				return uint8(math.Min(255, v))
			}
			img.Set(x, y, color.RGBA{mix(pair[0].R, pair[1].R), mix(pair[0].G, pair[1].G), mix(pair[0].B, pair[1].B), 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
