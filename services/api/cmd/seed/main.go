// Command seed fills a development database with clearly fake demo accounts.
//
// Safety rails: it refuses to run when APP_ENV=production, every account uses
// the reserved, undeliverable domain demo.invalid and is flagged users.is_demo,
// and `-purge` removes exactly those accounts (and their photo files).
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
	"os"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoDomain = "demo.invalid"

var asciiFold = strings.NewReplacer("é", "e", "è", "e", "ê", "e", "ï", "i", "ë", "e")

type person struct {
	name      string
	gender    string
	birth     string
	bio       string
	into      []string
	interests []int16
	lat, lng  float64
	city      string
}

var people = []person{
	{"Camille", "woman", "1994-03-12", "Rando le week-end, brunch le dimanche.", []string{"man"}, []int16{1, 6, 15, 9}, 48.8566, 2.3522, "Paris"},
	{"Léa", "woman", "1996-07-21", "Prof de yoga, fan de cinéma d'auteur.", []string{"man", "woman"}, []int16{8, 4, 5}, 48.8606, 2.3376, "Paris"},
	{"Inès", "woman", "1991-11-02", "Cuisine, vin et bonne musique.", []string{"man"}, []int16{2, 16, 3}, 48.8049, 2.1204, "Versailles"},
	{"Manon", "woman", "1998-01-30", "Je cherche quelqu'un pour partager mes balades à vélo.", []string{"man"}, []int16{20, 1, 17}, 48.9, 2.25, "Courbevoie"},
	{"Sarah", "woman", "1989-05-17", "Photographe. J'aime les plats épicés et les gens simples.", []string{"man", "woman"}, []int16{9, 2, 10}, 48.8738, 2.295, "Paris"},
	{"Jade", "woman", "1993-09-09", "Danse, théâtre et fous rires.", []string{"man"}, []int16{13, 14, 25}, 48.85, 2.4, "Montreuil"},
	{"Chloé", "woman", "1995-12-24", "Lectrice compulsive, joueuse de jeux de société.", []string{"man"}, []int16{5, 12, 15}, 48.83, 2.33, "Paris"},
	{"Nora", "woman", "1990-04-04", "Course à pied et podcasts, dans cet ordre.", []string{"woman"}, []int16{19, 30, 7}, 48.89, 2.38, "Pantin"},
	{"Eva", "woman", "1997-06-15", "Amoureuse des animaux, du jardin et du café.", []string{"man"}, []int16{17, 18, 15}, 48.7, 2.5, "Évry"},
	{"Lucas", "man", "1992-02-14", "Ingénieur le jour, grimpeur le week-end.", []string{"woman"}, []int16{28, 21, 6}, 48.8566, 2.3522, "Paris"},
	{"Hugo", "man", "1994-08-08", "Cuisine italienne et football.", []string{"woman"}, []int16{2, 7, 3}, 48.84, 2.28, "Paris"},
	{"Nathan", "man", "1990-10-10", "Musicien amateur, voyageur dans l'âme.", []string{"woman", "man"}, []int16{3, 1, 25}, 48.87, 2.35, "Paris"},
	{"Théo", "man", "1996-05-05", "Gamer repenti, cycliste convaincu.", []string{"woman"}, []int16{11, 20, 22}, 48.92, 2.44, "Saint-Denis"},
	{"Maxime", "man", "1988-12-12", "Photographie argentique et vieux livres.", []string{"woman"}, []int16{9, 5, 23}, 48.81, 2.38, "Vitry"},
	{"Antoine", "man", "1993-03-03", "Randonnée, vin nature et bonnes tables.", []string{"woman"}, []int16{6, 16, 2}, 48.78, 2.1, "Versailles"},
	{"Yanis", "man", "1999-09-19", "Étudiant en sciences, nageur.", []string{"woman"}, []int16{22, 27, 30}, 48.85, 2.34, "Paris"},
	{"Sacha", "non_binary", "1995-07-07", "Théâtre, art contemporain et thé glacé.", []string{"man", "woman", "non_binary"}, []int16{14, 10, 15}, 48.86, 2.36, "Paris"},
	{"Alex", "non_binary", "1992-01-01", "Bénévole, jardinier du dimanche.", []string{"man", "woman", "non_binary"}, []int16{26, 18, 8}, 48.88, 2.31, "Paris"},
}

