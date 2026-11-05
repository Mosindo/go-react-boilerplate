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

const testSecret = "integration-test-jwt-secret-0123456789"

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

func (m *captureMailer) body(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last[to]
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	mail   *captureMailer
}

func newEnv(t *testing.T) *env { return newEnvWith(t, 10000, 10000) }

func newEnvWith(t *testing.T, authPerMinute, authBurst int) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST or DATABASE_URL must be set for integration tests")
	}
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard)
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	// Fresh schema for every test: the test database is disposable.
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mail := &captureMailer{}
	r := setupRouter(&app{dbPool: pool, jwtSecret: []byte(testSecret), appBaseURL: "http://app.test", mailer: mail, authPerMinute: authPerMinute, authBurst: authBurst})
	return &env{t: t, pool: pool, router: r, mail: mail}
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) into(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (r resp) json(t *testing.T) map[string]any {
	t.Helper()
	m := map[string]any{}
	r.into(t, &m)
	return m
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func (e *env) upload(method, path, token string, data []byte, field string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile(field, "photo.jpg")
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes()}
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 120, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

type user struct {
	ID, Email, Token, Refresh string
}

type userSpec struct {
	Name         string
	Gender       string
	InterestedIn []string
	Age          int
	MinAge       int
	MaxAge       int
	Lat, Lng     *float64
	MaxDistance  *int
	Interests    []string
	NoPhoto      bool
}

func ptr[T any](v T) *T { return &v }

// onboard registers a user and completes the whole onboarding flow.
func (e *env) onboard(s userSpec) user {
	e.t.Helper()
	email := strings.ToLower(s.Name) + "@example.test"
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "correct-horse-1"}), 201)
	var a struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct{ ID string }
	}
	r.into(e.t, &a)
	u := user{ID: a.User.ID, Email: email, Token: a.AccessToken, Refresh: a.RefreshToken}

	birth := time.Now().AddDate(-s.Age, 0, -3).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": s.Name, "birthDate": birth, "gender": s.Gender, "bio": "Bio de " + s.Name,
		"city": "Lyon", "interests": s.Interests,
	}), 200)
	minAge, maxAge := s.MinAge, s.MaxAge
	if minAge == 0 {
		minAge = 18
	}
	if maxAge == 0 {
		maxAge = 99
	}
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{
		"interestedIn": s.InterestedIn, "minAge": minAge, "maxAge": maxAge, "maxDistanceKm": s.MaxDistance,
	}), 200)
	if s.Lat != nil {
		e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": *s.Lat, "longitude": *s.Lng}), 200)
	}
	if !s.NoPhoto {
		e.expect(e.upload("POST", "/me/photos", u.Token, pngBytes(200, 300), "photo"), 201)
	}
	return u
}

func feedIDs(t *testing.T, e *env, u user) []string {
	t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", u.Token, nil), 200)
	var out struct {
		Profiles []struct{ ID string } `json:"profiles"`
	}
	r.into(t, &out)
	ids := []string{}
	for _, p := range out.Profiles {
		ids = append(ids, p.ID)
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

func (e *env) swipe(from user, target user, action string) map[string]any {
	e.t.Helper()
	return e.expect(e.do("POST", "/swipes", from.Token, map[string]string{"userId": target.ID, "action": action}), 200).json(e.t)
}

// match makes two onboarded users like each other and returns the conversation id.
func (e *env) match(a, b user) string {
	e.t.Helper()
	e.swipe(a, b, "like")
	res := e.swipe(b, a, "like")
	if res["matched"] != true {
		e.t.Fatalf("expected match, got %v", res)
	}
	return res["conversationId"].(string)
}

func fmtID(prefix string, i int) string { return fmt.Sprintf("%s%d", prefix, i) }
