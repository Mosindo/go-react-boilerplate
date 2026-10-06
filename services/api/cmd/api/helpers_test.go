package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "integration-tests-jwt-secret-0123456789"

var ipCounter atomic.Int64

// captureMailer records outgoing email so tests can read password-reset codes.
type captureMailer struct {
	mu   sync.Mutex
	last map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.last == nil {
		m.last = map[string]string{}
	}
	m.last[to] = body
	return nil
}

func (m *captureMailer) codeFor(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	match := regexp.MustCompile(`: ([A-Z2-9]{8})`).FindStringSubmatch(m.last[email])
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	blobs  *storage.Disk
	dir    string
	mail   *captureMailer
}

// newEnv boots the full router against a real PostgreSQL database
// (TEST_DATABASE_URL, default: local app_test) and wipes all data first.
// The tests skip when no database is reachable, unless REQUIRE_DB=1 (CI).
func newEnv(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		if os.Getenv("REQUIRE_DB") == "1" {
			t.Fatalf("database required but unavailable: %v", err)
		}
		t.Skipf("database unavailable (%v); set TEST_DATABASE_URL or REQUIRE_DB=1", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, conversations CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	dir := t.TempDir()
	blobs, err := storage.NewDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	gin.SetMode(gin.TestMode)
	router := setupRouter(&app{
		dbPool:         pool,
		jwtSecret:      []byte(testJWTSecret),
		trustedProxies: []string{"0.0.0.0/0", "::/0"}, // lets tests simulate distinct client IPs
		blobs:          blobs,
		mailer:         mail,
	})
	return &env{t: t, router: router, pool: pool, blobs: blobs, dir: dir, mail: mail}
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %q: %v", string(r.Body), err)
	}
}

func (r response) json(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	r.decode(t, &m)
	return m
}

func (e *env) raw(method, path, token, ip string, body []byte, contentType string) response {
	e.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if ip == "" {
		ip = fmt.Sprintf("10.9.%d.%d", ipCounter.Add(1)/250, ipCounter.Load()%250+1)
	}
	req.Header.Set("X-Forwarded-For", ip)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *env) do(method, path, token string, body any) response {
	e.t.Helper()
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			e.t.Fatal(err)
		}
	}
	return e.raw(method, path, token, "", payload, "application/json")
}

func (e *env) expect(r response, status int) response {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, string(r.Body))
	}
	return r
}

func (e *env) upload(token, path string, filename string, data []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	h.Set("Content-Type", "image/jpeg")
	part, err := w.CreatePart(h)
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	return e.raw(http.MethodPost, path, token, "", buf.Bytes(), w.FormDataContentType())
}

func testJPEG(t *testing.T, seed int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 400, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{uint8((x + seed*40) % 256), uint8(y % 256), uint8(seed * 30 % 256), 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// --- user fixtures ---

type user struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

var emailSeq atomic.Int64

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s-%d@test.local", strings.ToLower(prefix), emailSeq.Add(1))
	r := e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": "Password123!"}), http.StatusCreated)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.decode(e.t, &out)
	return user{ID: out.User.ID, Email: email, Token: out.AccessToken, Refresh: out.RefreshToken}
}

type persona struct {
	Name         string
	Gender       string
	InterestedIn []string
	Age          int
	MinAge       int
	MaxAge       int
	MaxDistance  int
	Lat, Lng     float64
	Photos       int
	Bio          string
}

const (
	parisLat, parisLng = 48.8566, 2.3522
	lyonLat, lyonLng   = 45.7640, 4.8357
)

func birthdateFor(age int) string {
	return time.Now().AddDate(-age, 0, -3).Format("2006-01-02")
}

// onboard drives the real API to complete a profile.
func (e *env) onboard(u user, p persona) {
	e.t.Helper()
	if p.Age == 0 {
		p.Age = 28
	}
	if p.MinAge == 0 {
		p.MinAge = 18
	}
	if p.MaxAge == 0 {
		p.MaxAge = 60
	}
	if p.MaxDistance == 0 {
		p.MaxDistance = 50
	}
	if p.Lat == 0 {
		p.Lat, p.Lng = parisLat, parisLng
	}
	e.expect(e.do(http.MethodPut, "/profile", u.Token, map[string]any{
		"firstName": p.Name, "birthdate": birthdateFor(p.Age), "gender": p.Gender, "bio": p.Bio,
	}), http.StatusOK)
	e.expect(e.do(http.MethodPut, "/profile/location", u.Token, map[string]any{"latitude": p.Lat, "longitude": p.Lng, "city": "Quelque part"}), http.StatusOK)
	e.expect(e.do(http.MethodPut, "/preferences", u.Token, map[string]any{
		"interestedIn": p.InterestedIn, "minAge": p.MinAge, "maxAge": p.MaxAge, "maxDistanceKm": p.MaxDistance,
	}), http.StatusOK)
	for i := 0; i < max(p.Photos, 1); i++ {
		e.expect(e.upload(u.Token, "/photos", "p.jpg", testJPEG(e.t, i+1)), http.StatusCreated)
	}
}

func (e *env) newPerson(p persona) user {
	e.t.Helper()
	u := e.register(strings.ToLower(p.Name))
	e.onboard(u, p)
	return u
}

func (e *env) discoverIDs(u user) []string {
	e.t.Helper()
	r := e.expect(e.do(http.MethodGet, "/discover?limit=20", u.Token, nil), http.StatusOK)
	var out struct {
		Profiles []struct {
			UserID string `json:"userId"`
		} `json:"profiles"`
	}
	r.decode(e.t, &out)
	ids := make([]string, 0, len(out.Profiles))
	for _, p := range out.Profiles {
		ids = append(ids, p.UserID)
	}
	return ids
}

func contains(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

type swipeOut struct {
	Action           string `json:"action"`
	AlreadyProcessed bool   `json:"alreadyProcessed"`
	Matched          bool   `json:"matched"`
	MatchID          string `json:"matchId"`
}

func (e *env) swipe(from user, to user, action string, wantStatus int) swipeOut {
	e.t.Helper()
	r := e.expect(e.do(http.MethodPost, "/swipes", from.Token, map[string]string{"userId": to.ID, "action": action}), wantStatus)
	var out swipeOut
	if wantStatus < 300 {
		r.decode(e.t, &out)
	}
	return out
}

// makeMatch has both users like each other and returns the match id.
func (e *env) makeMatch(a, b user) string {
	e.t.Helper()
	e.swipe(a, b, "like", http.StatusCreated)
	out := e.swipe(b, a, "like", http.StatusCreated)
	if !out.Matched || out.MatchID == "" {
		e.t.Fatalf("expected a match, got %+v", out)
	}
	return out.MatchID
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMain(m *testing.M) {
	// Request logs are noise in test output; use -v to see them.
	flag.Parse()
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}
	os.Exit(m.Run())
}
