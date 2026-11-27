// Command seed creates clearly-labelled demo accounts for local testing.
//
// Safety rails:
//   - refuses to run when APP_ENV=production;
//   - every account uses the reserved ".invalid" TLD and is flagged
//     users.is_demo, which the API never shows in production discovery and
//     which the app labels as "Profil de démonstration";
//   - pictures are generated abstract artwork, never photos of real people.
//
// Usage: go run ./cmd/seed [-count 24] [-lat 48.8566 -lng 2.3522] [-like you@example.com] [-reset]
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
	"math/rand"
	"os"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const demoDomain = "demo.lueur.invalid"

type persona struct {
	name   string
	gender string
	wants  []string
	bio    string
}

var personas = []persona{
	{"Camille", "woman", []string{"man", "woman"}, "Toujours partante pour un café en terrasse et une expo improvisée."},
	{"Léa", "woman", []string{"man"}, "Randonneuse du dimanche, cuisinière du lundi au samedi."},
	{"Inès", "woman", []string{"man"}, "Je collectionne les vinyles et les fous rires."},
	{"Manon", "woman", []string{"woman"}, "Escalade, podcasts et petits restaurants de quartier."},
	{"Chloé", "woman", []string{"man"}, "Plutôt lever de soleil que grasse matinée."},
	{"Sarah", "woman", []string{"man", "nonbinary"}, "Danse, langues étrangères et voyages en train."},
	{"Jade", "woman", []string{"man"}, "Fan de jeux de société, imbattable au Dixit."},
	{"Louise", "woman", []string{"woman", "nonbinary"}, "Jardinage urbain et cinéma d'auteur."},
	{"Nora", "woman", []string{"man"}, "Yoga le matin, concerts le soir."},
	{"Lucas", "man", []string{"woman"}, "Cycliste urbain, amateur de bonnes tables."},
	{"Hugo", "man", []string{"woman"}, "Je fais le meilleur tiramisu de l'arrondissement."},
	{"Nathan", "man", []string{"man"}, "Course à pied, séries et brunchs."},
	{"Théo", "man", []string{"woman"}, "Photographe amateur, toujours un appareil sur moi."},
	{"Malik", "man", []string{"woman", "man"}, "Musique live, tennis et longues discussions."},
	{"Jules", "man", []string{"woman"}, "Bénévole le week-end, curieux de tout."},
	{"Antoine", "man", []string{"woman"}, "Grimpeur, lecteur, voyageur."},
	{"Yanis", "man", []string{"woman"}, "Football, cuisine du monde et balades à vélo."},
	{"Arthur", "man", []string{"man", "nonbinary"}, "Théâtre, musées et vin nature."},
	{"Sacha", "nonbinary", []string{"woman", "man", "nonbinary"}, "Illustration, méditation et marchés du dimanche."},
	{"Eden", "nonbinary", []string{"woman", "nonbinary"}, "Jeux vidéo indé et randonnées en forêt."},
	{"Alex", "nonbinary", []string{"man", "nonbinary"}, "Mode vintage, cafés cachés et vieux films."},
	{"Maël", "man", []string{"woman"}, "Natation, jazz et cuisine végétarienne."},
	{"Zoé", "woman", []string{"man"}, "Animaux, nature et pique-niques."},
	{"Rose", "woman", []string{"man", "woman"}, "Écriture, théâtre et café noir."},
}

var palettes = [][2]color.RGBA{
	{{0xC8, 0x55, 0x3D, 0xFF}, {0xF2, 0xA6, 0x5A, 0xFF}},
	{{0x6B, 0x8F, 0x71, 0xFF}, {0xE9, 0xD8, 0xA6, 0xFF}},
	{{0x3D, 0x40, 0x5B, 0xFF}, {0xE0, 0x7A, 0x5F, 0xFF}},
	{{0x81, 0xB2, 0x9A, 0xFF}, {0xF4, 0xF1, 0xDE, 0xFF}},
	{{0x9C, 0x6B, 0x98, 0xFF}, {0xF6, 0xBD, 0x60, 0xFF}},
}

