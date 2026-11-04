// Command seed fills a development database with clearly fake demo profiles.
//
// Demo accounts always use the reserved domain demo.invalid (RFC 2606), so they
// can never collide with a real address, and `-purge` removes exactly those.
// The command refuses to run when APP_ENV=production.
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
	"math/rand"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.invalid"

type demoPerson struct {
	Name, Gender, Bio, City string
	Lat, Lng                float64
	Interests               []string
}

var cities = []struct {
	Name     string
	Lat, Lng float64
}{
	{"Paris", 48.8566, 2.3522}, {"Lyon", 45.7640, 4.8357}, {"Marseille", 43.2965, 5.3698},
	{"Bordeaux", 44.8378, -0.5792}, {"Lille", 50.6292, 3.0573}, {"Nantes", 47.2184, -1.5536},
}

var people = []demoPerson{
	{Name: "Camille", Gender: "woman", Bio: "Toujours un livre dans le sac. Fan de randonnées et de bons petits plats."},
	{Name: "Hugo", Gender: "man", Bio: "Cycliste du dimanche, cuisinier du samedi soir."},
	{Name: "Inès", Gender: "woman", Bio: "Photographe amateure, curieuse de tout."},
	{Name: "Lucas", Gender: "man", Bio: "Musicien et grand voyageur. Je cherche quelqu'un avec qui rire."},
	{Name: "Jade", Gender: "woman", Bio: "Danse, yoga et cinéma d'auteur."},
	{Name: "Théo", Gender: "man", Bio: "Développeur le jour, joueur de société la nuit."},
	{Name: "Manon", Gender: "woman", Bio: "Les animaux avant tout. Et le café."},
	{Name: "Noah", Gender: "man", Bio: "Nature, sport et longues conversations."},
	{Name: "Sasha", Gender: "nonbinary", Bio: "Art, théâtre et balades sans but."},
	{Name: "Louise", Gender: "woman", Bio: "Bénévole, passionnée de mode et de brocantes."},
	{Name: "Adam", Gender: "man", Bio: "Photo, vin et vieux films."},
	{Name: "Eva", Gender: "woman", Bio: "Voyages, lecture et cuisine du monde."},
}

var interestPool = []string{"travel", "cooking", "music", "cinema", "reading", "sport", "hiking", "photography", "art", "dance", "yoga", "animals", "nature", "technology", "theatre", "wine", "cycling"}

func main() {
	count := flag.Int("n", 12, "number of demo profiles to create (max 60)")
	purge := flag.Bool("purge", false, "delete every @"+demoDomain+" account instead of seeding")
	likeUser := flag.String("like", "", "email of a real test account that all demo profiles should already like (to try matching quickly)")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed: APP_ENV=production")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}
	store, err := media.NewStore(uploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		if err := purgeDemo(ctx, pool, store); err != nil {
			log.Fatal(err)
		}
		return
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "DemoPassw0rd"
	}
	if *count < 1 || *count > 60 {
		log.Fatal("-n must be between 1 and 60")
	}
	created, err := seed(ctx, pool, store, *count, password, strings.ToLower(strings.TrimSpace(*likeUser)))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("seeded %d demo profiles (login demo-<n>@%s / %s)\n", created, demoDomain, password)
}

func seed(ctx context.Context, pool *pgxpool.Pool, store *media.Store, n int, password, likeEmail string) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	rng := rand.New(rand.NewSource(42))
	created := 0
	var ids []string
	for i := 0; i < n; i++ {
		p := people[i%len(people)]
		city := cities[(i/2)%len(cities)]
		email := fmt.Sprintf("demo-%d@%s", i+1, demoDomain)
		id, isNew, err := createDemoUser(ctx, pool, store, email, string(hash), p, city.Name, city.Lat, city.Lng, 20+rng.Intn(20), pickInterests(rng), i)
		if err != nil {
			return created, fmt.Errorf("%s: %w", email, err)
		}
		if isNew {
			created++
		}
		ids = append(ids, id)
	}
	if likeEmail != "" {
		var target string
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, likeEmail).Scan(&target); err != nil {
			return created, fmt.Errorf("-like account %q not found: %w", likeEmail, err)
		}
		for _, id := range ids {
			if _, err := pool.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, kind) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, id, target); err != nil {
				return created, err
			}
		}
	}
	return created, nil
}

func pickInterests(rng *rand.Rand) []string {
	perm := rng.Perm(len(interestPool))
	out := make([]string, 0, 4)
	for _, i := range perm[:4] {
		out = append(out, interestPool[i])
	}
	return out
}

func createDemoUser(ctx context.Context, pool *pgxpool.Pool, store *media.Store, email, hash string, p demoPerson, city string, lat, lng float64, age int, interests []string, seedIdx int) (string, bool, error) {
	var existing string
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&existing)
	if err == nil {
		return existing, false, nil
	}
	if err != pgx.ErrNoRows {
		return "", false, err
	}
	data, err := gradientJPEG(seedIdx)
	if err != nil {
		return "", false, err
	}
	key, err := store.Put(data)
	if err != nil {
		return "", false, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`, email, hash).Scan(&id); err != nil {
		_ = store.Delete(key)
		return "", false, err
	}
	birth := time.Now().AddDate(-age, -(seedIdx % 11), -(seedIdx % 27))
	interestedIn := []string{"woman", "man"}
	switch p.Gender {
	case "woman":
		interestedIn = []string{"man"}
	case "man":
		interestedIn = []string{"woman"}
	}
	// Demo people are placed near the city centre with a small deterministic offset.
	dLat := float64(seedIdx%5) * 0.01
	dLng := float64(seedIdx%7) * 0.01
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, p.Name, birth.Format("2006-01-02"), p.Gender, p.Bio, city, geo.Coarsen(lat+dLat), geo.Coarsen(lng+dLng)); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 18, 60, 100)`, id, interestedIn); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)`, id, interests); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (user_id, storage_key, mime_type, size_bytes, width, height, position)
		VALUES ($1, $2, 'image/jpeg', $3, 600, 800, 0)`, id, key, len(data)); err != nil {
		_ = store.Delete(key)
		return "", false, err
	}
	if err := tx.Commit(ctx); err != nil {
		_ = store.Delete(key)
		return "", false, err
	}
	return id, true, nil
}

// gradientJPEG draws a soft abstract placeholder: demo data never ships real people's photos.
func gradientJPEG(idx int) ([]byte, error) {
	const w, h = 600, 800
	palette := [][3]uint8{{198, 98, 70}, {96, 52, 94}, {222, 170, 120}, {70, 100, 110}, {150, 70, 90}, {190, 130, 90}}
	a := palette[idx%len(palette)]
	b := palette[(idx+2)%len(palette)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		for x := 0; x < w; x++ {
			u := float64(x) / float64(w)
			mix := (t*0.8 + u*0.2)
			img.Set(x, y, color.RGBA{
				R: uint8(float64(a[0])*(1-mix) + float64(b[0])*mix),
				G: uint8(float64(a[1])*(1-mix) + float64(b[1])*mix),
				B: uint8(float64(a[2])*(1-mix) + float64(b[2])*mix),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, store *media.Store) error {
	rows, err := pool.Query(ctx, `
		SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.email LIKE '%@' || $1`, demoDomain)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, k)
	}
	rows.Close()
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain)
	if err != nil {
		return err
	}
	for _, k := range keys {
		_ = store.Delete(k)
	}
	fmt.Printf("deleted %d demo accounts\n", tag.RowsAffected())
	return nil
}
