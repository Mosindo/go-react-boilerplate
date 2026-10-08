// Command seed loads (or removes) clearly-labelled demo accounts for local development.
//
// Demo users all have an email ending in @demo.invalid (a reserved, undeliverable domain), so they can
// never be mistaken for real people and can be removed with `-purge`. The command refuses to run
// when APP_ENV=production.
//
//	go run ./cmd/seed -yes            # create demo users
//	go run ./cmd/seed -purge          # delete them again
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
	"os"
	"time"

	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "@demo.invalid"

type demoUser struct {
	Name      string
	Gender    string
	Age       int
	Seeking   []string
	City      string
	Lat, Lng  float64
	Bio       string
	Looking   string
	Job       string
	Interests []string
	Hue       int
}

var demoUsers = []demoUser{
	{"Camille", "woman", 28, []string{"man"}, "Paris", 48.8566, 2.3522, "Grimpeuse le week-end, développeuse la semaine. Je cherche quelqu'un qui aime les marchés et les longues balades.", "relationship", "Développeuse", []string{"hiking", "cooking", "cinema", "travel"}, 10},
	{"Hugo", "man", 31, []string{"woman"}, "Paris", 48.8738, 2.2950, "Cuisinier amateur, fan de vélo et de bons cafés. Mon plat signature : les pâtes au citron.", "relationship", "Architecte", []string{"cooking", "cycling", "coffee", "art"}, 40},
	{"Inès", "woman", 26, []string{"man", "woman"}, "Boulogne-Billancourt", 48.8352, 2.2410, "Photographe, curieuse de tout. Dites-moi votre dernier coup de cœur culturel.", "open", "Photographe", []string{"photography", "art", "cinema", "reading"}, 80},
	{"Lucas", "man", 29, []string{"woman"}, "Montreuil", 48.8610, 2.4410, "Musicien, chasseur de vinyles. Je joue de la basse (pas dans votre immeuble).", "open", "Ingénieur du son", []string{"music", "dancing", "coffee", "technology"}, 120},
	{"Sarah", "woman", 33, []string{"man"}, "Versailles", 48.8049, 2.1204, "Prof de yoga, amoureuse de la nature et des chiens. Ambiance calme et bienveillante.", "relationship", "Professeure de yoga", []string{"yoga", "nature", "animals", "reading"}, 160},
	{"Théo", "man", 27, []string{"woman", "non_binary"}, "Saint-Denis", 48.9362, 2.3574, "Fan de jeux de société et de rando. Je gagne rarement mais je ris toujours.", "friends", "Enseignant", []string{"board_games", "hiking", "gaming", "languages"}, 200},
	{"Alex", "non_binary", 30, []string{"man", "woman", "non_binary"}, "Paris", 48.8530, 2.3499, "Danse contemporaine et théâtre. Toujours partant pour une expo ou un concert.", "open", "Chorégraphe", []string{"dancing", "theatre", "art", "music"}, 240},
	{"Manon", "woman", 24, []string{"man"}, "Nanterre", 48.8924, 2.2071, "Étudiante en médecine, accro au running et aux séries. Bonne humeur garantie.", "casual", "Étudiante", []string{"running", "cinema", "travel", "coffee"}, 280},
	{"Julien", "man", 35, []string{"woman"}, "Vincennes", 48.8473, 2.4393, "Papa d'un chat, passionné de vin et de cuisine. Je cherche une complice pour les dîners improvisés.", "relationship", "Chef de projet", []string{"wine", "cooking", "travel", "volunteering"}, 320},
	{"Léa", "woman", 29, []string{"man", "woman"}, "Paris", 48.8606, 2.3376, "Bénévole dans une association de quartier, passionnée de langues et de voyages lents.", "relationship", "Traductrice", []string{"languages", "volunteering", "travel", "reading"}, 350},
}

func main() {
	yes := flag.Bool("yes", false, "confirm creating demo users")
	purge := flag.Bool("purge", false, "delete all demo users and their files")
	flag.Parse()

	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("refusing to run the demo seed with APP_ENV=production")
	}
	if !*yes && !*purge {
		log.Fatal("pass -yes to create demo users or -purge to remove them")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 15*time.Second, time.Second)
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
	photos := photosfeature.NewService(photosfeature.NewPGRepository(pool), store, nil)

	if *purge {
		if err := purgeDemo(ctx, pool, photos); err != nil {
			log.Fatal(err)
		}
		return
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "Demo-Password-1"
	}
	if err := seed(ctx, pool, photos, password); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nDemo accounts ready. Log in with any email above and password %q.\n", password)
}

