// Command seed creates clearly-flagged demonstration members so the app can be
// tried locally without inviting real people.
//
// Every demo account:
//   - has users.is_demo = TRUE,
//   - uses an e-mail under the reserved ".invalid" TLD (it can never receive mail),
//   - starts its bio with "[Profil de démonstration]".
//
// The command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed                         # create 24 demo members around Paris
//	go run ./cmd/seed -like you@example.com   # demo members already like that account
//	go run ./cmd/seed -purge                  # delete every demo member and their photos
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

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/people"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	demoDomain   = "demo.lueur.invalid"
	demoBioTag   = "[Profil de démonstration] "
	demoPassword = "DemoPassword1"
)

var (
	womenNames = []string{"Camille", "Léa", "Manon", "Chloé", "Inès", "Sarah", "Jade", "Louise", "Zoé", "Alice", "Lina", "Emma"}
	menNames   = []string{"Hugo", "Lucas", "Nathan", "Louis", "Gabriel", "Arthur", "Jules", "Adam", "Paul", "Tom", "Noah", "Raphaël"}
	nbNames    = []string{"Alex", "Sacha", "Charlie", "Eden", "Noa", "Andrea"}
	bios       = []string{
		"Toujours partant·e pour un café en terrasse et une longue balade.",
		"Je cuisine mieux que je ne chante, et je chante beaucoup.",
		"Amoureux·se des musées le dimanche matin et des concerts le samedi soir.",
		"Randos, bouquins, et la recherche du meilleur croissant de la ville.",
		"Je cherche quelqu'un avec qui refaire le monde autour d'un plat partagé.",
		"Un pied dans la nature, l'autre dans les salles de cinéma.",
	}
	jobs      = []string{"Architecte", "Infirmier·e", "Développeur·se", "Professeur·e", "Graphiste", "Chef·fe de projet", "Musicien·ne", "Libraire"}
	goals     = []string{"long_term", "short_term", "friendship", "unsure"}
	interests = []string{"hiking", "climbing", "cycling", "running", "yoga", "cooking", "wine", "coffee", "travel", "reading", "cinema", "museums", "photography", "concerts", "jazz", "board_games", "video_games", "animals", "ecology", "meditation"}
	palette   = [][2]color.RGBA{
		{{0xC8, 0x55, 0x3D, 0xFF}, {0xF2, 0xC1, 0x8D, 0xFF}},
		{{0x2F, 0x48, 0x58, 0xFF}, {0x86, 0xBB, 0xD8, 0xFF}},
		{{0x58, 0x3E, 0x5B, 0xFF}, {0xE8, 0xA8, 0x9C, 0xFF}},
		{{0x3D, 0x5A, 0x40, 0xFF}, {0xC9, 0xD7, 0x8F, 0xFF}},
	}
)

