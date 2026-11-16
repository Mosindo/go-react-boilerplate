package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "Password123"

// captureMailer records sent emails so tests can read reset codes.
type captureMailer struct {
	mu   sync.Mutex
	sent map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent[to] = body
	return nil
}

var codePattern = regexp.MustCompile(`\b\d{6}\b`)

func (m *captureMailer) code(t *testing.T, to string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	code := codePattern.FindString(m.sent[to])
	if code == "" {
		t.Fatalf("no reset code sent to %s", to)
	}
	return code
}

type testEnv struct {
	t       *testing.T
	pool    *pgxpool.Pool
	router  *gin.Engine
	mailer  *captureMailer
	hub     *realtime.Hub
	storage *storage.Local
	dir     string
	emails  []string
}

var emailSeq atomic.Int64

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("integration tests need DATABASE_URL_TEST (or DATABASE_URL)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	dir := t.TempDir()
	store, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	env := &testEnv{t: t, pool: pool, mailer: &captureMailer{sent: map[string]string{}}, hub: realtime.NewHub(), storage: store, dir: dir}
	cfg := config.Config{Env: "test", JWTSecret: "integration-secret-0123456789abcdefghij"}
	relaxed := ratelimit.New(10_000, time.Second, 10_000)
	env.router = buildRouterWithLimits(cfg, Deps{Pool: pool, Storage: store, Mailer: env.mailer, Hub: env.hub}, RateLimits{
		Auth: relaxed, Swipes: relaxed, Messages: relaxed, Uploads: relaxed, Reports: relaxed,
	})
	t.Cleanup(func() {
		if len(env.emails) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, env.emails)
		}
		pool.Close()
	})
	return env
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *testEnv) do(method, path, token string, body any) response {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(b)
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
	return response{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *testEnv) expect(r response, status int) response {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func (e *testEnv) upload(method, path, token string, data []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", "photo.png")
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return response{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

type authPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		HasProfile bool   `json:"hasProfile"`
	} `json:"user"`
}

type testUser struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

func (e *testEnv) uniqueEmail(prefix string) string {
	email := fmt.Sprintf("%s.%d.%d@it.test", prefix, time.Now().UnixNano(), emailSeq.Add(1))
	e.emails = append(e.emails, email)
	return email
}

func (e *testEnv) register(prefix string) testUser {
	e.t.Helper()
	email := e.uniqueEmail(prefix)
	var p authPayload
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": testPassword}), 201).decode(e.t, &p)
	return testUser{ID: p.User.ID, Email: email, Token: p.AccessToken, Refresh: p.RefreshToken}
}

type profileSpec struct {
	Name         string
	Birthdate    string
	Gender       string
	InterestedIn []string
	AgeMin       int
	AgeMax       int
	Distance     int
	Lat, Lng     *float64
	Interests    []int
}

func ptr(f float64) *float64 { return &f }

func testPNG(w, h int, shade uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: shade, G: uint8(x), B: uint8(y), A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// onboard creates a complete, discoverable user.
func (e *testEnv) onboard(prefix string, spec profileSpec) testUser {
	e.t.Helper()
	u := e.register(prefix)
	if spec.Birthdate == "" {
		spec.Birthdate = "1994-05-10"
	}
	if spec.AgeMin == 0 {
		spec.AgeMin, spec.AgeMax = 18, 99
	}
	if spec.Distance == 0 {
		spec.Distance = 100
	}
	e.expect(e.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": spec.Name, "birthdate": spec.Birthdate, "gender": spec.Gender,
		"bio": "Bio de " + spec.Name, "interestIds": spec.Interests, "lookingFor": "relationship",
	}), 200)
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{
		"interestedIn": spec.InterestedIn, "ageMin": spec.AgeMin, "ageMax": spec.AgeMax, "maxDistanceKm": spec.Distance,
	}), 200)
	if spec.Lat != nil {
		e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": *spec.Lat, "longitude": *spec.Lng}), 204)
	}
	e.expect(e.upload("POST", "/me/photos", u.Token, testPNG(320, 400, 120)), 201)
	return u
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Interests  []struct {
		ID     int  `json:"id"`
		Shared bool `json:"shared"`
	} `json:"interests"`
	Photos []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
}

type deck struct {
	Profiles []card `json:"profiles"`
	Reason   string `json:"reason"`
}

func (e *testEnv) deck(u testUser) deck {
	e.t.Helper()
	var d deck
	e.expect(e.do("GET", "/discover?limit=30", u.Token, nil), 200).decode(e.t, &d)
	return d
}

func (d deck) has(userID string) bool {
	for _, c := range d.Profiles {
		if c.UserID == userID {
			return true
		}
	}
	return false
}

type swipeResult struct {
	Action  string `json:"action"`
	Matched bool   `json:"matched"`
	Match   *struct {
		ID   string `json:"id"`
		User card   `json:"user"`
	} `json:"match"`
}

func (e *testEnv) swipe(u testUser, target, action string) swipeResult {
	e.t.Helper()
	var s swipeResult
	e.expect(e.do("POST", "/swipes", u.Token, map[string]string{"targetUserId": target, "action": action}), 200).decode(e.t, &s)
	return s
}

func (e *testEnv) matchUsers(a, b testUser) string {
	e.t.Helper()
	e.swipe(a, b.ID, "like")
	s := e.swipe(b, a.ID, "like")
	if !s.Matched || s.Match == nil {
		e.t.Fatalf("expected match between %s and %s", a.Email, b.Email)
	}
	return s.Match.ID
}

func (e *testEnv) countRows(query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func contains(haystack []byte, needle string) bool { return strings.Contains(string(haystack), needle) }

func testConfig() config.Config {
	return config.Config{Env: "test", JWTSecret: "integration-secret-0123456789abcdefghij"}
}