func seed(ctx context.Context, pool *pgxpool.Pool, photos *photosfeature.Service, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 11)
	if err != nil {
		return err
	}
	profiles := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	for _, u := range demoUsers {
		email := fmt.Sprintf("%s%s", slug(u.Name), demoDomain)
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash) VALUES ($1, $2)
			ON CONFLICT (email) DO NOTHING RETURNING id`, email, string(hash)).Scan(&id)
		if err != nil {
			fmt.Printf("skip   %-28s (already exists)\n", email)
			continue
		}
		birth := time.Now().AddDate(-u.Age, 0, -10).Format("2006-01-02")
		if _, err := profiles.UpdateProfile(ctx, id, profilesfeature.UpdateProfileRequest{
			FirstName: u.Name, Birthdate: birth, Gender: u.Gender, Bio: u.Bio, Occupation: u.Job, LookingFor: u.Looking, City: u.City, Interests: u.Interests,
		}); err != nil {
			return fmt.Errorf("profile %s: %w", u.Name, err)
		}
		dist := 100
		if _, err := profiles.UpdatePreferences(ctx, id, profilesfeature.UpdatePreferencesRequest{
			InterestedIn: u.Seeking, MinAge: 20, MaxAge: 50, MaxDistanceKm: &dist, IsVisible: true, ShowDistance: true,
		}); err != nil {
			return fmt.Errorf("prefs %s: %w", u.Name, err)
		}
		lat, lng := u.Lat, u.Lng
		if _, err := profiles.SetLocation(ctx, id, profilesfeature.UpdateLocationRequest{Latitude: &lat, Longitude: &lng, City: u.City}); err != nil {
			return fmt.Errorf("location %s: %w", u.Name, err)
		}
		for i := 0; i < 3; i++ {
			if _, err := photos.Upload(ctx, id, portrait(u.Hue+i*18, i)); err != nil {
				return fmt.Errorf("photo %s: %w", u.Name, err)
			}
		}
		fmt.Printf("create %-28s %s, %d, %s\n", email, u.Name, u.Age, u.City)
	}
	return nil
}

func purgeDemo(ctx context.Context, pool *pgxpool.Pool, photos *photosfeature.Service) error {
	rows, err := pool.Query(ctx, `SELECT id FROM users WHERE email LIKE '%' || $1`, demoDomain)
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
		cleanup, err := photos.PrepareDelete(ctx, id)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
			return err
		}
		cleanup()
	}
	fmt.Printf("removed %d demo users\n", len(ids))
	return nil
}

func slug(name string) string {
	out := []rune{}
	for _, r := range name {
		switch r {
		case 'è', 'é', 'ê', 'ë':
			r = 'e'
		case 'î', 'ï', 'í':
			r = 'i'
		case 'à', 'â':
			r = 'a'
		}
		if r >= 'A' && r <= 'Z' {
			r += 32
		}
		out = append(out, r)
	}
	return string(out)
}

// portrait renders a deterministic abstract avatar so the demo needs no external image assets.
func portrait(hue, variant int) []byte {
	const w, h = 720, 960
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, bottom := hsv(float64(hue), 0.55, 0.95), hsv(float64(hue)+40, 0.65, 0.55)
	for y := 0; y < h; y++ {
		t := float64(y) / h
		c := color.RGBA{lerp(top.R, bottom.R, t), lerp(top.G, bottom.G, t), lerp(top.B, bottom.B, t), 255}
		draw.Draw(img, image.Rect(0, y, w, y+1), &image.Uniform{c}, image.Point{}, draw.Src)
	}
	cx, cy := w/2+(variant-1)*60, 380
	fill(img, cx, cy, 130, color.RGBA{255, 255, 255, 70})
	fill(img, cx, cy+330, 260, color.RGBA{255, 255, 255, 55})
	fill(img, cx, cy, 90, color.RGBA{255, 255, 255, 120})
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}

func lerp(a, b uint8, t float64) uint8 { return uint8(float64(a)*(1-t) + float64(b)*t) }

func fill(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	b := img.Bounds()
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r && image.Pt(x, y).In(b) {
				bg := img.RGBAAt(x, y)
				a := uint32(c.A)
				img.SetRGBA(x, y, color.RGBA{
					uint8((uint32(c.R)*a + uint32(bg.R)*(255-a)) / 255),
					uint8((uint32(c.G)*a + uint32(bg.G)*(255-a)) / 255),
					uint8((uint32(c.B)*a + uint32(bg.B)*(255-a)) / 255), 255})
			}
		}
	}
}

func hsv(h, s, v float64) color.RGBA {
	for h >= 360 {
		h -= 360
	}
	c := v * s
	x := c * (1 - abs(mod(h/60, 2)-1))
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

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func mod(a, b float64) float64 {
	for a >= b {
		a -= b
	}
	return a
}
