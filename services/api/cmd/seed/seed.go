package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"regexp"
	"strings"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"time"
)

// Options configure a seeding run.
type Options struct {
	Count     int
	Domain    string
	CenterLat float64
	CenterLng float64
	Password  string
	Now       time.Time
}

type person struct {
	name   string
	gender string
}

var people = []person{
	{"Alice", "woman"}, {"Bruno", "man"}, {"Camille", "non_binary"}, {"David", "man"},
	{"Emma", "woman"}, {"Farid", "man"}, {"Gabrielle", "woman"}, {"Hugo", "man"},
	{"Ines", "woman"}, {"Jules", "non_binary"}, {"Karim", "man"}, {"Lea", "woman"},
	{"Mathis", "man"}, {"Nadia", "woman"}, {"Oscar", "man"}, {"Pauline", "woman"},
	{"Quentin", "man"}, {"Rania", "woman"}, {"Sacha", "non_binary"}, {"Théo", "man"},
	{"Uma", "woman"}, {"Victor", "man"}, {"Wendy", "woman"}, {"Yanis", "man"},
}

var cities = []string{"Lyon", "Villeurbanne", "Caluire-et-Cuire", "Bron", "Écully", "Vénissieux"}

var bios = []string{
	"Weekend hiker, weekday coffee addict.",
	"Looking for someone to share good food and bad puns.",
	"Amateur photographer and full-time daydreamer.",
	"I cook better than I dance, but I dance anyway.",
	"Always planning the next trip.",
	"Board games, long walks and loud music.",
	"Bookworm with a soft spot for old cinema.",
	"Runner in the morning, couch person at night.",
}

var emailRe = regexp.MustCompile(`^[a-z0-9.+-]+\.[a-z]+$`)

func email(n int, domain string) string { return fmt.Sprintf("demo+%d@%s", n, domain) }