func main() {
	count := flag.Int("count", 24, "number of demo members to create")
	purge := flag.Bool("purge", false, "delete every demo member instead of creating new ones")
	like := flag.String("like", "", "e-mail of an existing account every demo member will like")
	lat := flag.Float64("lat", 48.8566, "latitude around which demo members are placed")
	lng := flag.Float64("lng", 2.3522, "longitude around which demo members are placed")
	flag.Parse()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		log.Fatal("refusing to seed demo data with APP_ENV=production")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	mediaDir := strings.TrimSpace(os.Getenv("MEDIA_DIR"))
	if mediaDir == "" {
		mediaDir = "./data/media"
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
	storage, err := media.NewDiskStorage(mediaDir)
	if err != nil {
		log.Fatal(err)
	}

	if *purge {
		n, err := purgeDemo(ctx, pool, storage)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("deleted %d demo members", n)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	var likeTarget string
	if *like != "" {
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(*like))).Scan(&likeTarget); err != nil {
			log.Fatalf("account %q not found: %v", *like, err)
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	batch := time.Now().Unix()
	for i := 0; i < *count; i++ {
		gender, name := pickIdentity(rng, i)
		email := fmt.Sprintf("demo-%d-%02d@%s", batch, i, demoDomain)
		if err := createMember(ctx, pool, storage, rng, memberSpec{
			email: email, hash: string(hash), firstName: name, gender: gender,
			lat: *lat, lng: *lng, likeTarget: likeTarget, paletteIndex: i,
		}); err != nil {
			log.Fatalf("create %s: %v", email, err)
		}
	}
	log.Printf("created %d demo members (password %q, e-mails @%s)", *count, demoPassword, demoDomain)
}

func pickIdentity(rng *rand.Rand, i int) (string, string) {
	switch i % 5 {
	case 0, 2:
		return "woman", womenNames[rng.Intn(len(womenNames))]
	case 1, 3:
		return "man", menNames[rng.Intn(len(menNames))]
	default:
		return "nonbinary", nbNames[rng.Intn(len(nbNames))]
	}
}

type memberSpec struct {
	email, hash, firstName, gender string
	lat, lng                       float64
	likeTarget                     string
	paletteIndex                   int
}

func createMember(ctx context.Context, pool *pgxpool.Pool, storage media.Storage, rng *rand.Rand, spec memberSpec) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, organization_id, is_demo)
		VALUES ($1, $2, (SELECT id FROM organizations WHERE slug = 'default'), TRUE)
		RETURNING id
	`, spec.email, spec.hash).Scan(&userID); err != nil {
		return err
	}

	age := 21 + rng.Intn(25)
	birthdate := time.Now().AddDate(-age, -rng.Intn(11), -rng.Intn(27))
	// Spread members within ~25 km, rounded like real locations.
	angle := rng.Float64() * 2 * math.Pi
	radiusKm := rng.Float64() * 25
	lat := people.RoundCoordinate(spec.lat + radiusKm/111*math.Sin(angle))
	lng := people.RoundCoordinate(spec.lng + radiusKm/(111*math.Cos(spec.lat*math.Pi/180))*math.Cos(angle))
	height := 155 + rng.Intn(40)

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, height_cm, relationship_goal,
		                      latitude, longitude, location_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'Paris', $7, $8, $9, $10, NOW())
	`, userID, spec.firstName, birthdate, spec.gender, demoBioTag+bios[rng.Intn(len(bios))], jobs[rng.Intn(len(jobs))],
		height, goals[rng.Intn(len(goals))], lat, lng); err != nil {
		return err
	}

	interestedIn := [][]string{{"man"}, {"woman"}, {"woman", "man", "nonbinary"}}[rng.Intn(3)]
	if spec.likeTarget != "" {
		interestedIn = []string{"woman", "man", "nonbinary"}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, 18, 60, 100)
	`, userID, interestedIn); err != nil {
		return err
	}

	picked := rng.Perm(len(interests))[:3+rng.Intn(4)]
	slugs := make([]string, 0, len(picked))
	for _, idx := range picked {
		slugs = append(slugs, interests[idx])
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, userID, slugs); err != nil {
		return err
	}

	for position := 0; position < 2; position++ {
		data, w, h, err := demoPhoto(spec.firstName, spec.paletteIndex+position)
		if err != nil {
			return err
		}
		key, err := media.NewKey()
		if err != nil {
			return err
		}
		if err := storage.Save(key, bytes.NewReader(data)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes) VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, position, key, w, h, len(data)); err != nil {
			return err
		}
	}

	if spec.likeTarget != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO swipes (actor_user_id, target_user_id, decision) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING
		`, userID, spec.likeTarget); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// demoPhoto renders an abstract portrait card with the member's initial and a
// visible "DÉMO" label, so it can never be mistaken for a real photo.
func demoPhoto(name string, paletteIndex int) ([]byte, int, int, error) {
	const w, h = 800, 1000
	colors := palette[paletteIndex%len(palette)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		c := color.RGBA{
			R: uint8(float64(colors[0].R)*(1-t) + float64(colors[1].R)*t),
			G: uint8(float64(colors[0].G)*(1-t) + float64(colors[1].G)*t),
			B: uint8(float64(colors[0].B)*(1-t) + float64(colors[1].B)*t),
			A: 0xFF,
		}
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	drawScaledText(img, []rune(name)[0:1], 36, image.Pt(w/2, h/2-60))
	drawScaledText(img, []rune("DEMO"), 10, image.Pt(w/2, h-140))

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}

func drawScaledText(dst *image.RGBA, text []rune, scale int, center image.Point) {
	face := basicfont.Face7x13
	small := image.NewRGBA(image.Rect(0, 0, 7*len(text), 13))
	d := &font.Drawer{Dst: small, Src: image.NewUniform(color.RGBA{0xFF, 0xFF, 0xFF, 0xE6}), Face: face, Dot: fixed.P(0, 11)}
	d.DrawString(strings.ToUpper(string(text)))
	sw, sh := small.Bounds().Dx()*scale, small.Bounds().Dy()*scale
	target := image.Rect(center.X-sw/2, center.Y-sh/2, center.X+sw/2, center.Y+sh/2)
	draw.NearestNeighbor.Scale(dst, target, small, small.Bounds(), draw.Over, nil)
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, storage media.Storage) (int64, error) {
	rows, err := pool.Query(ctx, `SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.is_demo`)
	if err != nil {
		return 0, err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return 0, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if _, err := pool.Exec(ctx, `
		DELETE FROM conversations WHERE id IN (
			SELECT cp.conversation_id FROM conversation_participants cp JOIN users u ON u.id = cp.user_id WHERE u.is_demo
		)
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
	return tag.RowsAffected(), nil
}
