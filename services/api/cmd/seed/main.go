// Command seed creates clearly-flagged demonstration members for local
// testing. It refuses to run when APP_ENV=production.
//
// Demo accounts:
//   - use the reserved, undeliverable domain demo.lueur.invalid;
//   - have users.is_demo = TRUE and are badged "Profil de démo" in the app;
//   - can be removed at any time with: go run ./cmd/seed -purge
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
	mrand "math/rand/v2"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	demoDomain   = "demo.lueur.invalid"
	demoPassword = "Demo12345"
)

var (
	womenNames  = []string{"Camille", "Léa", "Manon", "Chloé", "Inès", "Sarah", "Jade", "Louise", "Emma", "Alice", "Zoé", "Nina"}
	menNames    = []string{"Lucas", "Hugo", "Louis", "Nathan", "Gabriel", "Arthur", "Jules", "Adam", "Raphaël", "Paul", "Tom", "Noé"}
	nbNames     = []string{"Sacha", "Charlie", "Alex", "Eden"}
	jobs        = []string{"Architecte", "Infirmière", "Développeur·se", "Professeur·e", "Photographe", "Chef·fe de projet", "Graphiste", "Libraire", "Kiné", "Ingénieur·e"}
	bios        = []string{"Amateur·rice de cafés de quartier et de longues balades.", "Toujours partant·e pour un concert improvisé.", "Je cuisine mieux que je ne chante, promis.", "Week-ends en montagne, soirées jeux de société.", "À la recherche de quelqu'un avec qui refaire le monde en terrasse.", "Lecteur·rice compulsif·ve, voyageur·se occasionnel·le."}
	interestSet = []string{"hiking", "coffee", "music", "cinema", "travel", "cooking", "reading", "yoga", "running", "photography", "board_games", "wine", "concerts", "museums", "pets"}
)

func main() {
	count := flag.Int("count", 24, "number of demo members to create")
	lat := flag.Float64("lat", 48.8566, "centre latitude")
	lng := flag.Float64("lng", 2.3522, "centre longitude")
	radius := flag.Float64("radius-km", 15, "spread radius in km")
	likeEmail := flag.String("like-email", "", "make about half of the demo members like this account (instant matches when it likes back)")
	purge := flag.Bool("purge", false, "delete every demo member and exit")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	storage, err := media.NewLocalStorage(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		n, err := purgeDemo(ctx, pool, storage)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("removed %d demo members\n", n)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	var likeTarget string
	if *likeEmail != "" {
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(*likeEmail))).Scan(&likeTarget); err != nil {
			log.Fatalf("like-email account not found: %v", err)
		}
	}

	rng := mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), 42))
	created := 0
	for i := 0; i < *count; i++ {
		gender, name, interestedIn := pickIdentity(rng, i)
		email := fmt.Sprintf("demo-%s@%s", randomHex(4), demoDomain)
		age := 22 + rng.IntN(20)
		birthdate := time.Now().AddDate(-age, -rng.IntN(12), -rng.IntN(28))
		dLat, dLng := jitter(rng, *lat, *radius)

		err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			var userID string
			if err := tx.QueryRow(ctx, `
				INSERT INTO users (email, password_hash, organization_id, is_demo, last_active_at)
				VALUES ($1, $2, (SELECT id FROM organizations WHERE slug = 'default'), TRUE, NOW() - make_interval(hours => $3))
				RETURNING id
			`, email, string(hash), rng.IntN(96)).Scan(&userID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, latitude, longitude, city, location_updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'Paris', NOW())
			`, userID, name, birthdate, gender, bios[rng.IntN(len(bios))], jobs[rng.IntN(len(jobs))],
				round2(*lat+dLat), round2(*lng+dLng)); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
				VALUES ($1, $2, 18, 60, 80)
			`, userID, interestedIn); err != nil {
				return err
			}
			picks := pickInterests(rng)
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
			`, userID, picks); err != nil {
				return err
			}
			for pos := 0; pos < 1+rng.IntN(3); pos++ {
				img, err := media.ProcessUpload(gradientPNG(rng))
				if err != nil {
					return err
				}
				key := randomHex(16) + ".jpg"
				if err := storage.Save(key, img.Data); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO photos (user_id, storage_key, mime_type, size_bytes, width, height, position)
					VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6)
				`, userID, key, len(img.Data), img.Width, img.Height, pos); err != nil {
					return err
				}
			}
			if likeTarget != "" && i%2 == 0 {
				if _, err := tx.Exec(ctx, `
					INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING
				`, userID, likeTarget); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Fatalf("seed member %d: %v", i, err)
		}
		created++
	}
	fmt.Printf("created %d demo members around (%.4f, %.4f). Demo password: %s\n", created, *lat, *lng, demoPassword)
	fmt.Println("Demo accounts are flagged is_demo and use @" + demoDomain + ". Remove them with -purge.")
}