func main() {
	purge := flag.Bool("purge", false, "delete all demo accounts instead of creating them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed refuses to run when APP_ENV=production")
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
	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}
	profileSvc := profiles.NewService(profiles.NewPGRepository(pool), store)

	if *purge {
		purgeDemo(ctx, pool, profileSvc)
		return
	}

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "demo-password-123"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	all := append([]person{{
		"Démo", "woman", "1994-06-01", "Compte de démonstration : connectez-vous avec demo@demo.invalid.",
		[]string{"man", "woman", "non_binary"}, []int16{1, 3, 15}, 48.8566, 2.3522, "Paris",
	}}, people...)

	created := 0
	for i, p := range all {
		email := strings.ToLower(asciiFold.Replace(p.name)) + "@" + demoDomain
		if i == 0 {
			email = "demo@" + demoDomain
		}
		var userID string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, is_demo) VALUES ($1, $2, TRUE)
			ON CONFLICT (email) DO NOTHING RETURNING id`, email, string(hash)).Scan(&userID)
		if err != nil {
			continue // already seeded
		}
		_, err = profileSvc.Upsert(ctx, userID, profiles.UpsertProfileRequest{
			FirstName: p.name, BirthDate: p.birth, Gender: p.gender, Bio: p.bio, City: p.city,
			Location: &profiles.Coordinates{Lat: p.lat, Lng: p.lng}, Interests: p.interests,
		})
		if err != nil {
			log.Fatalf("profile %s: %v", p.name, err)
		}
		if _, err := profileSvc.UpdatePreferences(ctx, userID, profiles.UpdatePreferencesRequest{
			InterestedIn: p.into, MinAge: 18, MaxAge: 60, ShowDistance: true,
		}); err != nil {
			log.Fatalf("preferences %s: %v", p.name, err)
		}
		for n := 0; n < 2; n++ {
			if _, err := profileSvc.AddPhoto(ctx, userID, avatar(p.name, n), nil); err != nil {
				log.Fatalf("photo %s: %v", p.name, err)
			}
		}
		created++
	}

	// Make the demo account instantly testable: a few demo profiles already like it.
	_, err = pool.Exec(ctx, `
		INSERT INTO swipes (from_user, to_user, action)
		SELECT u.id, d.id, 'like'
		FROM users d, users u
		JOIN profiles p ON p.user_id = u.id
		WHERE d.email = 'demo@`+demoDomain+`' AND u.is_demo AND u.id <> d.id AND p.first_name IN ('Lucas', 'Hugo', 'Nathan', 'Sacha')
		ON CONFLICT DO NOTHING`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("seeded %d demo accounts (domain %s)\n", created, demoDomain)
	fmt.Printf("log in with demo@%s / %s\n", demoDomain, password)
}

// purgeDemo removes every flagged demo account; cascades remove their data and
// the photo files are deleted once the rows are gone.
func purgeDemo(ctx context.Context, pool *pgxpool.Pool, svc *profiles.Service) {
	rows, err := pool.Query(ctx, `SELECT id FROM users WHERE is_demo`)
	if err != nil {
		log.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		cleanup, err := svc.PrepareDeletion(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1 AND is_demo`, id); err != nil {
			log.Fatal(err)
		}
		cleanup()
	}
	fmt.Printf("removed %d demo accounts\n", len(ids))
}

// avatar draws a deterministic gradient from the name so demo photos are
// recognisably fake and need no binary assets in the repository.
func avatar(name string, variant int) []byte {
	seed := variant * 97
	for _, r := range name {
		seed = seed*31 + int(r)
	}
	r0, g0, b0 := uint8(seed), uint8(seed>>3), uint8(seed>>5)
	img := image.NewRGBA(image.Rect(0, 0, 600, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 600; x++ {
			t := float64(y) / 800
			img.Set(x, y, color.RGBA{
				R: uint8(float64(r0)*(1-t) + 40*t),
				G: uint8(float64(g0)*(1-t) + 30*t),
				B: uint8(float64(b0)*(1-t) + 90*t),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}
