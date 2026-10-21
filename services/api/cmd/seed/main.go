// Command seed fills a DEVELOPMENT database with clearly marked demo members.
//
// Demo accounts use the reserved domain @demo.alma.invalid and users.is_demo = TRUE, so they can
// never be confused with real members and can be wiped with -reset. The command refuses to run
// when APP_ENV=production.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	demoDomain   = "demo.alma.invalid"
	demoPassword = "DemoPass123!"
)

type demoMember struct {
	first, gender, city string
	lat, lng            float64
	age                 int
	bio                 string
	interests           []string
	into                []string
	hue                 float64
}

var members = []demoMember{
	{"Camille", "woman", "Paris", 48.8566, 2.3522, 29, "Cinéma d'auteur et balades à vélo le dimanche.", []string{"cinema", "sport", "travel"}, []string{"man"}, 12},
	{"Léa", "woman", "Paris", 48.8700, 2.3300, 27, "Je cuisine pour les amis, je lis pour moi.", []string{"cooking", "reading", "music"}, []string{"man", "woman"}, 330},
	{"Inès", "woman", "Montreuil", 48.8610, 2.4430, 31, "Photographe amatrice, randonneuse confirmée.", []string{"photography", "hiking", "nature"}, []string{"man"}, 200},
	{"Sofia", "woman", "Versailles", 48.8014, 2.1301, 33, "Danse, théâtre, et un chat nommé Pixel.", []string{"dancing", "theatre", "animals"}, []string{"man", "non_binary"}, 280},
	{"Jade", "non_binary", "Paris", 48.8460, 2.3380, 28, "Jeux de société, yoga, thé vert.", []string{"board_games", "yoga", "art"}, []string{"man", "woman", "non_binary"}, 150},
	{"Hugo", "man", "Paris", 48.8600, 2.3400, 30, "Développeur le jour, cuisinier le soir.", []string{"tech", "cooking", "gaming"}, []string{"woman"}, 220},
	{"Nathan", "man", "Boulogne-Billancourt", 48.8352, 2.2410, 32, "Running le matin, vin le soir (avec modération).", []string{"fitness", "wine", "travel"}, []string{"woman"}, 30},
	{"Malo", "man", "Saint-Denis", 48.9362, 2.3574, 26, "Musicien. Toujours partant pour un concert.", []string{"music", "art", "cinema"}, []string{"woman", "man"}, 260},
	{"Yanis", "man", "Paris", 48.8750, 2.3600, 35, "Bénévole, grimpeur, lecteur compulsif.", []string{"volunteering", "sport", "reading"}, []string{"woman"}, 100},
	{"Eliott", "man", "Lyon", 45.7640, 4.8357, 29, "Lyonnais d'adoption, amateur de bouchons.", []string{"wine", "hiking", "photography"}, []string{"woman"}, 60},
}

func main() {
	reset := flag.Bool("reset", false, "delete existing demo members first")
	confirm := flag.Bool("confirm", false, "required: acknowledge this writes demo data")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	if !*confirm {
		fmt.Println("This inserts demo members (@"+demoDomain+") into", cfg.DatabaseURL)
		fmt.Println("Re-run with -confirm to proceed (add -reset to wipe previous demo members).")
		os.Exit(2)
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
	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	if *reset {
		wipe(ctx, pool, store)
	}
	signer := storage.NewSigner([]byte(cfg.JWTSecret))
	authSvc := auth.NewService(auth.NewPGRepository(pool), []byte(cfg.JWTSecret), mail.LogMailer{}, store)
	profSvc := profiles.NewService(profiles.NewPGRepository(pool), signer)
	photoSvc := photos.NewService(photos.NewPGRepository(pool), store, signer)

	var ids []string
	for i, m := range members {
		email := fmt.Sprintf("demo%02d@%s", i+1, demoDomain)
		birth := time.Now().AddDate(-m.age, -1, 0).Format("2006-01-02")
		_, user, err := authSvc.Register(ctx, auth.RegisterRequest{Email: email, Password: demoPassword, BirthDate: birth}, "seed", "")
		if err != nil {
			log.Printf("skip %s: %v", email, err)
			continue
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET is_demo = TRUE WHERE id = $1`, user.ID); err != nil {
			log.Fatal(err)
		}
		if _, err := profSvc.Save(ctx, user.ID, profiles.ProfileInput{
			FirstName: m.first, Gender: m.gender, Bio: m.bio, City: m.city, Latitude: &m.lat, Longitude: &m.lng,
			LookingFor: "long_term", Interests: m.interests,
		}); err != nil {
			log.Fatal(err)
		}
		if _, err := profSvc.SavePreferences(ctx, user.ID, profiles.Preferences{
			InterestedIn: m.into, MinAge: 18, MaxAge: 60, Discoverable: true, ShowDistance: true,
		}); err != nil {
			log.Fatal(err)
		}
		for p := 0; p < 2; p++ {
			if _, err := photoSvc.Upload(ctx, user.ID, bytes.NewReader(placeholder(m.hue+float64(p)*25))); err != nil {
				log.Fatal(err)
			}
		}
		ids = append(ids, user.ID)
	}

	// Four men already like Camille (demo01): log in as her, like one of them back and you get a match.
	if len(ids) > 9 {
		for _, id := range ids[5:9] {
			_, _ = pool.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING`, id, ids[0])
		}
	}
	fmt.Printf("Seeded %d demo members. Log in with demo01@%s … demo%02d@%s / %s\n", len(ids), demoDomain, len(members), demoDomain, demoPassword)
}

func wipe(ctx context.Context, pool *pgxpool.Pool, store storage.Storage) {
	rows, err := pool.Query(ctx, `SELECT storage_key, thumb_key FROM photos WHERE user_id IN (SELECT id FROM users WHERE is_demo)`)
	if err == nil {
		for rows.Next() {
			var a, b string
			if rows.Scan(&a, &b) == nil {
				_ = store.Delete(a)
				_ = store.Delete(b)
			}
		}
		rows.Close()
	}
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE is_demo`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Removed %d demo members\n", tag.RowsAffected())
}

// placeholder draws a soft gradient with a neutral avatar disc — no real person's photo is used.
func placeholder(hue float64) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, bottom := hsl(hue, 0.55, 0.62), hsl(hue+40, 0.60, 0.36)
	for y := 0; y < h; y++ {
		t := float64(y) / h
		c := color.RGBA{
			uint8(float64(top.R)*(1-t) + float64(bottom.R)*t),
			uint8(float64(top.G)*(1-t) + float64(bottom.G)*t),
			uint8(float64(top.B)*(1-t) + float64(bottom.B)*t), 255,
		}
		draw.Draw(img, image.Rect(0, y, w, y+1), image.NewUniform(c), image.Point{}, draw.Src)
	}
	cx, cy, r := float64(w)/2, float64(h)*0.42, 130.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if math.Hypot(float64(x)-cx, float64(y)-cy) < r {
				img.Set(x, y, color.RGBA{236, 236, 236, 255})
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360) / 360
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	f := func(t float64) float64 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return color.RGBA{uint8(f(h+1.0/3) * 255), uint8(f(h) * 255), uint8(f(h-1.0/3) * 255), 255}
}
