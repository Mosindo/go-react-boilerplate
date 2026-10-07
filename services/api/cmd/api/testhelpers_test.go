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
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-secret-0123456789abcdef0123"

type capturedMail struct {
	mu   sync.Mutex
	mail []string
}

func (c *capturedMail) Send(_ context.Context, to, _ string, body string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mail = append(c.mail, to+"\n"+body)
	return nil
}

func (c *capturedMail) last(to string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.mail) - 1; i >= 0; i-- {
		if strings.HasPrefix(c.mail[i], to+"\n") {
			return c.mail[i]
		}
	}
	return ""
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	mail   *capturedMail
	emails []string
	lat    float64
	lng    float64
}

func newEnv(t *testing.T) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set; integration tests need PostgreSQL")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mail := &capturedMail{}
	router, err := setupRouter(&app{
		dbPool:    pool,
		jwtSecret: []byte(testSecret),
		uploadDir: t.TempDir(),
		mailer:    mail,
	})
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	e := &env{
		t: t, pool: pool, router: router, mail: mail,
		// An isolated spot on the globe keeps each test's users apart from any leftovers.
		lat: rand.Float64()*120 - 60,
		lng: rand.Float64()*340 - 170,
	}
	t.Cleanup(func() {
		for _, email := range e.emails {
			_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM reports WHERE reporter_id IS NULL AND target_id IS NULL`)
		pool.Close()
	})
	return e
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
	return m
}

func (r resp) items(t *testing.T) []map[string]any {
	t.Helper()
	raw, _ := r.json(t)["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, v := range raw {
		out = append(out, v.(map[string]any))
	}
	return out
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", rand.Intn(250), rand.Intn(250), rand.Intn(250)) // keeps per-IP limiters independent
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func pngBytes(w, h int, seed uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x) + seed, G: uint8(y), B: seed, A: 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func (e *env) upload(method, path, token string, data []byte, filename string) resp {
	e.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", filename)
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "10.1.1.1:1"
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

type person struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

func (e *env) register(prefix string) person {
	e.t.Helper()
	email := fmt.Sprintf("%s-%d-%d@test.invalid", prefix, time.Now().UnixNano(), rand.Intn(1e6))
	e.emails = append(e.emails, email)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": "Sup3rSecret!"}), 201).json(e.t)
	user := r["user"].(map[string]any)
	if user["email"] != email {
		e.t.Fatalf("email should be normalised, got %v", user["email"])
	}
	return person{ID: user["id"].(string), Email: email, Token: r["accessToken"].(string), Refresh: r["refreshToken"].(string)}
}

type profileSpec struct {
	Name      string
	Gender    string
	BirthDate string
	Interests []string
	Interest  []string
}

// onboard creates profile + location + one photo, returning the person ready to swipe.
func (e *env) onboard(prefix, gender string, interests ...string) person {
	e.t.Helper()
	p := e.register(prefix)
	e.expect(e.do("PUT", "/profile/me", p.Token, map[string]any{
		"firstName": strings.ToUpper(prefix[:1]) + prefix[1:], "birthDate": "1994-06-15", "gender": gender,
		"bio": "Hello there", "city": "Testville", "interests": interests,
	}), 200)
	e.expect(e.do("PUT", "/profile/me/location", p.Token, map[string]float64{
		"latitude": e.lat + rand.Float64()*0.05, "longitude": e.lng + rand.Float64()*0.05,
	}), 204)
	e.expect(e.upload("POST", "/photos", p.Token, pngBytes(320, 400, 7), "me.png"), 201)
	return p
}

func (e *env) swipe(p person, target person, action string) resp {
	return e.do("POST", "/swipes", p.Token, map[string]string{"targetUserId": target.ID, "action": action})
}

func containsUser(items []map[string]any, id string) bool {
	for _, it := range items {
		if it["userId"] == id {
			return true
		}
	}
	return false
}
