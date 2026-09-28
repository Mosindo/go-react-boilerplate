// Command seed fills a development database with demo data: ~24 demo users
// (emails @demo.invalid, one shared password), synthetic generated portraits,
// interests, locations around a configurable city, and a few swipes, matches
// and messages. It is idempotent and refuses to run when APP_ENV=production.
//
// See docs/SEED.md.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DemoPassword is shared by every demo account (documented in docs/SEED.md).
	DemoPassword = "DemoPass123!"
	demoDomain   = "demo.invalid"
	userCount    = 24
)

type demoUser struct {
	Email     string
	Name      string
	Gender    string
	Age       int
	Bio       string
	Interests []string
}

var names = map[string][]string{
	"woman":      {"Alma", "Bea", "Chloe", "Dara", "Elin", "Fay", "Gia", "Hana", "Iris", "Jade", "Kira", "Lena"},
	"man":        {"Adam", "Ben", "Cal", "Dan", "Eli", "Finn", "Gus", "Hugo", "Ivan", "Jon", "Kai", "Leo"},
	"non_binary": {"Ash", "Blair", "Charlie", "Devon", "Emery", "Frankie", "Harper", "Indigo", "Jules", "Kit", "Lux", "Morgan"},
}

var bios = []string{
	"Weekend hiker and amateur cook.",
	"Always up for a good coffee and a better conversation.",
	"Dog person. Bad at karaoke, great at trying.",
	"Museums by day, board games by night.",
	"Looking for someone to explore new places with.",
	"Bookworm with a soft spot for live music.",
	"Runner, reader, occasional baker.",
	"Curious about everything, especially food.",
}

func main() {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		log.Fatal("seed: refusing to run with APP_ENV=production")
	}
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		log.Fatal("seed: DATABASE_URL is required")
	}
	lat := envFloat("SEED_CITY_LAT", 48.8566)
	lon := envFloat("SEED_CITY_LON", 2.3522)
	label := envString("SEED_CITY_LABEL", "Demo City")
	radiusKm := envFloat("SEED_RADIUS_KM", 15)

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, dbURL, 15*time.Second, time.Second)
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("seed: migrations: %v", err)
	}

	hash, err := auth.HashPassword(DemoPassword, 0)
	if err != nil {
		log.Fatal(err)
	}
	if err := seed(ctx, pool, hash, lat, lon, label, radiusKm); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Printf("seed: done. Log in as demo01@%s .. demo%02d@%s with password %q", demoDomain, userCount, demoDomain, DemoPassword)
}

func seed(ctx context.Context, pool *pgxpool.Pool, hash string, lat, lon float64, label string, radiusKm float64) error {
	users := buildUsers()
	ids := make([]string, len(users))
	created := 0
	for i, u := range users {
		id, isNew, err := seedUser(ctx, pool, hash, u, i, lat, lon, label, radiusKm)
		if err != nil {
			return fmt.Errorf("user %s: %w", u.Email, err)
		}
		ids[i] = id
		if isNew {
			created++
		}
	}
	log.Printf("seed: %d users (%d new)", len(users), created)
	return seedRelations(ctx, pool, ids, users)
}

func buildUsers() []demoUser {
	genders := []string{"woman", "man", "non_binary"}
	var out []demoUser
	for i := 0; i < userCount; i++ {
		// 10 women, 10 men, 4 non-binary, interleaved.
		g := genders[i%2]
		if i%6 == 5 {
			g = "non_binary"
		}
		pool := names[g]
		name := pool[(i/2)%len(pool)]
		out = append(out, demoUser{
			Email:  fmt.Sprintf("demo%02d@%s", i+1, demoDomain),
			Name:   name,
			Gender: g,
			Age:    22 + (i*7)%23,
			Bio:    bios[i%len(bios)],
		})
	}
	return out
}

