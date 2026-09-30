// Command seed creates demo members for local development and testing.
//
// Demo accounts are clearly separated from real ones: they use the reserved
// ".invalid" email domain (RFC 2606, cannot receive mail), are flagged with
// users.is_demo = TRUE, and the command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed                         # create demo members
//	go run ./cmd/seed -like-email me@x.com    # demo members also like this account
//	go run ./cmd/seed -purge                  # delete every demo member
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
	"strings"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	demoDomain   = "demo.invalid"
	demoPassword = "DemoPassword1"
)

type demoMember struct {
	name, gender, birthdate, job, city, bio, goal string
	interestedIn                                  []string
	lat, lon                                      float64
	interests                                     []string
	hue                                           float64
}

var members = []demoMember{
	{"Camille", "woman", "1995-04-12", "Architecte", "Paris 11e", "Amoureuse des marchés du dimanche et des expos photo.", "long_term", []string{"man", "woman"}, 48.859, 2.379, []string{"photography", "museums", "brunch"}, 12},
	{"Léa", "woman", "1998-09-03", "Infirmière", "Paris 18e", "Toujours partante pour une rando ou un concert.", "unsure", []string{"man"}, 48.892, 2.344, []string{"hiking", "concerts", "travel"}, 340},
	{"Inès", "woman", "1992-01-22", "Cheffe de projet", "Boulogne-Billancourt", "Je cuisine mieux que je ne danse. Enfin, presque.", "long_term", []string{"man"}, 48.835, 2.241, []string{"cooking", "dance", "wine"}, 25},
	{"Chloé", "woman", "2000-06-30", "Étudiante en design", "Paris 5e", "Carnet de croquis toujours dans le sac.", "friendship", []string{"woman", "nonbinary"}, 48.846, 2.345, []string{"design", "art", "coffee"}, 280},
	{"Sarah", "woman", "1989-11-15", "Avocate", "Paris 16e", "Tennis le samedi, séries le dimanche.", "long_term", []string{"man"}, 48.864, 2.276, []string{"tennis", "series", "reading"}, 200},
	{"Manon", "woman", "1996-03-08", "Développeuse", "Montreuil", "Jeux de société et escalade, dans cet ordre.", "short_term", []string{"man", "woman"}, 48.861, 2.443, []string{"board_games", "climbing", "tech"}, 160},
	{"Julie", "woman", "1993-07-19", "Professeure des écoles", "Vincennes", "Je collectionne les plantes et les bonnes adresses.", "long_term", []string{"man"}, 48.847, 2.439, []string{"gardening", "brunch", "reading"}, 100},
	{"Nora", "woman", "1991-12-01", "Photographe", "Paris 10e", "Toujours à la recherche de la meilleure lumière.", "unsure", []string{"man", "woman"}, 48.876, 2.360, []string{"photography", "travel", "cinema"}, 50},
	{"Thomas", "man", "1994-02-14", "Ingénieur", "Paris 12e", "Vélo le matin, cinéma le soir.", "long_term", []string{"woman"}, 48.840, 2.395, []string{"cycling", "cinema", "coffee"}, 220},
	{"Lucas", "man", "1997-05-27", "Barista", "Paris 3e", "Je peux vous parler de café pendant des heures.", "short_term", []string{"woman", "man"}, 48.863, 2.360, []string{"coffee", "music", "concerts"}, 30},
	{"Hugo", "man", "1990-10-10", "Médecin", "Neuilly-sur-Seine", "Coureur du dimanche et lecteur du soir.", "long_term", []string{"woman"}, 48.884, 2.268, []string{"running", "reading", "travel"}, 190},
	{"Karim", "man", "1993-08-21", "Chef cuisinier", "Paris 20e", "Si vous aimez manger, on va bien s'entendre.", "unsure", []string{"woman"}, 48.864, 2.398, []string{"cooking", "wine", "football"}, 5},
	{"Antoine", "man", "1988-04-04", "Musicien", "Paris 9e", "Piano, jazz et longues balades.", "long_term", []string{"woman", "man"}, 48.877, 2.337, []string{"music", "concerts", "nature"}, 260},
	{"Maxime", "man", "1999-01-17", "Étudiant", "Créteil", "Gamer assumé, randonneur occasionnel.", "friendship", []string{"woman"}, 48.790, 2.455, []string{"gaming", "hiking", "tech"}, 140},
	{"Yanis", "man", "1995-09-09", "Kiné", "Issy-les-Moulineaux", "Yoga et natation pour rester zen.", "long_term", []string{"woman"}, 48.824, 2.273, []string{"yoga", "swimming", "meditation"}, 180},
	{"Alex", "nonbinary", "1996-11-11", "Illustratrice·eur", "Paris 19e", "Je dessine, je voyage, je recommence.", "unsure", []string{"woman", "man", "nonbinary"}, 48.884, 2.382, []string{"art", "travel", "writing"}, 300},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo members and exit")
	likeEmail := flag.String("like-email", "", "make demo members like this existing account (to test matches)")
	flag.Parse()

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

	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	photoService := photos.NewService(photos.NewPGRepository(pool), store, media.NewSigner([]byte(cfg.JWTSecret)))

	if *purge {
		if err := purgeDemo(ctx, pool, photoService); err != nil {
			log.Fatal(err)
		}
		return
	}

	authService := auth.NewService(auth.NewPGRepository(pool), authtoken.NewManager([]byte(cfg.JWTSecret)), nil)
	created := 0
	for i, m := range members {
		email := fmt.Sprintf("%s.%d@%s", strings.ToLower(asciiName(m.name)), i+1, demoDomain)
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
			log.Fatal(err)
		}
		if exists {
			continue
		}
		_, user, err := authService.Register(ctx, email, demoPassword, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		if err := fillProfile(ctx, pool, user.ID, m); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photoService.Upload(ctx, user.ID, portrait(m.hue+float64(p)*35, p)); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
		created++
	}
	log.Printf("demo members created: %d (password for all: %s, domain: @%s)", created, demoPassword, demoDomain)

	if *likeEmail != "" {
		tag, err := pool.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action)
			SELECT d.id, t.id, 'like'
			FROM users d, users t
			WHERE d.is_demo AND t.email = $1 AND d.id <> t.id
			ON CONFLICT DO NOTHING
		`, auth.NormalizeEmail(*likeEmail))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("demo likes sent to %s: %d (like them back to get matches)", *likeEmail, tag.RowsAffected())
	}
}

func fillProfile(ctx context.Context, pool *pgxpool.Pool, userID string, m demoMember) error {
	if _, err := pool.Exec(ctx, `UPDATE users SET is_demo = TRUE WHERE id = $1`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE profiles SET first_name = $2, birthdate = $3, gender = $4, bio = $5, job_title = $6,
		       relationship_goal = $7, city = $8, latitude = $9, longitude = $10, location_updated_at = NOW()
		WHERE user_id = $1
	`, userID, m.name, m.birthdate, m.gender, m.bio, m.job, m.goal, m.city, math.Round(m.lat*100)/100, math.Round(m.lon*100)/100); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE preferences SET interested_in = $2, min_age = 18, max_age = 60, max_distance_km = 100 WHERE user_id = $1
	`, userID, m.interestedIn); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
		ON CONFLICT DO NOTHING
	`, userID, m.interests)
	return err
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, photoService *photos.Service) error {
	rows, err := pool.Query(ctx, `SELECT id FROM users WHERE is_demo`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		finalize, err := photoService.PrepareAccountDeletion(ctx, id)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1 AND is_demo`, id); err != nil {
			return err
		}
		finalize(ctx)
	}
	log.Printf("demo members deleted: %d", len(ids))
	return nil
}

// portrait renders an abstract warm gradient placeholder (no real faces).
func portrait(hue float64, variant int) []byte {
	const w, h = 720, 900
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			dx, dy := float64(x)-w/2, float64(y)-h*0.42
			circle := math.Exp(-(dx*dx+dy*dy)/(2*170*170)) * 0.35
			if variant == 1 {
				circle = math.Exp(-(dx*dx)/(2*260*260)) * 0.2
			}
			r, g, b := hsl(math.Mod(hue+t*40, 360), 0.55, 0.35+0.3*t+circle)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) (uint8, uint8, uint8) {
	l = math.Min(l, 0.95)
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
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

func asciiName(name string) string {
	return strings.NewReplacer("é", "e", "è", "e", "É", "E", "ï", "i", "î", "i").Replace(name)
}
