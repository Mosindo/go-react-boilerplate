package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
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

	"example.com/api/internal/platform/blob"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "e2e-tests-signing-key-0123456789abcdef"

// ---- harness ----

type captureMailer struct {
	mu   sync.Mutex
	last string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last = to + "\n" + body
	return nil
}

func (m *captureMailer) code(t *testing.T) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	match := regexp.MustCompile(`\b(\d{6})\b`).FindStringSubmatch(m.last)
	if match == nil {
		t.Fatal("no recovery code was emailed")
	}
	return match[1]
}

type env struct {
	t         *testing.T
	router    *gin.Engine
	pool      *pgxpool.Pool
	app       *app
	mail      *captureMailer
	uploadDir string
}

type person struct {
	ID      string
	Email   string
	Token   string
	Refresh string
	Name    string
}

func relaxedLimits() rateLimits {
	l := func() *middleware.Limiter { return middleware.NewLimiter(6_000_000, 1_000_000) }
	return rateLimits{auth: l(), upload: l(), swipe: l(), send: l(), report: l()}
}

func newEnv(t *testing.T) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)

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

	dir := t.TempDir()
	store, err := blob.NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	a := &app{
		dbPool:    pool,
		jwtSecret: []byte(testSecret),
		store:     store,
		mailer:    mail,
		hub:       realtime.NewHub(),
		limits:    relaxedLimits(),
	}
	e := &env{t: t, router: setupRouter(a), pool: pool, app: a, mail: mail, uploadDir: dir}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email LIKE '%@dating.test' AND email LIKE $1`, e.emailPrefix()+"%"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
		pool.Close()
	})
	return e
}

// emailPrefix scopes cleanup to the users created by this test.
func (e *env) emailPrefix() string {
	return strings.ToLower(strings.ReplaceAll(e.t.Name(), "/", "_")) + "_"
}

var emailSeq atomic.Int64

func (e *env) newEmail(label string) string {
	return fmt.Sprintf("%s%s_%d@dating.test", e.emailPrefix(), label, emailSeq.Add(1))
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) json(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatalf("response is not a JSON object (status %d): %s", r.Status, r.Body)
	}
	return out
}

func (r response) code(t *testing.T) string {
	t.Helper()
	code, _ := r.json(t)["code"].(string)
	return code
}

func (e *env) do(method, path, token string, body any) response {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
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
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *env) upload(method, path, token, filename string, data []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	part, err := w.CreatePart(h)
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *env) mustStatus(r response, want int, what string) response {
	e.t.Helper()
	if r.Status != want {
		e.t.Fatalf("%s: expected %d, got %d: %s", what, want, r.Status, r.Body)
	}
	return r
}

func (e *env) register(label string) *person {
	e.t.Helper()
	email := e.newEmail(label)
	r := e.mustStatus(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": "Password123"}), http.StatusCreated, "register")
	body := r.json(e.t)
	user := body["user"].(map[string]any)
	return &person{
		ID:      user["id"].(string),
		Email:   email,
		Token:   body["accessToken"].(string),
		Refresh: body["refreshToken"].(string),
		Name:    label,
	}
}

type profileSpec struct {
	Name      string
	Gender    string
	Seeks     []string
	Age       int
	MinAge    int
	MaxAge    int
	Distance  int
	Lat, Lng  float64
	NoPhoto   bool
	Interests []string
}

func (s profileSpec) withDefaults() profileSpec {
	if s.Age == 0 {
		s.Age = 28
	}
	if s.MinAge == 0 {
		s.MinAge = 18
	}
	if s.MaxAge == 0 {
		s.MaxAge = 60
	}
	if s.Distance == 0 {
		s.Distance = 50
	}
	if s.Lat == 0 && s.Lng == 0 {
		s.Lat, s.Lng = 48.8566, 2.3522
	}
	return s
}

func birthDateFor(age int) string {
	return time.Now().AddDate(-age, 0, -30).Format("2006-01-02")
}

// signup registers a user and completes the whole onboarding.
func (e *env) signup(spec profileSpec) *person {
	e.t.Helper()
	spec = spec.withDefaults()
	p := e.register(spec.Name)
	p.Name = spec.Name

	e.mustStatus(e.do(http.MethodPut, "/me/profile", p.Token, map[string]any{
		"firstName": spec.Name, "birthDate": birthDateFor(spec.Age), "gender": spec.Gender,
		"bio": "Hello, I am " + spec.Name, "interests": spec.Interests,
	}), http.StatusOK, "save profile")
	e.mustStatus(e.do(http.MethodPut, "/me/preferences", p.Token, map[string]any{
		"interestedIn": spec.Seeks, "minAge": spec.MinAge, "maxAge": spec.MaxAge, "maxDistanceKm": spec.Distance,
	}), http.StatusOK, "save preferences")
	e.mustStatus(e.do(http.MethodPut, "/me/location", p.Token, map[string]any{
		"latitude": spec.Lat, "longitude": spec.Lng, "city": "Testville",
	}), http.StatusOK, "save location")
	if !spec.NoPhoto {
		e.mustStatus(e.upload(http.MethodPost, "/photos", p.Token, "me.png", testPNG(600, 800)), http.StatusCreated, "upload photo")
	}
	return p
}

func (e *env) discoverIDs(p *person) []string {
	e.t.Helper()
	r := e.mustStatus(e.do(http.MethodGet, "/discover?limit=20", p.Token, nil), http.StatusOK, "discover")
	var ids []string
	for _, c := range r.json(e.t)["profiles"].([]any) {
		ids = append(ids, c.(map[string]any)["userId"].(string))
	}
	return ids
}

func (e *env) swipe(from *person, to *person, action string) response {
	e.t.Helper()
	return e.do(http.MethodPost, "/swipes", from.Token, map[string]string{"userId": to.ID, "action": action})
}

// match makes two compatible people like each other and returns the conversation id.
func (e *env) match(a, b *person) string {
	e.t.Helper()
	e.mustStatus(e.swipe(a, b, "like"), http.StatusOK, "first like")
	r := e.mustStatus(e.swipe(b, a, "like"), http.StatusOK, "second like").json(e.t)
	if r["matched"] != true {
		e.t.Fatalf("expected a match, got %v", r)
	}
	return r["match"].(map[string]any)["conversationId"].(string)
}

func contains(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func testPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func testJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: uint8(x % 255), B: uint8(y % 255), A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

// ---- health & transport ----

func TestHealthEndpoint(t *testing.T) {
	e := newEnv(t)
	r := e.do(http.MethodGet, "/health", "", nil)
	if r.Status != http.StatusOK || !strings.Contains(string(r.Body), `"status":"ok"`) {
		t.Fatalf("unexpected health response: %d %s", r.Status, r.Body)
	}
	if r.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers are missing")
	}
}

func TestHealthPropagatesRequestID(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "req-integration-fixed-id")
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "req-integration-fixed-id" {
		t.Fatalf("expected the request id to be echoed, got %q", got)
	}
}

func TestHealthReturns503WhenDBUnavailable(t *testing.T) {
	e := newEnv(t)
	// A second pool, closed, stands in for an unreachable database.
	pool, err := db.Connect(context.Background(), os.Getenv("DATABASE_URL_TEST"))
	if err != nil {
		t.Skip("postgres unavailable")
	}
	pool.Close()
	broken := *e.app
	broken.dbPool = pool
	rec := httptest.NewRecorder()
	setupRouter(&broken).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAuthRateLimitReturns429(t *testing.T) {
	e := newEnv(t)
	strict := *e.app
	strict.limits = relaxedLimits()
	strict.limits.auth = middleware.NewLimiter(60, 3)
	router := setupRouter(&strict)

	var last int
	var retryAfter string
	for i := 0; i < 6; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"a@dating.test","password":"whatever12"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		last, retryAfter = rec.Code, rec.Header().Get("Retry-After")
	}
	if last != http.StatusTooManyRequests || retryAfter == "" {
		t.Fatalf("expected 429 with Retry-After, got %d %q", last, retryAfter)
	}
}
