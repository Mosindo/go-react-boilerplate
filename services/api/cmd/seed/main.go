// Command seed creates clearly labelled demo accounts (is_demo = true, @demo.invalid addresses) so the
// app can be tried locally without real users. It refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed          # create (or refresh) the demo accounts
//	go run ./cmd/seed -reset   # delete every demo account and its data
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.invalid"

type person struct {
	slug      string
	name      string
	gender    string
	interest  []string
	age       int
	bio       string
	city      string
	lat, lng  float64
	interests []string
	hue       float64
}

var people = []person{
	{"alex", "Alex", "man", []string{"woman", "nonbinary"}, 31, "Compte de démonstration. Randonnées le week-end, cuisine le dimanche.", "Paris", 48.8566, 2.3522, []string{"hiking", "cooking", "music"}, 20},
	{"sam", "Sam", "woman", []string{"man", "nonbinary"}, 29, "Compte de démonstration. Cinéma, café et longues balades.", "Paris", 48.8606, 2.3376, []string{"cinema", "coffee", "travel"}, 330},
	{"camille", "Camille", "woman", []string{"man"}, 27, "Illustratrice, passionnée de musées et de vélo.", "Paris", 48.87, 2.33, []string{"art", "museums", "cycling"}, 300},
	{"lea", "Léa", "woman", []string{"man", "woman"}, 25, "Yoga au lever du jour, voyages dès que possible.", "Montreuil", 48.86, 2.44, []string{"yoga", "travel", "photography"}, 280},
	{"nina", "Nina", "woman", []string{"man"}, 33, "Cheffe de projet le jour, joueuse de jeux de société la nuit.", "Boulogne-Billancourt", 48.84, 2.24, []string{"board-games", "wine", "reading"}, 340},
	{"jade", "Jade", "woman", []string{"man"}, 30, "Danseuse amateure, fan de concerts.", "Paris", 48.85, 2.37, []string{"dancing", "concerts", "music"}, 350},
	{"manon", "Manon", "woman", []string{"man", "woman", "nonbinary"}, 26, "Développeuse, podcasts et course à pied.", "Saint-Denis", 48.93, 2.36, []string{"tech", "podcasts", "running"}, 310},
	{"sarah", "Sarah", "woman", []string{"man"}, 35, "Amoureuse des animaux et du jardinage.", "Versailles", 48.8, 2.13, []string{"animals", "gardening", "cooking"}, 10},
	{"ines", "Inès", "woman", []string{"man"}, 28, "Théâtre, langues étrangères et escapades.", "Paris", 48.88, 2.35, []string{"theatre", "languages", "travel"}, 320},
	{"louise", "Louise", "woman", []string{"man"}, 31, "Natation le matin, lecture le soir.", "Vincennes", 48.85, 2.44, []string{"swimming", "reading", "coffee"}, 15},
	{"maya", "Maya", "woman", []string{"man", "nonbinary"}, 24, "Bénévole, grande randonneuse, petite cuisinière.", "Paris", 48.83, 2.32, []string{"volunteering", "hiking", "diy"}, 340},
	{"lucas", "Lucas", "man", []string{"woman"}, 30, "Photographe, amateur de bon café.", "Paris", 48.86, 2.35, []string{"photography", "coffee", "travel"}, 200},
	{"hugo", "Hugo", "man", []string{"woman"}, 34, "Cuisinier amateur, fan de vin et de concerts.", "Paris", 48.87, 2.38, []string{"cooking", "wine", "concerts"}, 210},
	{"theo", "Théo", "man", []string{"woman"}, 28, "Vélo, bricolage et jeux vidéo.", "Montrouge", 48.81, 2.32, []string{"cycling", "diy", "gaming"}, 190},
	{"nathan", "Nathan", "man", []string{"woman", "man"}, 32, "Musées, cinéma et marche en ville.", "Paris", 48.85, 2.34, []string{"museums", "cinema", "running"}, 220},
	{"adam", "Adam", "man", []string{"woman"}, 27, "Sport tous les jours, curieux de tout.", "Créteil", 48.79, 2.46, []string{"sports", "swimming", "tech"}, 180},
	{"ethan", "Ethan", "man", []string{"woman"}, 36, "Camping, feux de bois et bonne musique.", "Paris", 48.89, 2.34, []string{"camping", "music", "hiking"}, 230},
	{"noe", "Noé", "man", []string{"woman"}, 29, "Théâtre, danse et cuisine du monde.", "Paris", 48.84, 2.36, []string{"theatre", "dancing", "cooking"}, 170},
	{"jules", "Jules", "man", []string{"woman", "nonbinary"}, 26, "Gamer et fan de podcasts.", "Paris", 48.86, 2.31, []string{"gaming", "podcasts", "board-games"}, 240},
	{"robin", "Robin", "nonbinary", []string{"man", "woman", "nonbinary"}, 28, "Artiste, lecteur·rice, toujours en balade.", "Paris", 48.85, 2.35, []string{"art", "reading", "yoga"}, 260},
	{"alix", "Alix", "nonbinary", []string{"woman", "nonbinary"}, 30, "Concerts, vélo et bons restos.", "Paris", 48.87, 2.36, []string{"concerts", "cycling", "wine"}, 270},
}