func pickIdentity(rng *mrand.Rand, i int) (gender, name string, interestedIn []string) {
	switch {
	case i%10 == 9:
		return "nonbinary", nbNames[rng.IntN(len(nbNames))], []string{"man", "nonbinary", "woman"}
	case i%2 == 0:
		if i%6 == 4 {
			return "woman", womenNames[rng.IntN(len(womenNames))], []string{"man", "woman"}
		}
		return "woman", womenNames[rng.IntN(len(womenNames))], []string{"man"}
	default:
		return "man", menNames[rng.IntN(len(menNames))], []string{"woman"}
	}
}

func pickInterests(rng *mrand.Rand) []string {
	perm := rng.Perm(len(interestSet))
	n := 3 + rng.IntN(4)
	out := make([]string, 0, n)
	for _, idx := range perm[:n] {
		out = append(out, interestSet[idx])
	}
	return out
}

// jitter returns a random offset (in degrees) within radiusKm.
func jitter(rng *mrand.Rand, lat, radiusKm float64) (float64, float64) {
	r := radiusKm * math.Sqrt(rng.Float64())
	theta := rng.Float64() * 2 * math.Pi
	dLat := (r * math.Cos(theta)) / 111.0
	dLng := (r * math.Sin(theta)) / (111.0 * math.Cos(lat*math.Pi/180))
	return dLat, dLng
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// gradientPNG draws an abstract warm gradient: no real person is depicted.
func gradientPNG(rng *mrand.Rand) []byte {
	const w, h = 600, 800
	palette := [][2]color.RGBA{
		{{232, 168, 124, 255}, {94, 52, 90, 255}},
		{{127, 209, 174, 255}, {33, 60, 82, 255}},
		{{240, 180, 138, 255}, {150, 70, 80, 255}},
		{{180, 160, 230, 255}, {40, 35, 70, 255}},
	}
	p := palette[rng.IntN(len(palette))]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			s := t*0.8 + float64(x)/w*0.2
			img.Pix[y*img.Stride+x*4+0] = uint8(float64(p[0].R)*(1-s) + float64(p[1].R)*s)
			img.Pix[y*img.Stride+x*4+1] = uint8(float64(p[0].G)*(1-s) + float64(p[1].G)*s)
			img.Pix[y*img.Stride+x*4+2] = uint8(float64(p[0].B)*(1-s) + float64(p[1].B)*s)
			img.Pix[y*img.Stride+x*4+3] = 255
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, storage media.Storage) (int, error) {
	rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.is_demo`)
	if err != nil {
		return 0, err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, err
	}
	if _, err := pool.Exec(ctx, `
		DELETE FROM conversations c USING conversation_participants cp, users u
		WHERE cp.conversation_id = c.id AND cp.user_id = u.id AND u.is_demo
	`); err != nil {
		return 0, err
	}
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		return 0, err
	}
	for _, k := range keys {
		_ = storage.Delete(k)
	}
	return int(tag.RowsAffected()), nil
}