func seedUser(ctx context.Context, pool *pgxpool.Pool, hash string, u demoUser, idx int, cityLat, cityLon float64, label string, radiusKm float64) (string, bool, error) {
	rng := rand.New(rand.NewSource(int64(idx) + 1))
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	isNew := false
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, u.Email).Scan(&id)
	if err == pgx.ErrNoRows {
		isNew = true
		if err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`, u.Email, hash).Scan(&id); err != nil {
			return "", false, err
		}
	} else if err != nil {
		return "", false, err
	}

	// Locations: random point inside radiusKm, rounded to 2 decimals like the API does.
	angle := rng.Float64() * 2 * math.Pi
	dist := math.Sqrt(rng.Float64()) * radiusKm
	pLat := cityLat + dist/111.0*math.Cos(angle)
	pLon := cityLon + dist/(111.0*math.Cos(cityLat*math.Pi/180))*math.Sin(angle)
	pLat, pLon = math.Round(pLat*100)/100, math.Round(pLon*100)/100
	birth := time.Now().UTC().AddDate(-u.Age, 0, -(idx*13)%300)

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, location_label, latitude, longitude, last_active_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW() - ($9::text || ' minutes')::interval)
		ON CONFLICT (user_id) DO NOTHING`,
		id, u.Name, birth, u.Gender, u.Bio, label, pLat, pLon, strconv.Itoa(idx*37)); err != nil {
		return "", false, err
	}

	interested := []string{"man", "woman", "non_binary"}
	switch idx % 5 {
	case 0:
		interested = []string{"woman", "non_binary"}
	case 1:
		interested = []string{"man", "non_binary"}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, 18, 60, 50) ON CONFLICT (user_id) DO NOTHING`, id, interested); err != nil {
		return "", false, err
	}

	// 3..6 interests, chosen deterministically.
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, i.id FROM (SELECT id FROM interests ORDER BY md5(slug || $2::text) LIMIT $3) i
		ON CONFLICT DO NOTHING`, id, strconv.Itoa(idx), 3+idx%4); err != nil {
		return "", false, err
	}

	// 1..3 generated portraits (only when the user has none yet).
	var have int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, id).Scan(&have); err != nil {
		return "", false, err
	}
	if have == 0 {
		for p := 0; p < 1+idx%3; p++ {
			data, w, h, err := portrait(idx*10 + p)
			if err != nil {
				return "", false, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO photos (user_id, position, content, byte_size, width, height)
				VALUES ($1, $2, $3, $4, $5, $6)`, id, p, data, len(data), w, h); err != nil {
				return "", false, err
			}
		}
	}
	return id, isNew, tx.Commit(ctx)
}

// seedRelations creates some swipes, matches and messages between demo users.
func seedRelations(ctx context.Context, pool *pgxpool.Pool, ids []string, users []demoUser) error {
	// 1) one-sided likes towards demo01 (so "likes you" style flows have data)
	for i := 4; i < 10; i++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'like')
			ON CONFLICT DO NOTHING`, ids[i], ids[0]); err != nil {
			return err
		}
	}
	// 2) some passes so the queue is not empty but not identical for everyone
	for i := 10; i < 14; i++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'pass')
			ON CONFLICT DO NOTHING`, ids[0], ids[i]); err != nil {
			return err
		}
	}
	// 3) matches with conversations (demo01 with demo02..demo04, plus two other pairs)
	pairs := [][2]int{{0, 1}, {0, 2}, {0, 3}, {14, 15}, {16, 17}}
	conv := map[[2]int]string{}
	for _, p := range pairs {
		a, b := ids[p[0]], ids[p[1]]
		for _, dir := range [][2]string{{a, b}, {b, a}} {
			if _, err := pool.Exec(ctx, `
				INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'like')
				ON CONFLICT DO NOTHING`, dir[0], dir[1]); err != nil {
				return err
			}
		}
		var matchID string
		err := pool.QueryRow(ctx, `
			INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid))
			ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = matches.user_a
			RETURNING id`, a, b).Scan(&matchID)
		if err != nil {
			return err
		}
		var convID string
		err = pool.QueryRow(ctx, `
			INSERT INTO match_conversations (match_id) VALUES ($1)
			ON CONFLICT (match_id) DO UPDATE SET match_id = match_conversations.match_id
			RETURNING id`, matchID).Scan(&convID)
		if err != nil {
			return err
		}
		conv[p] = convID
	}
	// 4) messages (only when the conversation is still empty, keeps re-runs idempotent)
	script := map[[2]int][]string{
		{0, 1}:   {"Hey! Loved your profile.", "Thanks! Yours too. Coffee this week?", "Sounds great, Thursday?"},
		{0, 2}:   {"Hi there, how was your weekend?"},
		{14, 15}: {"Hello!", "Hi! How are you?"},
	}
	for p, lines := range script {
		convID := conv[p]
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM match_messages WHERE conversation_id = $1`, convID).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		base := time.Now().Add(-time.Duration(len(lines)) * time.Hour)
		for i, body := range lines {
			sender := ids[p[i%2]]
			if _, err := pool.Exec(ctx, `
				INSERT INTO match_messages (conversation_id, sender_id, body, created_at)
				VALUES ($1, $2, $3, $4)`, convID, sender, body, base.Add(time.Duration(i)*time.Hour)); err != nil {
				return err
			}
		}
		if _, err := pool.Exec(ctx, `
			UPDATE match_conversations SET last_message_at = (SELECT max(created_at) FROM match_messages WHERE conversation_id = $1)
			WHERE id = $1`, convID); err != nil {
			return err
		}
	}
	log.Printf("seed: %d matches, swipes and messages ensured", len(pairs))
	return nil
}

// portrait renders a synthetic 480x600 JPEG: a diagonal two-colour gradient
// with a soft radial highlight and a stylised head-and-shoulders silhouette.
// No text, no real people.
func portrait(seed int) ([]byte, int, int, error) {
	const w, h = 480, 600
	rng := rand.New(rand.NewSource(int64(seed)*7919 + 13))
	hue := rng.Float64() * 360
	c1 := hsv(hue, 0.55, 0.95)
	c2 := hsv(math.Mod(hue+40+rng.Float64()*60, 360), 0.65, 0.55)
	skin := hsv(20+rng.Float64()*15, 0.25+rng.Float64()*0.3, 0.55+rng.Float64()*0.4)
	body := hsv(math.Mod(hue+180, 360), 0.35, 0.35+rng.Float64()*0.3)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)/2+rng.Float64()*40-20, float64(h)*0.40
	headR := 95 + rng.Float64()*15
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := (float64(x)/w + float64(y)/h) / 2
			c := mix(c1, c2, t)
			// radial highlight
			dx, dy := float64(x)-w*0.3, float64(y)-h*0.25
			c = mix(c, color.RGBA{255, 255, 255, 255}, 0.25*math.Exp(-(dx*dx+dy*dy)/(2*140*140)))
			// head
			hx, hy := float64(x)-cx, (float64(y)-cy)*0.92
			if hx*hx+hy*hy < headR*headR {
				c = skin
			}
			// shoulders: a wide ellipse below the head
			sx, sy := float64(x)-cx, float64(y)-(cy+headR*2.35)
			if sx*sx/(190*190)+sy*sy/(150*150) < 1 && float64(y) > cy+headR*0.9 {
				c = body
			}
			// neck
			if math.Abs(float64(x)-cx) < 32 && float64(y) > cy+headR*0.7 && float64(y) < cy+headR*1.5 {
				c = skin
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	f := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

func hsv(h, s, v float64) color.RGBA {
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

func envString(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		log.Fatalf("seed: %s must be a number", key)
	}
	return def
}
