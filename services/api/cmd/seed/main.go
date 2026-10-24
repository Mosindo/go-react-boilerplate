// Command seed fills a development database with clearly fake demo accounts.
//
// Every demo account uses the reserved domain "seed.invalid" so it can never be confused
// with a real user, and the command refuses to run when APP_ENV=production.
//
//	go run ./cmd/seed          # create demo data (idempotent: existing demo accounts are replaced)
//	go run ./cmd/seed --purge  # only remove demo data
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
	"os"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
)

const (
	demoDomain   = "seed.invalid"
	demoPassword = "Demo-password-1"
)

type demoUser struct {
	name, gender string
	interestedIn []string
	age          int
	city         string
	lat, lon     float64
	bio          string
	interests    []string
	hue          float64
}

var demoUsers = []demoUser{
	{"Camille", "woman", []string{"man"}, 29, "Paris", 48.8566, 2.3522, "Passionnée de cuisine et de randonnée. Toujours partante pour un café.", []string{"cuisine", "randonnee", "cafe"}, 0.02},
	{"Lucas", "man", []string{"woman"}, 31, "Paris", 48.8606, 2.3376, "Photographe le week-end, développeur la semaine.", []string{"photo", "technologie", "voyage"}, 0.58},
	{"Inès", "woman", []string{"man", "woman"}, 27, "Paris", 48.8738, 2.2950, "Danse, cinéma et longues balades.", []string{"danse", "cinema", "nature"}, 0.85},
	{"Hugo", "man", []string{"woman"}, 34, "Boulogne-Billancourt", 48.8352, 2.2412, "Cycliste, amateur de bons vins et de vieux livres.", []string{"velo", "vin", "lecture"}, 0.33},
	{"Sarah", "woman", []string{"man"}, 26, "Lyon", 45.7640, 4.8357, "Yoga le matin, concerts le soir.", []string{"yoga", "concerts", "musique"}, 0.12},
	{"Thomas", "man", []string{"woman", "man"}, 30, "Lyon", 45.7578, 4.8320, "Toujours un jeu de société dans le sac.", []string{"jeux-societe", "cuisine", "cinema"}, 0.65},
	{"Léa", "woman", []string{"man"}, 28, "Paris", 48.8530, 2.3499, "Théâtre, art et petites adresses.", []string{"theatre", "art", "cafe"}, 0.92},
	{"Mehdi", "man", []string{"woman"}, 32, "Paris", 48.8467, 2.3370, "Course à pied et plage dès que possible.", []string{"course", "plage", "sport"}, 0.48},
	{"Alex", "nonbinary", []string{"man", "woman", "nonbinary"}, 29, "Marseille", 43.2965, 5.3698, "Jardinage, bricolage et animaux.", []string{"jardinage", "bricolage", "animaux"}, 0.25},
	{"Chloé", "woman", []string{"woman"}, 33, "Paris", 48.8584, 2.2945, "Montagne, lecture et bénévolat.", []string{"montagne", "lecture", "benevolat"}, 0.73},
	{"Nathan", "man", []string{"man"}, 35, "Paris", 48.8655, 2.3212, "Jeux vidéo, mode et voyages.", []string{"jeux-video", "mode", "voyage"}, 0.07},
	{"Emma", "woman", []string{"man"}, 25, "Paris", 48.8499, 2.3470, "Photographie et nature.", []string{"photo", "nature", "randonnee"}, 0.4},
}

