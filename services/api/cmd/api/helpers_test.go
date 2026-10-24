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
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
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

const testJWTSecret = "integration-tests-jwt-secret-0123456789"

// captureMailer records outgoing mail so tests can read reset codes.
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

func (m *captureMailer) body(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last[to]
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	files  *storage.Local
	hub    *realtime.Hub
	mail   *captureMailer
	dir    string
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func newEnv(t *testing.T) *env { return newEnvScaled(t, 1000) }

func newEnvScaled(t *testing.T, scale float64) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("skip integration tests: postgres unavailable (%v)", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	dir := t.TempDir()
	files, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, pool: pool, files: files, hub: realtime.NewHub(), mail: &captureMailer{}, dir: dir}
	e.router = setupRouter(&app{
		dbPool: pool, jwtSecret: []byte(testJWTSecret), files: files, mailer: e.mail, hub: e.hub, limitScale: scale,
	})
	return e
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) json(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (r resp) str(t *testing.T, path ...string) string {
	t.Helper()
	var cur any
	if err := json.Unmarshal(r.Body, &cur); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("path %v: not an object in %s", path, r.Body)
		}
		cur = m[p]
	}
	s, _ := cur.(string)
	return s
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func (e *env) upload(method, path, token string, data []byte, filename string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", filename)
	part.Write(data)
	w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes()}
}

func testPNG(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type person struct {
	ID, Email, Token string
}

var emailSeq int64
var emailMu sync.Mutex

func uniqueEmail(prefix string) string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailSeq++
	return fmt.Sprintf("%s-%d-%d@test.invalid", prefix, time.Now().UnixNano(), emailSeq)
}

// register creates an account and removes it (cascading everything) when the test ends.
func (e *env) register(prefix string) person {
	e.t.Helper()
	email := uniqueEmail(prefix)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Passw0rd!x"}), http.StatusCreated)
	p := person{ID: r.str(e.t, "user", "id"), Email: email, Token: r.str(e.t, "accessToken")}
	e.t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, p.ID)
	})
	return p
}

type profileSpec struct {
	Name         string
	Gender       string
	InterestedIn []string
	Age          int
	Lat, Lon     *float64
	MaxDistance  *int
}

// onboard completes a profile: profile + preferences + one photo (+ optional location).
func (e *env) onboard(p person, s profileSpec) {
	e.t.Helper()
	birth := time.Now().AddDate(-s.Age, 0, -3).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", p.Token, map[string]any{
		"firstName": s.Name, "birthDate": birth, "gender": s.Gender, "bio": "Bio de " + s.Name,
		"city": "Paris", "interests": []string{"voyage", "cuisine"},
	}), http.StatusOK)
	e.expect(e.do("PUT", "/me/preferences", p.Token, map[string]any{
		"interestedIn": s.InterestedIn, "ageMin": 18, "ageMax": 60, "maxDistanceKm": s.MaxDistance,
	}), http.StatusOK)
	e.expect(e.upload("POST", "/me/photos", p.Token, testPNG(40, 60, color.RGBA{200, 80, 80, 255}), "a.png"), http.StatusCreated)
	if s.Lat != nil {
		e.expect(e.do("PUT", "/me/location", p.Token, map[string]any{"latitude": *s.Lat, "longitude": *s.Lon}), http.StatusNoContent)
	}
}

func fptr(v float64) *float64 { return &v }
func iptr(v int) *int         { return &v }

func contains(haystack []byte, needle string) bool { return strings.Contains(string(haystack), needle) }
