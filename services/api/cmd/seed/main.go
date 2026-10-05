// Command seed fills a development database with clearly labelled demo profiles, or removes them.
//
//	go run ./cmd/seed          # create demo users (idempotent)
//	go run ./cmd/seed --purge  # delete every demo user and their photos
//
// Demo users have users.is_demo = TRUE and an @demo.invalid address (a reserved TLD that can never receive
// mail), so they cannot be confused with real accounts. The command refuses to run when APP_ENV=production.
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
	"image/jpeg"
	"log"
	"math"
	"os"
	"time"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type person struct {
	first, gender, interestedIn string
	age                         int
	job, bio                    string
	interests                   []string
	dLat, dLng                  float64 // offset from the centre, in degrees (~111 km per degree)
}

var people = []person{
	{"Camille", "woman", "man", 29, "Architecte", "Amatrice de randonnées et de bons cafés. Je cherche quelqu'un de curieux.", []string{"hiking", "coffee", "art", "travel"}, 0.01, 0.02},
	{"Julien", "man", "woman", 32, "Ingénieur", "Cuisine du dimanche, vélo et séries. Pas de prise de tête.", []string{"cooking", "cycling", "series"}, -0.02, 0.01},
	{"Inès", "woman", "man", 26, "Médecin", "Entre deux gardes : escalade, cinéma et longues discussions.", []string{"climbing", "cinema", "reading"}, 0.03, -0.01},
	{"Thomas", "man", "woman", 35, "Photographe", "Je capture des moments. J'aimerais en vivre avec toi.", []string{"photography", "travel", "music"}, -0.01, -0.03},
	{"Sofia", "woman", "man", 31, "Enseignante", "Yoga le matin, vin le soir. Équilibre.", []string{"yoga", "wine", "reading", "meditation"}, 0.02, 0.04},
	{"Lucas", "man", "woman", 28, "Développeur", "Jeux de société, tech et randos. Je fais de bonnes crêpes.", []string{"board_games", "technology", "hiking", "cooking"}, 0.04, 0.00},
	{"Léa", "woman", "man", 27, "Designer", "Musées, concerts et brunchs. Toujours un carnet de croquis.", []string{"art", "museums", "concerts", "coffee"}, -0.03, 0.02},
	{"Hugo", "man", "woman", 30, "Chef de projet", "Course à pied et football. Fan de road trips improvisés.", []string{"running", "football", "road_trips"}, 0.00, 0.05},
	{"Manon", "woman", "man", 33, "Vétérinaire", "Les animaux avant tout, mais je garde de la place pour toi.", []string{"animals", "nature", "gardening", "cooking"}, 0.05, -0.02},
	{"Nathan", "man", "woman", 25, "Étudiant en droit", "Théâtre, danse et bonne humeur.", []string{"theatre", "dancing", "music"}, -0.04, -0.01},
	{"Clara", "woman", "man", 24, "Journaliste", "Curieuse de tout, spécialiste des questions gênantes.", []string{"writing", "languages", "travel", "cinema"}, 0.02, -0.04},
	{"Maxime", "man", "woman", 37, "Restaurateur", "Gastronomie, vin et bonne compagnie.", []string{"food", "wine", "cooking", "music"}, -0.02, -0.04},
	{"Jade", "woman", "woman", 28, "Kiné", "Natation, jardinage et bonne énergie.", []string{"swimming", "gardening", "yoga"}, 0.01, 0.03},
	{"Chloé", "woman", "woman", 30, "Libraire", "Je vis entre deux romans. Je cherche mon prochain chapitre.", []string{"reading", "coffee", "museums"}, -0.01, 0.04},
	{"Axel", "man", "man", 31, "Musicien", "Guitare, concerts et voyages à petit budget.", []string{"music", "concerts", "travel"}, 0.03, 0.03},
	{"Romain", "man", "man", 34, "Pompier", "Sport, grillades et amis. Je cherche du sérieux.", []string{"fitness", "football", "food"}, -0.03, 0.04},
	{"Alex", "non_binary", "woman", 27, "Illustrateur·rice", "Dessin, jeux vidéo et plantes vertes.", []string{"art", "gaming", "gardening"}, 0.02, 0.01},
	{"Sacha", "non_binary", "man", 29, "Ingénieur·e son", "Concerts, synthés et longues balades de nuit.", []string{"music", "concerts", "photography"}, -0.02, -0.02},
	{"Emma", "woman", "man", 40, "Avocate", "Voyage, lecture et bonne table. Directe et drôle.", []string{"travel", "reading", "wine", "languages"}, 0.06, 0.01},
	{"Paul", "man", "woman", 42, "Entrepreneur", "Voile, café et projets fous.", []string{"travel", "coffee", "technology"}, -0.05, 0.03},
	{"Anna", "woman", "man", 23, "Étudiante", "Danse, festivals et fous rires.", []string{"dancing", "concerts", "fashion"}, 0.01, -0.05},
	{"Théo", "man", "woman", 26, "Barista", "Latte art, vélo et vinyles.", []string{"coffee", "cycling", "music"}, 0.02, 0.02},
	{"Lina", "woman", "man", 36, "Cheffe de projet", "Bénévole le week-end, aventurière en vacances.", []string{"volunteering", "hiking", "travel"}, -0.01, 0.01},
	{"Noé", "man", "woman", 38, "Menuisier", "Je construis des choses. Peut-être une histoire ?", []string{"nature", "gardening", "road_trips"}, 0.04, -0.04},
	{"Zoé", "woman", "man", 22, "Apprentie", "Jeux vidéo, séries et pizzas.", []string{"gaming", "series", "food"}, -0.02, 0.05},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo users and their photos")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed: refusing to run with APP_ENV=production")
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
	storage, err := photos.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		if err := purgeDemo(ctx, pool, storage); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := seed(ctx, pool, storage); err != nil {
		log.Fatal(err)
	}
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, storage *photos.LocalStorage) error {
	rows, err := pool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id IN (SELECT id FROM users WHERE is_demo)`)
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
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		return err
	}
	for _, k := range keys {
		_ = storage.Delete(ctx, k)
	}
	fmt.Printf("removed %d demo users and %d photos\n", tag.RowsAffected(), len(keys))
	return nil
}

func seed(ctx context.Context, pool *pgxpool.Pool, storage *photos.LocalStorage) error {
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "demo-password-123"
	}
	lat, lng := envFloat("SEED_LAT", 48.8566), envFloat("SEED_LNG", 2.3522)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	created := 0
	ids := make(map[string]string)
	for i, p := range people {
		email := fmt.Sprintf("demo.%s@demo.invalid", slug(p.first))
		var id string
		err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
		if err == nil {
			ids[p.first] = id
			continue
		}
		id, err = insertPerson(ctx, pool, storage, string(hash), email, p, lat+p.dLat, lng+p.dLng, i)
		if err != nil {
			return fmt.Errorf("seed %s: %w", p.first, err)
		}
		ids[p.first] = id
		created++
	}

	// A few ready-made matches between demo users so chat screens have something to show.
	for _, pair := range [][2]string{{"Camille", "Julien"}, {"Inès", "Thomas"}, {"Sofia", "Lucas"}} {
		if err := matchPair(ctx, pool, ids[pair[0]], ids[pair[1]]); err != nil {
			return err
		}
	}
	fmt.Printf("demo data ready: %d new users (password %q), %d total.\nLog in as e.g. demo.camille@demo.invalid\n", created, password, len(people))
	return nil
}

func insertPerson(ctx context.Context, pool *pgxpool.Pool, storage *photos.LocalStorage, hash, email string, p person, lat, lng float64, index int) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE) RETURNING id`, email, hash).Scan(&id); err != nil {
		return "", err
	}
	birth := time.Now().AddDate(-p.age, 0, -index*11-3)
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, occupation, city, latitude, longitude, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'Démo', round($7::numeric, 2), round($8::numeric, 2), NOW())
	`, id, p.first, birth, p.gender, p.bio, p.job, lat, lng); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, ARRAY[$2]::text[], 18, 60, 100)
	`, id, p.interestedIn); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, id, p.interests); err != nil {
		return "", err
	}

	var keys []string
	for pos := 0; pos < 2; pos++ {
		key, w, h, size, err := writePhoto(ctx, storage, index*2+pos)
		if err != nil {
			return "", err
		}
		keys = append(keys, key)
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, position, content_type, width, height, size_bytes)
			VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		`, id, key, pos, w, h, size); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		for _, k := range keys {
			_ = storage.Delete(ctx, k)
		}
		return "", err
	}
	return id, nil
}

// writePhoto stores a generated abstract portrait placeholder (no real person's photo is ever used).
func writePhoto(ctx context.Context, storage *photos.LocalStorage, seed int) (key string, w, h, size int, err error) {
	w, h = 720, 960
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	hue := float64((seed*47)%360) / 360
	top, bottom := hsv(hue, 0.55, 0.95), hsv(math.Mod(hue+0.12, 1), 0.65, 0.55)
	cx, cy := float64(w)/2, float64(h)*0.40
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		base := mix(top, bottom, t)
		for x := 0; x < w; x++ {
			c := base
			dx, dy := float64(x)-cx, float64(y)-cy
			switch d := math.Hypot(dx, dy); {
			case d < 120: // head
				c = mix(base, color.RGBA{250, 244, 238, 255}, 0.85)
			case y > int(cy)+150 && math.Hypot(dx/1.9, float64(y)-float64(h)*0.95) < 260: // shoulders
				c = mix(base, color.RGBA{250, 244, 238, 255}, 0.7)
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return
	}
	raw := make([]byte, 16)
	if _, err = rand.Read(raw); err != nil {
		return
	}
	key = hex.EncodeToString(raw) + ".jpg"
	err = storage.Put(ctx, key, buf.Bytes())
	return key, w, h, buf.Len(), err
}

func matchPair(ctx context.Context, pool *pgxpool.Pool, a, b string) error {
	low, high := a, b
	if low > high {
		low, high = high, low
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like'), ($2, $1, 'like') ON CONFLICT DO NOTHING
	`, a, b); err != nil {
		return err
	}
	var matchID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
		ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = EXCLUDED.user_a RETURNING id
	`, low, high).Scan(&matchID); err != nil {
		return err
	}
	var convID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, match_id, direct_key) VALUES ('match', $1, $2)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id RETURNING id
	`, matchID, low+":"+high).Scan(&convID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING
	`, convID, low, high); err != nil {
		return err
	}
	var has bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM messages WHERE conversation_id = $1)`, convID).Scan(&has); err != nil {
		return err
	}
	if !has {
		if _, err := tx.Exec(ctx, `
			INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content, created_at) VALUES
			($1, $2, $3, 'Salut ! Ravi(e) de ce match 😊', NOW() - INTERVAL '2 hours'),
			($1, $3, $2, 'Salut ! Moi aussi. Ta dernière sortie, c''était où ?', NOW() - INTERVAL '1 hour')
		`, convID, a, b); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE messages SET read_at = NOW() WHERE conversation_id = $1 AND created_at < NOW() - INTERVAL '90 minutes'`, convID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func slug(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case 'é', 'è', 'ê':
			r = 'e'
		case 'ï':
			r = 'i'
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}

func envFloat(key string, def float64) float64 {
	var v float64
	if _, err := fmt.Sscanf(os.Getenv(key), "%f", &v); err != nil {
		return def
	}
	return v
}

func hsv(h, s, v float64) color.RGBA {
	i := math.Floor(h * 6)
	f := h*6 - i
	p, q, t := v*(1-s), v*(1-f*s), v*(1-(1-f)*s)
	var r, g, b float64
	switch int(i) % 6 {
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

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}