func main() {
	purgeOnly := flag.Bool("purge", false, "only delete demo accounts")
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
	files, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	// Purge first so the command is idempotent. Photo files are removed as well.
	rows, err := pool.Query(ctx, `SELECT ph.storage_key FROM photos ph JOIN users u ON u.id = ph.user_id WHERE u.email LIKE '%@' || $1`, demoDomain)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var key string
		if rows.Scan(&key) == nil {
			_ = files.Delete(key)
			_ = files.Delete(strings.TrimSuffix(key, ".jpg") + "_t.jpg")
		}
	}
	rows.Close()
	tag, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@' || $1`, demoDomain)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("removed %d existing demo accounts\n", tag.RowsAffected())
	if *purgeOnly {
		return
	}

	authSvc := authfeature.NewService(authfeature.NewPGRepository(pool), []byte(cfg.JWTSecret), mailer.LogMailer{}, files)
	profilesSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(pool), files)
	matchingSvc := matchingfeature.NewService(matchingfeature.NewPGRepository(pool), profilesSvc, silentNotifier{}, realtime.NopPublisher{})
	chatSvc := chatfeature.NewService(chatfeature.NewPGRepository(pool), profilesSvc, realtime.NopPublisher{})

	ids := make([]string, len(demoUsers))
	for i, u := range demoUsers {
		email := fmt.Sprintf("demo%02d@%s", i+1, demoDomain)
		_, created, err := authSvc.Register(ctx, email, demoPassword, "seed", "")
		if err != nil {
			log.Fatalf("register %s: %v", email, err)
		}
		ids[i] = created.ID
		birth := time.Now().AddDate(-u.age, 0, -7).Format("2006-01-02")
		if _, err := profilesSvc.Upsert(ctx, created.ID, profilesfeature.UpsertProfileRequest{
			FirstName: u.name, BirthDate: birth, Gender: u.gender, Bio: u.bio, City: u.city, Interests: u.interests,
		}); err != nil {
			log.Fatalf("profile %s: %v", email, err)
		}
		if _, err := profilesSvc.SetPreferences(ctx, created.ID, profilesfeature.Preferences{InterestedIn: u.interestedIn, AgeMin: 18, AgeMax: 60}); err != nil {
			log.Fatalf("preferences %s: %v", email, err)
		}
		if err := profilesSvc.SetLocation(ctx, created.ID, u.lat, u.lon); err != nil {
			log.Fatalf("location %s: %v", email, err)
		}
		for n := 0; n < 2; n++ {
			if _, err := photosSvc.Upload(ctx, created.ID, avatar(u.hue+float64(n)*0.08)); err != nil {
				log.Fatalf("photo %s: %v", email, err)
			}
		}
	}

	// A few pre-made interactions so every feature has something to show.
	mustLike := func(from, to int) matchingfeature.SwipeResult {
		res, err := matchingSvc.Swipe(ctx, ids[from], ids[to], matchingfeature.ActionLike)
		if err != nil {
			log.Fatalf("seed like %d->%d: %v", from, to, err)
		}
		return res
	}
	mustLike(0, 1)
	match := mustLike(1, 0) // Camille <-> Lucas
	if match.Match != nil {
		for _, m := range []struct {
			from    int
			content string
		}{{1, "Salut Camille ! Ton café préféré, c'est lequel ?"}, {0, "Un flat white, sans hésiter. Et toi ?"}} {
			if _, err := chatSvc.Send(ctx, ids[m.from], match.Match.ConversationID, m.content); err != nil {
				log.Fatalf("seed message: %v", err)
			}
		}
	}
	mustLike(6, 1) // Léa already likes Lucas: he will see her first
	mustLike(7, 0) // Mehdi already likes Camille

	fmt.Printf("created %d demo accounts (password for all: %s)\n", len(demoUsers), demoPassword)
	fmt.Printf("  e.g. demo01@%s (Camille, has a match with Lucas), demo02@%s (Lucas)\n", demoDomain, demoDomain)
	fmt.Println("remove them with: go run ./cmd/seed --purge")
	_ = os.Stdout.Sync()
}

type silentNotifier struct{}

func (silentNotifier) Notify(string, notificationsfeature.New) {}

// avatar draws a soft gradient portrait placeholder; no real person's picture is ever used.
func avatar(hue float64) []byte {
	const w, h = 600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / h
			img.Set(x, y, hsl(hue+0.1*float64(x)/w, 0.55, 0.78-0.35*t))
		}
	}
	// simple head and shoulders silhouette
	fill := color.RGBA{255, 255, 255, 140}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x-w/2), float64(y-300)
			head := dx*dx+dy*dy < 110*110
			bx, by := float64(x-w/2), float64(y-640)
			body := bx*bx/(230*230)+by*by/(210*210) < 1
			if head || body {
				img.Set(x, y, blend(img.RGBAAt(x, y), fill))
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func blend(dst, src color.RGBA) color.RGBA {
	a := float64(src.A) / 255
	mix := func(d, s uint8) uint8 { return uint8(float64(d)*(1-a) + float64(s)*a) }
	return color.RGBA{mix(dst.R, src.R), mix(dst.G, src.G), mix(dst.B, src.B), 255}
}

func hsl(h, s, l float64) color.RGBA {
	h -= float64(int(h))
	c := (1 - abs(2*l-1)) * s
	x := c * (1 - abs(mod(h*6, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch int(h * 6) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func mod(a, b float64) float64 {
	for a >= b {
		a -= b
	}
	return a
}
