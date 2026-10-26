package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-secret-0123456789abcdef-xyz"

// captureMailer keeps sent emails so tests can read password reset codes.
type captureMailer struct {
	mu   sync.Mutex
	sent []struct{ To, Body string }
}

func (m *captureMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, struct{ To, Body string }{to, body})
	return nil
}

func (m *captureMailer) last() (string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return "", ""
	}
	l := m.sent[len(m.sent)-1]
	return l.To, l.Body
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	srv    *httptest.Server
	mail   *captureMailer
	hub    *realtime.Hub
	limits limits
}

// newEnv boots the whole API against PostgreSQL in a throw-away schema so
// tests are isolated and leave no data behind.
func newEnv(t *testing.T, mutate ...func(*limits)) *env {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	schema := fmt.Sprintf("t_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	l := limits{
		credentials: middleware.NewLimiter(100000, time.Minute),
		upload:      middleware.NewLimiter(100000, time.Minute),
		swipe:       middleware.NewLimiter(100000, time.Minute),
		message:     middleware.NewLimiter(100000, time.Minute),
		report:      middleware.NewLimiter(100000, time.Minute),
	}
	for _, m := range mutate {
		m(&l)
	}
	e := &env{t: t, pool: pool, mail: &captureMailer{}, hub: realtime.NewHub(), limits: l}
	a := &app{dbPool: pool, jwtSecret: []byte(testSecret), mailer: e.mail, hub: e.hub, limits: l}
	gin.SetMode(gin.TestMode)
	r, err := setupRouter(a)
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewServer(r)

	t.Cleanup(func() {
		e.srv.Close()
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	return e
}

var _ mailer.Mailer = (*captureMailer)(nil)

type resp struct {
	Status int
	Raw    []byte
	JSON   map[string]any
}

func (r resp) str(path ...string) string {
	var cur any = r.JSON
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[p]
	}
	s, _ := cur.(string)
	return s
}

func (r resp) list(key string) []map[string]any {
	arr, _ := r.JSON[key].([]any)
	out := make([]map[string]any, 0, len(arr))
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.send(req)
}

func (e *env) send(req *http.Request) resp {
	e.t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := resp{Status: res.StatusCode, Raw: raw}
	_ = json.Unmarshal(raw, &out.JSON)
	return out
}

func (e *env) expect(r resp, status int, what string) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("%s: expected status %d, got %d (%s)", what, status, r.Status, string(r.Raw))
	}
	return r
}

func (e *env) upload(method, path, token string, file []byte, filename string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "image/jpeg")
	part, _ := w.CreatePart(h)
	_, _ = part.Write(file)
	_ = w.Close()
	req, _ := http.NewRequest(method, e.srv.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.send(req)
}

func testJPEG(w, h int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return b.Bytes()
}

type user struct {
	ID, Email, Token, Refresh string
}

type userSpec struct {
	Name         string
	Gender       string
	InterestedIn []string
	Lat, Lng     float64
	AgeYears     int
	MaxDistance  int
	Interests    []string
	NoPhoto      bool
}

const testPassword = "Password123"

var emailSeq int
var emailMu sync.Mutex

func uniqueEmail(prefix string) string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailSeq++
	return fmt.Sprintf("%s-%d@test.example", prefix, emailSeq)
}

func (e *env) register(email string) user {
	e.t.Helper()
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": testPassword}), 201, "register")
	return user{ID: r.str("user", "id"), Email: email, Token: r.str("accessToken"), Refresh: r.str("refreshToken")}
}

// newUser registers a user and completes onboarding (profile, preferences, location, photo).
func (e *env) newUser(s userSpec) user {
	e.t.Helper()
	u := e.register(uniqueEmail(strings.ToLower(s.Name)))
	birth := time.Now().AddDate(-s.AgeYears, 0, -1).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": s.Name, "birthDate": birth, "gender": s.Gender, "bio": "Bio de " + s.Name, "interests": s.Interests,
	}), 200, "save profile")
	dist := s.MaxDistance
	if dist == 0 {
		dist = 50
	}
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{
		"interestedIn": s.InterestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": dist,
	}), 200, "save preferences")
	e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{
		"latitude": s.Lat, "longitude": s.Lng, "city": "Paris",
	}), 200, "save location")
	if !s.NoPhoto {
		e.expect(e.upload("POST", "/me/photos", u.Token, testJPEG(400, 500, color.RGBA{200, 80, 90, 255}), "a.jpg"), 201, "upload photo")
	}
	return u
}

func (e *env) discoverIDs(u user) []string {
	e.t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", u.Token, nil), 200, "discover")
	ids := []string{}
	for _, p := range r.list("profiles") {
		ids = append(ids, p["userId"].(string))
	}
	return ids
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		e.t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// Paris-area coordinates used across tests.
const (
	parisLat, parisLng = 48.8566, 2.3522
	nearLat, nearLng   = 48.8700, 2.3700 // ~2 km
	lyonLat, lyonLng   = 45.7640, 4.8357 // ~390 km
)