// Seed creates the demo users that do not exist yet. It is deterministic and idempotent.
func Seed(ctx context.Context, pool *pgxpool.Pool, store storage.Storage, o Options) (created, skipped int, err error) {
	if !emailRe.MatchString(o.Domain) {
		return 0, 0, fmt.Errorf("invalid domain %q", o.Domain)
	}
	hash, err := auth.HashPassword(o.Password)
	if err != nil {
		return 0, 0, err
	}
	rows, err := pool.Query(ctx, `SELECT slug FROM interests ORDER BY slug`)
	if err != nil {
		return 0, 0, err
	}
	var interestSlugs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			return 0, 0, err
		}
		interestSlugs = append(interestSlugs, s)
	}
	rows.Close()
	if len(interestSlugs) == 0 {
		return 0, 0, fmt.Errorf("no interests found; migrations not applied?")
	}

	rng := rand.New(rand.NewSource(20260929))
	for i := 1; i <= o.Count; i++ {
		// Draw every random value up front so a skipped user does not change the others.
		p := people[(i-1)%len(people)]
		age := 19 + rng.Intn(27)
		birth := o.Now.UTC().AddDate(-age, 0, -rng.Intn(360)).Format("2006-01-02")
		lat := round2(o.CenterLat + (rng.Float64()-0.5)*0.16)
		lng := round2(o.CenterLng + (rng.Float64()-0.5)*0.22)
		city := cities[rng.Intn(len(cities))]
		bio := bios[rng.Intn(len(bios))]
		nInterests := 3 + rng.Intn(4)
		picked := rng.Perm(len(interestSlugs))[:nInterests]
		nPhotos := 2 + rng.Intn(2)
		seed := rng.Int63()
		interestedIn := [][]string{
			{"woman", "man", "non_binary"}, {"woman"}, {"man"}, {"woman", "non_binary"}, {"man", "non_binary"},
		}[i%5]
		ageMin := max(18, age-8-rng.Intn(4))
		ageMax := min(99, age+8+rng.Intn(8))
		maxDist := []int{25, 50, 100}[rng.Intn(3)]
		showAge := i%7 != 0
		showDistance := i%9 != 0

		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email(i, o.Domain)).Scan(&exists); err != nil {
			return created, skipped, err
		}
		if exists {
			skipped++
			continue
		}

		var keys []string
		cleanup := func() {
			for _, k := range keys {
				_ = store.Delete(context.Background(), k)
			}
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return created, skipped, err
		}
		var uid string
		err = tx.QueryRow(ctx, `INSERT INTO users (email, password_hash, birth_date) VALUES ($1,$2,$3) RETURNING id`,
			email(i, o.Domain), hash, birth).Scan(&uid)
		if err == nil {
			_, err = tx.Exec(ctx, `
				INSERT INTO profiles (user_id, first_name, gender, bio, city, latitude, longitude, location_updated_at, show_age, show_distance)
				VALUES ($1,$2,$3,$4,$5,$6,$7,NOW(),$8,$9)`, uid, p.name, p.gender, bio, city, lat, lng, showAge, showDistance)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km) VALUES ($1,$2,$3,$4,$5)`,
				uid, interestedIn, int16(ageMin), int16(ageMax), maxDist)
		}
		if err == nil {
			chosen := make([]string, len(picked))
			for k, idx := range picked {
				chosen[k] = interestSlugs[idx]
			}
			_, err = tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug = ANY($2)`, uid, chosen)
		}
		for pos := 0; err == nil && pos < nPhotos; pos++ {
			data := portrait(p.name, seed+int64(pos)*7919, pos)
			key := fmt.Sprintf("photos/%s/seed-%d.jpg", uid, pos)
			if _, perr := store.Put(ctx, key, bytes.NewReader(data)); perr != nil {
				err = perr
				break
			}
			keys = append(keys, key)
			_, err = tx.Exec(ctx, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1,$2,$3,$4,$5,$6)`,
				uid, key, pos, portraitW, portraitH, len(data))
		}
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		if err != nil {
			cleanup()
			return created, skipped, fmt.Errorf("seed user %d: %w", i, err)
		}
		created++
	}
	return created, skipped, nil
}

// Purge deletes exactly the users seeded under domain (demo+<n>@domain) and their photo files.
func Purge(ctx context.Context, pool *pgxpool.Pool, store storage.Storage, domain string) (int, error) {
	if !emailRe.MatchString(domain) {
		return 0, fmt.Errorf("invalid domain %q", domain)
	}
	pattern := `^demo\+[0-9]+@` + strings.ReplaceAll(domain, ".", `\.`) + `$`
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT p.storage_key FROM photos p JOIN users u ON u.id = p.user_id WHERE u.email ~ $1`, pattern)
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
	if err := rows.Err(); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE email ~ $1`, pattern)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	for _, k := range keys {
		_ = store.Delete(ctx, k)
	}
	return int(tag.RowsAffected()), nil
}

func round2(v float64) float64 {
	return float64(int64(v*100+sign(v)*0.5)) / 100
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

const (
	portraitW = 640
	portraitH = 800
)

// portrait draws a synthetic placeholder: a two-colour gradient with the person's initial.
// No external assets are involved.
func portrait(name string, seed int64, variant int) []byte {
	r := rand.New(rand.NewSource(seed))
	c1 := color.RGBA{uint8(60 + r.Intn(160)), uint8(60 + r.Intn(160)), uint8(60 + r.Intn(160)), 255}
	c2 := color.RGBA{uint8(60 + r.Intn(160)), uint8(60 + r.Intn(160)), uint8(60 + r.Intn(160)), 255}
	img := image.NewRGBA(image.Rect(0, 0, portraitW, portraitH))
	for y := 0; y < portraitH; y++ {
		t := float64(y) / float64(portraitH-1)
		row := color.RGBA{
			uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
			uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
			uint8(float64(c1.B)*(1-t) + float64(c2.B)*t), 255,
		}
		for x := 0; x < portraitW; x++ {
			img.SetRGBA(x, y, row)
		}
	}
	// A soft circle differs per variant so a person's photos are distinguishable.
	cx, cy, rad := portraitW/2+variant*40-40, portraitH/3, 90+variant*25
	for y := cy - rad; y <= cy+rad; y++ {
		for x := cx - rad; x <= cx+rad; x++ {
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= rad*rad && x >= 0 && y >= 0 && x < portraitW && y < portraitH {
				img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}

	initial := strings.ToUpper(string([]rune(name)[:1]))
	small := image.NewRGBA(image.Rect(0, 0, 16, 16))
	d := font.Drawer{Dst: small, Src: image.NewUniform(color.RGBA{40, 40, 60, 255}), Face: basicfont.Face7x13,
		Dot: fixed.P(4, 12)}
	d.DrawString(initial)
	dst := image.Rect(portraitW/2-160, portraitH-360, portraitW/2+160, portraitH-40)
	xdraw.NearestNeighbor.Scale(img, dst, small, small.Bounds(), xdraw.Over, nil)

	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}
