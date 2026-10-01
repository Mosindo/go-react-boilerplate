package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var jwtSecret = []byte("integration-test-secret-0123456789abcdef")

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

// code waits for the recovery email, which is sent in the background by design.
func (m *captureMailer) code(to string) string {
	for i := 0; i < 100; i++ {
		m.mu.Lock()
		body := m.last[to]
		m.mu.Unlock()
		if j := strings.Index(body, ": "); j >= 0 && len(body) >= j+10 {
			return body[j+2 : j+10]
		}
		time.Sleep(20 * time.Millisecond)
	}
	return ""
}

type env struct {
	t       *testing.T
	pool    *pgxpool.Pool
	svc     *app.Services
	mailer  *captureMailer
	uploads string
	seq     atomic.Int64
	ipSeq   atomic.Int64
	fixedIP string
}

// newEnv starts from an empty schema so every run also proves the migrations apply from scratch.
// It only ever touches DATABASE_URL_TEST: never point that at a database you care about.
func newEnv(t *testing.T) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST not set; skipping integration tests")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	// Running twice must be a no-op (schema_migrations tracking).
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations rerun: %v", err)
	}
	uploads := t.TempDir()
	store, err := newStore(uploads)
	if err != nil {
		t.Fatal(err)
	}
	m := &captureMailer{}
	svc, err := app.NewRouter(app.Deps{Pool: pool, JWTSecret: jwtSecret, Store: store, Mailer: m})
	if err != nil {
		t.Fatal(err)
	}
	return &env{t: t, pool: pool, svc: svc, mailer: m, uploads: uploads}
}

// nextIP gives every request its own client address so per-IP rate limits (which are real and
// enabled in tests) only trigger in the tests that exercise them, via fixedIP.
func (e *env) nextIP() string {
	if e.fixedIP != "" {
		return e.fixedIP
	}
	n := e.ipSeq.Add(1)
	return fmt.Sprintf("10.%d.%d.%d:4000", (n>>16)&255, (n>>8)&255, n&255)
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json() map[string]any {
	var m map[string]any
	_ = json.Unmarshal(r.Body, &m)
	return m
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.RemoteAddr = e.nextIP()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.svc.Router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *env) upload(method, path, token string, file []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "photo.bin")
	_, _ = fw.Write(file)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = e.nextIP()
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.svc.Router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *env) want(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, string(r.Body))
	}
	return r
}

func jpegBytes(seed int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 640, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x/3 + seed), G: uint8(y / 4), B: uint8(seed * 7), A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

type user struct {
	ID, Email, Token, Refresh, Password string
}

type spec struct {
	Name         string
	Gender       string
	InterestedIn []string
	Age          int
	Lat, Lng     float64
	MinAge       int
	MaxAge       int
	MaxKm        int
	NoPhoto      bool
	NoLocation   bool
}

const (
	parisLat, parisLng = 48.8566, 2.3522
	nearLat, nearLng   = 48.90, 2.40     // ~6 km
	lyonLat, lyonLng   = 45.7640, 4.8357 // ~390 km
)

// register creates an account only.
func (e *env) register(prefix string) user {
	e.t.Helper()
	n := e.seq.Add(1)
	email := fmt.Sprintf("%s%d@test.invalid", prefix, n)
	pw := "Password123"
	r := e.want(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": pw}), 201).json()
	u := r["user"].(map[string]any)
	return user{ID: u["id"].(string), Email: email, Token: r["accessToken"].(string), Refresh: r["refreshToken"].(string), Password: pw}
}

// newUser registers and fully onboards a user so they appear in discovery.
func (e *env) newUser(s spec) user {
	e.t.Helper()
	u := e.register(strings.ToLower(s.Name))
	if s.Age == 0 {
		s.Age = 28
	}
	if len(s.InterestedIn) == 0 {
		s.InterestedIn = []string{"man", "woman", "non_binary"}
	}
	if s.MinAge == 0 {
		s.MinAge = 18
	}
	if s.MaxAge == 0 {
		s.MaxAge = 99
	}
	if s.MaxKm == 0 {
		s.MaxKm = 50
	}
	birth := time.Now().AddDate(-s.Age, 0, -10).Format("2006-01-02")
	e.want(e.do("PUT", "/me/profile", u.Token, map[string]any{"firstName": s.Name, "birthDate": birth, "gender": s.Gender, "bio": "Bonjour"}), 200)
	e.want(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": s.InterestedIn, "minAge": s.MinAge, "maxAge": s.MaxAge, "maxDistanceKm": s.MaxKm}), 200)
	if !s.NoLocation {
		lat, lng := s.Lat, s.Lng
		if lat == 0 && lng == 0 {
			lat, lng = parisLat, parisLng
		}
		e.want(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": lat, "longitude": lng, "city": "Paris"}), 200)
	}
	if !s.NoPhoto {
		e.want(e.upload("POST", "/me/photos", u.Token, jpegBytes(int(e.seq.Load()%50))), 201)
	}
	return u
}

func (e *env) feedIDs(u user) []string {
	e.t.Helper()
	r := e.want(e.do("GET", "/discover?limit=20", u.Token, nil), 200).json()
	ids := []string{}
	for _, p := range r["profiles"].([]any) {
		ids = append(ids, p.(map[string]any)["userId"].(string))
	}
	return ids
}

func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func (e *env) swipe(from user, to user, action string) resp {
	return e.do("POST", "/discover/swipes", from.Token, map[string]string{"userId": to.ID, "action": action})
}

// match makes two users like each other and returns the conversation id.
func (e *env) match(a, b user) string {
	e.t.Helper()
	e.want(e.swipe(a, b, "like"), 200)
	r := e.want(e.swipe(b, a, "like"), 200).json()
	if r["matched"] != true {
		e.t.Fatalf("expected match, got %v", r)
	}
	return r["conversationId"].(string)
}

func count(e *env, q string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		e.t.Fatalf("count query: %v", err)
	}
	return n
}
