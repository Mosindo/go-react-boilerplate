package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testEmailDomain = "@itest.invalid"

type mailCapture struct {
	mu   sync.Mutex
	last map[string]string
}

func (m *mailCapture) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.last == nil {
		m.last = map[string]string{}
	}
	m.last[to] = body
	return nil
}

func (m *mailCapture) body(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last[to]
}

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mail   *mailCapture
	app    *app
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func newEnv(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST or DATABASE_URL must be set for integration tests")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("migrations: %v", err)
	}
	files, err := storage.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mail := &mailCapture{}
	a := &app{dbPool: pool, jwtSecret: []byte("integration-secret-0123456789abcdef"), files: files, mailer: mail, hub: realtime.NewHub(), appBaseURL: "http://app.test"}
	e := &env{t: t, router: setupRouter(a), pool: pool, mail: mail, app: a}
	e.cleanup()
	t.Cleanup(func() { e.cleanup(); pool.Close() })
	return e
}

func (e *env) cleanup() {
	_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE email LIKE '%`+testEmailDomain+`'`)
	_, _ = e.pool.Exec(context.Background(), `DELETE FROM reports WHERE reporter_id IS NULL AND reported_id IS NULL`)
}

func (e *env) do(method, path string, payload any, token string) resp {
	e.t.Helper()
	var body *bytes.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *env) upload(token, filename string, content []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", filename)
	_, _ = part.Write(content)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/profile/photos", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
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

var emailSeq atomic.Int64

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d%s", prefix, emailSeq.Add(1), testEmailDomain)
}

type testUser struct {
	ID, Email, Token, Refresh string
}

type profileSpec struct {
	Name, Gender, BirthDate string
	InterestedIn            []string
	Lat, Lon                float64
	Interests               []string
	MaxDistance             int
}

func (e *env) register(prefix string) testUser {
	e.t.Helper()
	email := uniqueEmail(prefix)
	r := e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": "correct-horse"}, ""), 201)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.json(e.t, &out)
	return testUser{ID: out.User.ID, Email: email, Token: out.AccessToken, Refresh: out.RefreshToken}
}

func (e *env) setProfile(u testUser, s profileSpec) resp {
	e.t.Helper()
	if s.MaxDistance == 0 {
		s.MaxDistance = 50
	}
	body := map[string]any{
		"firstName": s.Name, "birthDate": s.BirthDate, "gender": s.Gender, "bio": "Hello", "city": "Testville",
		"latitude": s.Lat, "longitude": s.Lon, "interests": s.Interests,
		"preferences": map[string]any{"interestedIn": s.InterestedIn, "ageMin": 18, "ageMax": 60, "maxDistanceKm": s.MaxDistance},
	}
	return e.do("PUT", "/profile/me", body, u.Token)
}

func samplePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// onboarded registers a user with a complete profile (details + preferences + one photo).
func (e *env) onboarded(prefix string, s profileSpec) testUser {
	e.t.Helper()
	u := e.register(prefix)
	e.expect(e.setProfile(u, s), 200)
	e.expect(e.upload(u.Token, "me.png", samplePNG(400, 300)), 201)
	return u
}
