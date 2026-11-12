// Command seed loads demonstration members so the app can be tried locally.
//
// Demo data is isolated by construction: every demo account uses the reserved
// ".invalid" TLD (RFC 2606, can never be a real address), bios start with
// "[Démo]", and the tool refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed          # create / refresh demo members
//	go run ./cmd/seed -reset   # delete all demo members
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
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	demoDomain   = "demo.lueur.invalid"
	demoPassword = "Demo12345"
)

type demoMember struct {
	Key          string
	Name         string
	Birthdate    string
	Gender       string
	InterestedIn []string
	Goal         string
	Occupation   string
	City         string
	Lat, Lng     float64
	Bio          string
	Interests    []string
	LikesDemo    bool
	Hue          float64
}

var members = []demoMember{
	{"demo", "Camille", "1994-04-12", "woman", []string{"man", "woman", "nonbinary"}, "long_term", "Architecte", "Lyon", 45.76, 4.84, "Compte de démonstration : connectez-vous avec lui pour explorer l'app.", []string{"coffee", "hiking", "photography", "travel"}, false, 18},
	{"lucas", "Lucas", "1992-09-03", "man", []string{"woman"}, "long_term", "Ingénieur", "Lyon", 45.75, 4.85, "Grimpeur du dimanche, cuisinier du lundi.", []string{"climbing", "cooking", "board_games"}, true, 210},
	{"ines", "Inès", "1997-01-22", "woman", []string{"woman"}, "unsure", "Graphiste", "Villeurbanne", 45.77, 4.88, "Expos, vinyles et longues balades.", []string{"art", "music", "museums"}, true, 320},
	{"hugo", "Hugo", "1990-07-14", "man", []string{"woman"}, "short_term", "Chef", "Lyon", 45.74, 4.82, "Je vous invite au marché, vous choisissez le dessert.", []string{"cooking", "wine", "brunch"}, false, 30},
	{"sam", "Sam", "1995-11-30", "nonbinary", []string{"woman", "man", "nonbinary"}, "friendship", "Libraire", "Lyon", 45.76, 4.83, "Toujours un livre dans le sac.", []string{"reading", "podcasts", "cats"}, true, 150},
	{"yanis", "Yanis", "1993-03-08", "man", []string{"woman"}, "long_term", "Kiné", "Caluire", 45.79, 4.85, "Course le matin, ciné le soir.", []string{"running", "cinema", "series"}, true, 260},
	{"lea", "Léa", "1998-05-19", "woman", []string{"man"}, "long_term", "Infirmière", "Lyon", 45.75, 4.86, "Rires faciles et road trips improvisés.", []string{"road_trips", "festivals", "dogs"}, false, 340},
	{"noah", "Noah", "1996-12-02", "man", []string{"woman", "man"}, "unsure", "Développeur", "Lyon", 45.76, 4.81, "Jeux de société et café de spécialité.", []string{"board_games", "coffee", "tech"}, true, 190},
	{"chloe", "Chloé", "1991-08-27", "woman", []string{"man", "woman"}, "long_term", "Avocate", "Lyon", 45.77, 4.83, "Yoga, théâtre et dîners qui s'éternisent.", []string{"yoga", "theatre", "wine"}, true, 0},
	{"theo", "Théo", "1999-02-11", "man", []string{"woman"}, "short_term", "Étudiant", "Lyon", 45.73, 4.84, "Surf l'été, ski l'hiver.", []string{"surf", "ski", "travel"}, false, 230},
	{"maya", "Maya", "1993-10-05", "woman", []string{"man"}, "long_term", "Médecin", "Paris", 48.86, 2.35, "Parisienne, amoureuse de Lyon.", []string{"museums", "running", "languages"}, true, 300},
	{"adam", "Adam", "1989-06-21", "man", []string{"woman", "nonbinary"}, "long_term", "Photographe", "Lyon", 45.76, 4.85, "Je photographie les gens qui rient.", []string{"photography", "nature", "camping"}, true, 120},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo members and exit")
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
	storage, err := media.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if err := deleteDemo(ctx, pool, storage); err != nil {
		log.Fatal(err)
	}
	if *reset {
		fmt.Println("demo members deleted")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	ids := map[string]string{}
	for _, m := range members {
		id, err := createMember(ctx, pool, storage, m, string(hash))
		if err != nil {
			log.Fatalf("seed %s: %v", m.Key, err)
		}
		ids[m.Key] = id
	}
	for _, m := range members {
		if m.LikesDemo {
			if _, err := pool.Exec(ctx, `INSERT INTO swipes (actor_user_id, target_user_id, decision) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, ids[m.Key], ids["demo"]); err != nil {
				log.Fatal(err)
			}
		}
	}

	fmt.Printf("seeded %d demo members (password for all: %s)\n", len(members), demoPassword)
	fmt.Printf("main demo account: demo@%s — several members already liked it, so liking them back creates matches.\n", demoDomain)
}

func deleteDemo(ctx context.Context, pool *pgxpool.Pool, storage media.Storage) error {
	rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.email LIKE '%@' || $1`, demoDomain)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if _, err := pool.Exec(ctx, `
		DELETE FROM conversations c USING conversation_participants cp, users u
		WHERE cp.conversation_id = c.id AND u.id = cp.user_id AND u.email LIKE '%@' || $1
	`, demoDomain); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain); err != nil {
		return err
	}
	for _, k := range keys {
		_ = storage.Delete(k)
	}
	return nil
}

func createMember(ctx context.Context, pool *pgxpool.Pool, storage media.Storage, m demoMember, hash string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	email := m.Key + "@" + demoDomain
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, organization_id)
		VALUES ($1, $2, (SELECT id FROM organizations WHERE slug = 'default'))
		RETURNING id
	`, email, hash).Scan(&id); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, occupation, relationship_goal, city, latitude, longitude, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`, id, m.Name, m.Birthdate, m.Gender, "[Démo] "+m.Bio, m.Occupation, m.Goal, m.City, m.Lat, m.Lng); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 18, 60, 100)
	`, id, m.InterestedIn); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, id, m.Interests); err != nil {
		return "", err
	}
	for pos := 0; pos < 2; pos++ {
		img, err := media.Process(portrait(m.Hue+float64(pos)*25, pos))
		if err != nil {
			return "", err
		}
		key, err := media.NewKey()
		if err != nil {
			return "", err
		}
		if err := storage.Save(key, img.Data); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, size_bytes, width, height, position) VALUES ($1, $2, $3, $4, $5, $6)
		`, id, key, len(img.Data), img.Width, img.Height, pos); err != nil {
			return "", err
		}
	}
	return id, tx.Commit(ctx)
}

// portrait draws an abstract placeholder (warm gradient + soft silhouette):
// obviously illustrative, never mistakable for a real person's photo.
func portrait(hue float64, variant int) []byte {
	const w, h = 480, 600
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			c := hsl(math.Mod(hue+t*40, 360), 0.32, 0.58-0.22*t)
			dx, dy := float64(x-w/2), float64(y-int(float64(h)*0.38))
			head := dx*dx+dy*dy < 85*85
			bx, by := float64(x-w/2), float64(y-h)
			body := bx*bx/(190*190)+by*by/(230*230) < 1
			if variant == 1 {
				head = head && x > w/2-200
			}
			if head || body {
				c = hsl(math.Mod(hue+180, 360), 0.25, 0.88)
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
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
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}

func init() {
	if os.Getenv("UPLOAD_DIR") == "" {
		_ = os.Setenv("UPLOAD_DIR", "./data/uploads")
	}
}