func main() {
	count := flag.Int("count", len(personas), "number of demo profiles (max 24)")
	lat := flag.Float64("lat", 48.8566, "centre latitude for demo profiles")
	lng := flag.Float64("lng", 2.3522, "centre longitude for demo profiles")
	likeEmail := flag.String("like", "", "e-mail of an existing account every demo profile will like (instant matches)")
	reset := flag.Bool("reset", false, "delete existing demo accounts first")
	flag.Parse()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
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
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoLueur2026"
	}
	if *count < 1 || *count > len(personas) {
		log.Fatalf("count must be between 1 and %d", len(personas))
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, databaseURL, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store, err := storage.NewLocalStore(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *reset {
		if err := deleteDemo(ctx, pool, store); err != nil {
			log.Fatal(err)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	var interestIDs []int
	rows, err := pool.Query(ctx, `SELECT id FROM interests ORDER BY id`)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		interestIDs = append(interestIDs, id)
	}
	rows.Close()

	rng := rand.New(rand.NewSource(42))
	created := 0
	for i := 0; i < *count; i++ {
		p := personas[i]
		email := fmt.Sprintf("demo%02d@%s", i+1, demoDomain)
		var userID string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, organization_id, is_demo)
			VALUES ($1, $2, (SELECT id FROM organizations WHERE slug = 'default'), TRUE)
			ON CONFLICT (email) DO NOTHING
			RETURNING id
		`, email, string(hash)).Scan(&userID)
		if err != nil {
			continue // already seeded
		}

		age := 21 + rng.Intn(20)
		birthdate := time.Now().AddDate(-age, -rng.Intn(11), -rng.Intn(27))
		// Scatter within ~12 km, then snap like the API does.
		dLat := (rng.Float64() - 0.5) * 0.2
		dLng := (rng.Float64() - 0.5) * 0.3
		pLat := profiles.SnapCoordinate(*lat + dLat)
		pLng := profiles.SnapCoordinate(*lng + dLng)

		if _, err := pool.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, city, latitude, longitude, location_updated_at, last_active_at)
			VALUES ($1, $2, $3, $4, $5, 'Démo', $6, $7, NOW(), NOW() - make_interval(hours => $8))
		`, userID, p.name, birthdate, p.gender, "[Profil de démonstration] "+p.bio, pLat, pLng, rng.Intn(72)); err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
			VALUES ($1, $2, 18, 60, 100)
		`, userID, p.wants); err != nil {
			log.Fatal(err)
		}
		picked := map[int]bool{}
		for len(picked) < 4 && len(interestIDs) > 0 {
			picked[interestIDs[rng.Intn(len(interestIDs))]] = true
		}
		for id := range picked {
			if _, err := pool.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, id); err != nil {
				log.Fatal(err)
			}
		}
		for pos := 0; pos < 2; pos++ {
			data := artwork(p.name, palettes[(i+pos)%len(palettes)], pos)
			key, err := storage.NewPhotoKey(userID)
			if err != nil {
				log.Fatal(err)
			}
			if err := store.Put(ctx, key, data); err != nil {
				log.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes)
				VALUES ($1, $2, $3, 900, 1200, $4)
			`, userID, pos, key, len(data)); err != nil {
				log.Fatal(err)
			}
		}
		created++
	}

	if *likeEmail != "" {
		tag, err := pool.Exec(ctx, `
			INSERT INTO swipes (swiper_user_id, target_user_id, action)
			SELECT u.id, target.id, 'like'
			FROM users u, users target
			WHERE u.is_demo AND target.email = lower(trim($1)) AND target.id <> u.id
			ON CONFLICT DO NOTHING
		`, *likeEmail)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d demo profiles now like %s: like them back to get a match.\n", tag.RowsAffected(), *likeEmail)
	}

	fmt.Printf("Seeded %d demo profiles (password: %s, e-mails demoNN@%s).\n", created, password, demoDomain)
}

func deleteDemo(ctx context.Context, pool *pgxpool.Pool, store storage.Store) error {
	rows, err := pool.Query(ctx, `SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.is_demo`)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		return err
	}
	for _, k := range keys {
		_ = store.Delete(ctx, k)
	}
	fmt.Printf("Removed %d demo accounts.\n", tag.RowsAffected())
	return nil
}

// artwork renders a warm gradient with soft circles and the initial.
func artwork(name string, pal [2]color.RGBA, variant int) []byte {
	const w, h = 900, 1200
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		for x := 0; x < w; x++ {
			u := t*0.8 + float64(x)/w*0.2
			img.Set(x, y, color.RGBA{
				R: uint8(float64(pal[0].R)*(1-u) + float64(pal[1].R)*u),
				G: uint8(float64(pal[0].G)*(1-u) + float64(pal[1].G)*u),
				B: uint8(float64(pal[0].B)*(1-u) + float64(pal[1].B)*u),
				A: 255,
			})
		}
	}
	cx, cy := float64(w)*(0.3+0.4*float64(variant)), float64(h)*0.35
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			if d < 260 {
				c := img.RGBAAt(x, y)
				a := 0.18 * (1 - d/260)
				img.Set(x, y, color.RGBA{
					R: uint8(float64(c.R)*(1-a) + 255*a),
					G: uint8(float64(c.G)*(1-a) + 255*a),
					B: uint8(float64(c.B)*(1-a) + 255*a),
					A: 255,
				})
			}
		}
	}

	// Draw the initial with the built-in bitmap font, scaled up.
	glyph := image.NewRGBA(image.Rect(0, 0, 14, 16))
	d := &font.Drawer{Dst: glyph, Src: image.NewUniform(color.White), Face: basicfont.Face7x13, Dot: fixed.P(3, 13)}
	initial := []rune(name)[0]
	d.DrawString(string(initial))
	scale := 24
	ox, oy := (w-14*scale)/2, h-16*scale-120
	for y := 0; y < 16; y++ {
		for x := 0; x < 14; x++ {
			if glyph.RGBAAt(x, y).A > 0 {
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						img.Set(ox+x*scale+sx, oy+y*scale+sy, color.RGBA{255, 255, 255, 230})
					}
				}
			}
		}
	}

	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82})
	return buf.Bytes()
}
