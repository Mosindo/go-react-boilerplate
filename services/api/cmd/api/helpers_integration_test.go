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
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func init() { gin.SetMode(gin.TestMode) }

// captureMailer records the last body sent to each address.
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

func (m *captureMailer) codeFor(t *testing.T, email string) string {
	t.Helper()
	for i := 0; i < 50; i++ {
		m.mu.Lock()
		body := m.last[email]
		m.mu.Unlock()
		if body != "" {
			const marker = "réinitialisation : "
			idx := strings.Index(body, marker)
			if idx < 0 {
				t.Fatalf("unexpected mail body %q", body)
			}
			return strings.Fields(body[idx+len(marker):])[0]
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no reset mail for %s", email)
	return ""
}

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mail   *captureMailer
	emails []string
}

func newEnv(t *testing.T) *env { return newEnvWith(t, 100000, 100000) }

func newEnvWith(t *testing.T, authPerMinute, apiPerMinute int) *env {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("skip integration tests: postgres unavailable (%v)", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	mail := &captureMailer{}
	e := &env{t: t, pool: pool, mail: mail}
	e.router = setupRouter(&app{
		dbPool:        pool,
		jwtSecret:     []byte("integration-test-secret-0123456789abcdef"),
		mailer:        mail,
		authPerMinute: authPerMinute,
		apiPerMinute:  apiPerMinute,
	})
	t.Cleanup(func() {
		if len(e.emails) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, e.emails)
		}
		pool.Close()
	})
	return e
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) JSON(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("decode %q: %v", string(r.Body), err)
	}
	return m
}

func (e *env) do(method, path string, payload any, token string) resp {
	e.t.Helper()
	var body *bytes.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatal(err)
		}
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, body)
	req.RemoteAddr = "192.0.2.50:4000"
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.serve(req)
}

func (e *env) serve(req *http.Request) resp {
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *env) upload(method, path, field, filename string, data []byte, token string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, filename))
	h.Set("Content-Type", "application/octet-stream")
	part, _ := w.CreatePart(h)
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "192.0.2.50:4000"
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.serve(req)
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected %d, got %d body=%s", status, r.Status, string(r.Body))
	}
	return r
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 3), G: uint8(y * 3), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type user struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

var emailSeq int
var emailMu sync.Mutex

func uniqueEmail(prefix string) string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailSeq++
	return fmt.Sprintf("%s-%d-%d@itest.example.com", prefix, time.Now().UnixNano(), emailSeq)
}

const testPassword = "Password123"

// register creates an account (no profile yet).
func (e *env) register(prefix string) user {
	e.t.Helper()
	email := uniqueEmail(prefix)
	e.emails = append(e.emails, email)
	r := e.expect(e.do(http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, ""), http.StatusCreated)
	body := r.JSON(e.t)
	u := body["user"].(map[string]any)
	return user{ID: u["id"].(string), Email: email, Token: body["accessToken"].(string), Refresh: body["refreshToken"].(string)}
}

func birthDateForAge(age int) string {
	return time.Now().AddDate(-age, 0, -2).Format("2006-01-02")
}

type profileOpts struct {
	gender       string
	interestedIn []string
	age          int
	lat, lng     *float64
	interests    []string
}

// onboard creates the profile, preferences (age band == own age) and one photo.
func (e *env) onboard(u user, name string, o profileOpts) {
	e.t.Helper()
	if o.age == 0 {
		o.age = 30
	}
	body := map[string]any{
		"firstName": name, "birthDate": birthDateForAge(o.age), "gender": o.gender,
		"bio": "Bio de " + name, "city": "Lyon", "interests": o.interests,
	}
	if body["interests"] == nil {
		body["interests"] = []string{}
	}
	if o.lat != nil {
		body["latitude"], body["longitude"] = *o.lat, *o.lng
	}
	e.expect(e.do(http.MethodPut, "/me/profile", body, u.Token), http.StatusOK)
	e.expect(e.do(http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": o.interestedIn, "minAge": o.age, "maxAge": o.age, "maxDistanceKm": 100,
		"discoverable": true, "showDistance": true,
	}, u.Token), http.StatusOK)
	e.expect(e.upload(http.MethodPost, "/me/photos", "photo", "me.png", pngBytes(200, 200), u.Token), http.StatusCreated)
}

// pair returns an onboarded man and woman who are compatible.
func (e *env) pair() (user, user) {
	a := e.register("man")
	b := e.register("woman")
	e.onboard(a, "Adam", profileOpts{gender: "man", interestedIn: []string{"woman"}})
	e.onboard(b, "Berta", profileOpts{gender: "woman", interestedIn: []string{"man"}})
	return a, b
}

func (e *env) match(a, b user) string {
	e.t.Helper()
	e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": b.ID, "action": "like"}, a.Token), http.StatusOK)
	r := e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": a.ID, "action": "like"}, b.Token), http.StatusOK).JSON(e.t)
	m, ok := r["match"].(map[string]any)
	if !ok {
		e.t.Fatalf("expected a match, got %v", r)
	}
	return m["id"].(string)
}

func (e *env) discoverIDs(u user) []string {
	e.t.Helper()
	r := e.expect(e.do(http.MethodGet, "/discover?limit=20", nil, u.Token), http.StatusOK).JSON(e.t)
	var ids []string
	for _, p := range r["profiles"].([]any) {
		ids = append(ids, p.(map[string]any)["userId"].(string))
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

func httpRequest(path, token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "192.0.2.50:4000"
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}
