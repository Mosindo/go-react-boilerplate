package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const testPassword = "correct-horse-battery"

var testSecret = []byte("0123456789abcdef0123456789abcdef-test")

type recordingMailer struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (m *recordingMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m *recordingMailer) last(to string) (mailer.Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.msgs) - 1; i >= 0; i-- {
		if m.msgs[i].To == to {
			return m.msgs[i], true
		}
	}
	return mailer.Message{}, false
}

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	srv  *httptest.Server
	d    *deps
	mail *recordingMailer
	seq  int
	mu   sync.Mutex
}

func hugeLimits() rateLimits {
	return rateLimits{
		Auth:    middleware.NewLimiter(100000, time.Minute),
		Swipes:  middleware.NewLimiter(100000, time.Minute),
		Message: middleware.NewLimiter(100000, time.Minute),
		Reports: middleware.NewLimiter(100000, time.Hour),
	}
}

func newEnv(t *testing.T) *env { return newEnvWith(t, hugeLimits()) }

func newEnvWith(t *testing.T, limits rateLimits) *env {
	t.Helper()
	pool := testutil.NewDB(t)
	rec := &recordingMailer{}
	d := &deps{
		Pool:       pool,
		JWTSecret:  testSecret,
		Mailer:     rec,
		Limits:     limits,
		BcryptCost: bcrypt.MinCost,
	}
	router, err := setupRouter(d)
	if err != nil {
		t.Fatalf("setup router: %v", err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(func() { d.Hub.Close(); srv.Close() })
	return &env{t: t, pool: pool, srv: srv, d: d, mail: rec}
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json() map[string]any {
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		panic(fmt.Sprintf("response is not a JSON object (%d): %s", r.Status, r.Body))
	}
	return m
}

func (r resp) code() string {
	if r.Status < 400 {
		return ""
	}
	c, _ := r.json()["code"].(string)
	return c
}

func (e *env) raw(method, path, token string, contentType string, body io.Reader) resp {
	e.t.Helper()
	req, err := http.NewRequest(method, e.srv.URL+path, body)
	if err != nil {
		e.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Body: b, Header: res.Header}
}

func (e *env) call(method, path, token string, body any) resp {
	e.t.Helper()
	var rd io.Reader
	ct := ""
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		rd = bytes.NewReader(b)
		ct = "application/json"
	}
	return e.raw(method, path, token, ct, rd)
}

func (e *env) upload(method, path, token, filename string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	h.Set("Content-Type", "image/jpeg") // a lie for non-JPEG payloads on purpose
	part, _ := w.CreatePart(h)
	_, _ = part.Write(data)
	_ = w.Close()
	return e.raw(method, path, token, w.FormDataContentType(), &buf)
}

func (e *env) must(r resp, want int) resp {
	e.t.Helper()
	if r.Status != want {
		e.t.Fatalf("expected status %d, got %d: %s", want, r.Status, r.Body)
	}
	return r
}

type user struct {
	e        *env
	ID       string
	Email    string
	Access   string
	Refresh  string
	Name     string
	PhotoIDs []string
}

func (e *env) nextSeq() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seq++
	return e.seq
}

// register creates a bare account (no profile).
func (e *env) register() *user {
	e.t.Helper()
	n := e.nextSeq()
	email := fmt.Sprintf("user%d@test.invalid", n)
	r := e.must(e.call("POST", "/auth/register", "", map[string]string{"email": email, "password": testPassword}), 201)
	j := r.json()
	u := j["user"].(map[string]any)
	return &user{e: e, ID: u["id"].(string), Email: email, Access: j["accessToken"].(string), Refresh: j["refreshToken"].(string)}
}

type spec struct {
	Name         string
	Gender       string
	Age          int
	Lat, Lon     float64
	Interests    []int
	Prefs        map[string]any
	Discoverable *bool
	NoPhoto      bool
	NoLocation   bool
}

func birthFor(age int) string {
	now := time.Now().UTC()
	return time.Date(now.Year()-age, now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -30).Format("2006-01-02")
}

// person registers a user with a complete profile.
func (e *env) person(s spec) *user {
	e.t.Helper()
	u := e.register()
	if s.Name == "" {
		s.Name = fmt.Sprintf("Person%d", e.nextSeq())
	}
	if s.Gender == "" {
		s.Gender = "woman"
	}
	if s.Age == 0 {
		s.Age = 30
	}
	u.Name = s.Name
	body := map[string]any{
		"firstName": s.Name, "birthDate": birthFor(s.Age), "gender": s.Gender, "bio": "hello there",
		"interestIds": s.Interests,
	}
	if s.Interests == nil {
		body["interestIds"] = []int{}
	}
	if s.Discoverable != nil {
		body["isDiscoverable"] = *s.Discoverable
	}
	e.must(e.call("PUT", "/me/profile", u.Access, body), 200)
	if !s.NoLocation {
		e.must(e.call("PUT", "/me/location", u.Access, map[string]any{"latitude": s.Lat, "longitude": s.Lon, "label": "Testville"}), 200)
	}
	if s.Prefs != nil {
		e.must(e.call("PUT", "/me/preferences", u.Access, s.Prefs), 200)
	}
	if !s.NoPhoto {
		r := e.must(e.upload("POST", "/me/photos", u.Access, "a.jpg", testutil.JPEG(200, 300)), 201)
		u.PhotoIDs = append(u.PhotoIDs, r.json()["id"].(string))
	}
	return u
}

func (u *user) call(method, path string, body any) resp {
	u.e.t.Helper()
	return u.e.call(method, path, u.Access, body)
}

// like swipes like and returns the decoded response.
func (u *user) swipe(target *user, action string) resp {
	u.e.t.Helper()
	return u.call("POST", "/swipes", map[string]string{"targetUserId": target.ID, "action": action})
}

// matchWith makes both users like each other and returns (matchId, conversationId).
func (e *env) matchUsers(a, b *user) (string, string) {
	e.t.Helper()
	e.must(a.swipe(b, "like"), 200)
	r := e.must(b.swipe(a, "like"), 200).json()
	if r["matched"] != true {
		e.t.Fatalf("expected match, got %v", r)
	}
	m := r["match"].(map[string]any)
	return m["matchId"].(string), m["conversationId"].(string)
}

func discoverIDs(t *testing.T, u *user, query string) []string {
	t.Helper()
	r := u.e.must(u.call("GET", "/discover?limit=50"+query, nil), 200).json()
	var ids []string
	for _, it := range r["items"].([]any) {
		ids = append(ids, it.(map[string]any)["userId"].(string))
	}
	return ids
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func wsURL(e *env) string { return "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws" }

func count(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count query %q: %v", query, err)
	}
	return n
}

// injectExif inserts an APP1/Exif segment (with GPS-looking text) after SOI.
func injectExif(jpg []byte) []byte {
	payload := append([]byte("Exif\x00\x00"), []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00GPSLatitude=48.85")...)
	seg := []byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte((len(payload) + 2) & 0xff)}
	seg = append(seg, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func decodeConfig(t *testing.T, data []byte) image.Config {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	return cfg
}

func newTestLimiter(limit int, window time.Duration) *middleware.Limiter {
	return middleware.NewLimiter(limit, window)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	if os.Getenv("TEST_LOGS") == "" {
		log.SetOutput(io.Discard)
	}
	os.Exit(m.Run())
}