func main() {
	reset := flag.Bool("reset", false, "delete all demo accounts and exit")
	flag.Parse()

	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("refusing to seed demo data when APP_ENV=production")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is required")
	}
	password := os.Getenv("DEMO_PASSWORD")
	if password == "" {
		password = "demo-password-1"
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, url, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		log.Fatal(err)
	}
	if *reset {
		fmt.Printf("deleted %d demo accounts\n", tag.RowsAffected())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	if err := seed(ctx, pool, string(hash)); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created %d demo accounts (password %q)\n", len(people), password)
	fmt.Printf("try: demo.alex@%s or demo.sam@%s\n", demoDomain, demoDomain)
}

func seed(ctx context.Context, pool *pgxpool.Pool, passwordHash string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ids := map[string]string{}
	for _, p := range people {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE) RETURNING id
		`, "demo."+p.slug+"@"+demoDomain, passwordHash).Scan(&id)
		if err != nil {
			return fmt.Errorf("user %s: %w", p.slug, err)
		}
		ids[p.slug] = id

		birth := time.Now().AddDate(-p.age, 0, -20)
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, p.name, birth, p.gender, p.bio, p.city, math.Round(p.lat*100)/100, math.Round(p.lng*100)/100); err != nil {
			return fmt.Errorf("profile %s: %w", p.slug, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km) VALUES ($1, $2, 21, 45, 100)
		`, id, p.interest); err != nil {
			return fmt.Errorf("preferences %s: %w", p.slug, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, id, p.interests); err != nil {
			return fmt.Errorf("interests %s: %w", p.slug, err)
		}
		for position := 0; position < 2; position++ {
			full, thumb, err := portrait(p.hue+float64(position)*35, p.name, position)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO photos (user_id, position, content, thumb) VALUES ($1, $2, $3, $4)`, id, position, full, thumb); err != nil {
				return fmt.Errorf("photo %s: %w", p.slug, err)
			}
		}
	}

	// Demo users have already liked the two demo entry accounts, so a tester can get a match straight away.
	// Nothing ever links a demo account to a real user.
	for _, p := range people[2:] {
		for _, target := range []string{"alex", "sam"} {
			if !compatible(p, byslug(target)) {
				continue
			}
			if err := like(ctx, tx, ids[p.slug], ids[target]); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func byslug(slug string) person {
	for _, p := range people {
		if p.slug == slug {
			return p
		}
	}
	return person{}
}

func compatible(a, b person) bool {
	return contains(a.interest, b.gender) && contains(b.interest, a.gender)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func like(ctx context.Context, tx pgx.Tx, from, to string) error {
	_, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, from, to)
	return err
}

// portrait draws a generated 3:4 illustration (gradient plus a soft silhouette); no real photos are shipped.
func portrait(hue float64, label string, variant int) (full, thumb []byte, err error) {
	render := func(w, h int) ([]byte, error) {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		top, bottom := hsv(hue, 0.55, 0.95), hsv(hue+40, 0.75, 0.55)
		for y := 0; y < h; y++ {
			t := float64(y) / float64(h-1)
			c := mix(top, bottom, t)
			for x := 0; x < w; x++ {
				img.Set(x, y, c)
			}
		}
		shade := color.RGBA{255, 255, 255, 70}
		cx, cy := float64(w)*(0.5+0.12*float64(variant)), float64(h)*0.42
		radius := float64(w) * 0.17
		fillCircle(img, cx, cy, radius, shade)
		fillCircle(img, cx, float64(h)*0.95, float64(w)*0.42, shade)
		return encode(img)
	}
	if full, err = render(600, 800); err != nil {
		return nil, nil, err
	}
	thumb, err = render(240, 320)
	_ = label
	return full, thumb, err
}

func encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fillCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	overlay := image.NewUniform(c)
	b := img.Bounds()
	mask := image.NewAlpha(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			if dx*dx+dy*dy <= r*r {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
	}
	draw.DrawMask(img, b, overlay, image.Point{}, mask, image.Point{}, draw.Over)
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 255}
}

func hsv(h, s, v float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
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
