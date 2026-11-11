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
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-0123456789abcdef0123456789"

// captureMailer records outgoing e-mails so tests can read reset codes.
type captureMailer struct {
	mu   sync.Mutex
	last map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.last == nil {
		m.last = map[string]string{}
	}
	m.last[to] = body
	return nil
}

var codePattern = regexp.MustCompile(`\b(\d{6})\b`)

func (m *captureMailer) code(t *testing.T, to string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	match := codePattern.FindStringSubmatch(m.last[to])
	if match == nil {
		t.Fatalf("no reset code sent to %s", to)
	}
	return match[1]
}

type testEnv struct {
	t      *testing.T
	pool   *pgxpool.Pool
	server *httptest.Server
	mail   *captureMailer
	store  *storage.Local
	dir    string
}

func setupEnv(t *testing.T, rateLimited bool) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("set DATABASE_URL_TEST to run API integration tests")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("migrations: %v", err)
	}
	dir := t.TempDir()
	store, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	cfg := config.Config{Env: "test", JWTSecret: testSecret, RateLimitDisabled: !rateLimited}
	router, err := newRouter(cfg, pool, store, mail)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(func() {
		srv.Close()
		pool.Close()
	})
	return &testEnv{t: t, pool: pool, server: srv, mail: mail, store: store, dir: dir}
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *testEnv) do(method, path, token string, body any) response {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, e.server.URL+path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.send(req)
}

func (e *testEnv) send(req *http.Request) response {
	e.t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return response{Status: res.StatusCode, Body: data, Header: res.Header}
}

func (e *testEnv) expect(r response, status int, context string) {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("%s: expected %d, got %d: %s", context, status, r.Status, r.Body)
	}
}

func (e *testEnv) upload(path, method, token string, data []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", "photo.png")
	part.Write(data)
	w.Close()
	req, _ := http.NewRequest(method, e.server.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.send(req)
}

type member struct {
	ID       string
	Email    string
	Password string
	Access   string
	Refresh  string
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d_%d@example-test.dev", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
}

func (e *testEnv) register(prefix string) member {
	e.t.Helper()
	m := member{Email: uniqueEmail(prefix), Password: "Password123"}
	r := e.do("POST", "/auth/register", "", map[string]string{"email": m.Email, "password": m.Password})
	e.expect(r, http.StatusCreated, "register "+prefix)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.decode(e.t, &out)
	m.ID, m.Access, m.Refresh = out.User.ID, out.AccessToken, out.RefreshToken
	e.t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, m.ID)
	})
	return m
}

func pngImage(t *testing.T, w, h int, tint uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y += 4 {
		for x := 0; x < w; x += 4 {
			img.Set(x, y, color.RGBA{tint, uint8(x), uint8(y), 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type onboarding struct {
	firstName, birthdate, gender string
	interestedIn                 []string
	lat, lng                     float64
	interests                    []string
}

// onboard creates a complete profile: info, preferences, interests, location, photo.
func (e *testEnv) onboard(m member, o onboarding) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", m.Access, map[string]any{
		"firstName": o.firstName, "birthdate": o.birthdate, "gender": o.gender,
		"bio": "Bio de " + o.firstName, "jobTitle": "Testeur", "city": "Testville",
	}), http.StatusOK, "profile "+o.firstName)
	e.expect(e.do("PUT", "/me/preferences", m.Access, map[string]any{
		"interestedIn": o.interestedIn, "ageMin": 18, "ageMax": 60, "maxDistanceKm": 25,
	}), http.StatusOK, "preferences "+o.firstName)
	e.expect(e.do("PUT", "/me/interests", m.Access, map[string]any{"interests": o.interests}), http.StatusOK, "interests")
	e.expect(e.do("PUT", "/me/location", m.Access, map[string]any{"latitude": o.lat, "longitude": o.lng}), http.StatusOK, "location")
	e.expect(e.upload("/me/photos", "POST", m.Access, pngImage(e.t, 600, 800, 120)), http.StatusCreated, "photo "+o.firstName)
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Interests  []struct {
		Slug   string `json:"slug"`
		Shared bool   `json:"shared"`
	} `json:"interests"`
	Photos []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
}

func (e *testEnv) discover(m member) []card {
	e.t.Helper()
	r := e.do("GET", "/discover?limit=25", m.Access, nil)
	e.expect(r, http.StatusOK, "discover")
	var out struct {
		Profiles []card `json:"profiles"`
	}
	r.decode(e.t, &out)
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

// randomArea returns coordinates in an isolated random area so concurrent or
// previous test runs never show up in each other's discovery results.
func randomArea() (float64, float64) {
	return -60 + rand.Float64()*120, -170 + rand.Float64()*340
}

func mustNotContain(t *testing.T, body []byte, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if strings.Contains(string(body), n) {
			t.Fatalf("response leaks %q: %s", n, body)
		}
	}
}
