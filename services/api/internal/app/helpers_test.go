package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-0123456789abcdef0123456789"

// captureMailer records emails so tests can read reset codes.
type captureMailer struct {
	mu   sync.Mutex
	sent map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sent == nil {
		m.sent = map[string]string{}
	}
	m.sent[to] = body
	return nil
}

var codeRe = regexp.MustCompile(`\b(\d{6})\b`)

func (m *captureMailer) code(t *testing.T, email string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	match := codeRe.FindStringSubmatch(m.sent[email])
	if match == nil {
		t.Fatalf("no reset code sent to %s", email)
	}
	return match[1]
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	mailer *captureMailer
	hub    *realtime.Hub
	store  *storage.LocalStore
	// isolated geographic area so parallel/previous runs never interfere
	lat, lon float64
}

var (
	poolOnce sync.Once
	pool     *pgxpool.Pool
	poolErr  error
)

func sharedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("integration tests need DATABASE_URL_TEST (or DATABASE_URL)")
	}
	poolOnce.Do(func() {
		pool, poolErr = db.Connect(context.Background(), url)
		if poolErr == nil {
			poolErr = db.RunMigrations(context.Background(), pool)
		}
	})
	if poolErr != nil {
		t.Fatalf("database: %v", poolErr)
	}
	return pool
}

func newEnv(t *testing.T, limits Limits) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	p := sharedPool(t)
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, pool: p, mailer: &captureMailer{}, hub: realtime.NewHub(), store: store,
		lat: -50 + rand.Float64()*100, lon: -170 + rand.Float64()*340}
	e.router, err = NewRouter(Options{
		Pool: p, JWTSecret: []byte(testSecret), Store: store, Mailer: e.mailer, Hub: e.hub, Limits: limits,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func generousLimits() Limits {
	l := DefaultLimits()
	big := [2]int{100000, 100000}
	l.GlobalPerIP, l.Credentials, l.Sensitive, l.Uploads, l.Messages, l.Swipes, l.Reports, l.RealtimeAuth = big, big, big, big, big, big, big, big
	return l
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *env) expect(r resp, status int, context string) {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("%s: expected %d, got %d: %s", context, status, r.Status, r.Body)
	}
}

func (e *env) upload(method, path, token string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("photo", "photo.png")
	part.Write(data)
	mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

type authPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type user struct {
	ID, Email, Token, Refresh string
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d_%d@integration.test", prefix, time.Now().UnixNano(), rand.IntN(1_000_000))
}

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := uniqueEmail(prefix)
	r := e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"})
	e.expect(r, http.StatusCreated, "register")
	var a authPayload
	r.json(e.t, &a)
	e.t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})
	return user{ID: a.User.ID, Email: email, Token: a.AccessToken, Refresh: a.RefreshToken}
}

func testPNG(t *testing.T, seed uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 320, 400))
	for x := 0; x < 320; x++ {
		for y := 0; y < 400; y++ {
			img.Set(x, y, color.RGBA{uint8(x) + seed, uint8(y), seed, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// onboard creates a complete, discoverable profile.
func (e *env) onboard(u user, name, gender string, interestedIn []string, birthdate string) {
	e.t.Helper()
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{
		"firstName": name, "birthdate": birthdate, "gender": gender, "bio": "Bio de " + name,
		"interestIds": []int{1, 10, 21},
	}), http.StatusOK, "profile upsert "+name)
	e.expect(e.do("PUT", "/profile/preferences", u.Token, map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": 10,
	}), http.StatusNoContent, "preferences "+name)
	e.expect(e.do("PUT", "/profile/location", u.Token, map[string]any{
		"latitude": e.lat + rand.Float64()*0.01, "longitude": e.lon + rand.Float64()*0.01, "city": "Testville",
	}), http.StatusNoContent, "location "+name)
	e.expect(e.upload("POST", "/profile/photos", u.Token, testPNG(e.t, uint8(len(name)))), http.StatusCreated, "photo "+name)
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	Interests []struct {
		ID int `json:"id"`
	} `json:"interests"`
}

func (e *env) deck(u user) []card {
	e.t.Helper()
	r := e.do("GET", "/discovery?limit=30", u.Token, nil)
	e.expect(r, http.StatusOK, "deck")
	var out struct {
		Profiles []card `json:"profiles"`
	}
	r.json(e.t, &out)
	return out.Profiles
}

func containsCard(cards []card, id string) bool {
	for _, c := range cards {
		if c.UserID == id {
			return true
		}
	}
	return false
}

func (e *env) swipe(u user, target, action string) resp {
	e.t.Helper()
	return e.do("POST", "/discovery/swipes", u.Token, map[string]string{"targetId": target, "action": action})
}

func birthdateYearsAgo(years int) string {
	return time.Now().AddDate(-years, 0, -10).Format("2006-01-02")
}

func trimQuery(u string) string { return strings.SplitN(u, "?", 2)[0] }
