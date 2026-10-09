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
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func init() { gin.SetMode(gin.TestMode) }

// captureMailer records outgoing e-mails so tests can read password-reset codes.
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

func (m *captureMailer) codeFor(t *testing.T, to string) string {
	t.Helper()
	// Mails are sent in the background: wait briefly for delivery.
	for i := 0; i < 100; i++ {
		m.mu.Lock()
		body := m.last[to]
		m.mu.Unlock()
		if match := regexp.MustCompile(`: ([A-Z2-9]{8})\n`).FindStringSubmatch(body); match != nil {
			return match[1]
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no reset code mailed to %s", to)
	return ""
}

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	app    *app
	mailer *captureMailer
	store  storage.Store
}

func newEnv(t *testing.T) *env {
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
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mailer := &captureMailer{}
	a := &app{
		dbPool:    pool,
		jwtSecret: []byte("integration-test-secret-integration-test"),
		store:     store,
		mailer:    mailer,
		hub:       realtime.NewHub(),
		tickets:   realtime.NewTickets(),
	}
	return &env{t: t, router: setupRouter(a), pool: pool, app: a, mailer: mailer, store: store}
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (r resp) m(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	r.json(t, &out)
	return out
}

func (e *env) do(method, path string, payload any, token string) resp {
	e.t.Helper()
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			e.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *env) upload(method, path, token, filename string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", filename)
	_, _ = fw.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
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

var (
	emailSeq int64
	emailMu  sync.Mutex
)

func uniqueEmail() string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailSeq++
	return fmt.Sprintf("t%d-%d@test.example.org", time.Now().UnixNano(), emailSeq)
}

type user struct {
	ID, Email, Token, Refresh string
}

func (e *env) register(email string) *user {
	e.t.Helper()
	if email == "" {
		email = uniqueEmail()
	}
	// Every registration comes from its own address so the per-IP signup limiter
	// (covered by its own test) does not throttle fixtures.
	body, _ := json.Marshal(map[string]any{"email": email, "password": "correct horse battery"})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:4000", rand.Intn(250)+1, rand.Intn(250)+1, rand.Intn(250)+1)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	r := e.expect(resp{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}, http.StatusCreated)
	m := r.m(e.t)
	u := m["user"].(map[string]any)
	usr := &user{ID: u["id"].(string), Email: email, Token: m["accessToken"].(string), Refresh: m["refreshToken"].(string)}
	e.t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, usr.ID)
	})
	return usr
}

// area is an isolated geographic cell so leftover data from other tests never leaks into discovery.
type area struct{ lat, lon float64 }

func newArea() area {
	return area{lat: -60 + rand.Float64()*120, lon: -170 + rand.Float64()*340}
}

type persona struct {
	Name         string
	Gender       string
	Age          int
	InterestedIn []string
	MinAge       int
	MaxAge       int
	MaxDistance  int
	Dlat         float64 // offset in degrees of latitude from the area centre (1 degree ~ 111 km)
	Photos       int
	Discoverable bool
}

func (a area) persona(name, gender string, age int, interestedIn ...string) persona {
	return persona{Name: name, Gender: gender, Age: age, InterestedIn: interestedIn, MinAge: 18, MaxAge: 80, MaxDistance: 50, Photos: 1, Discoverable: true}
}

func birthDateForAge(age int) string {
	return time.Now().UTC().AddDate(-age, 0, -1).Format("2006-01-02")
}

// onboard completes the whole onboarding for a user.
func (e *env) onboard(u *user, a area, p persona) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", map[string]any{
		"firstName": p.Name, "birthDate": birthDateForAge(p.Age), "gender": p.Gender,
		"bio": "bio of " + p.Name, "city": "Testville",
		"latitude": a.lat + p.Dlat, "longitude": a.lon,
		"interests":    []string{"travel", "music"},
		"discoverable": p.Discoverable,
	}, u.Token), http.StatusOK)
	e.expect(e.do("PUT", "/me/preferences", map[string]any{
		"interestedIn": p.InterestedIn, "minAge": p.MinAge, "maxAge": p.MaxAge, "maxDistanceKm": p.MaxDistance,
	}, u.Token), http.StatusOK)
	for i := 0; i < p.Photos; i++ {
		e.expect(e.upload("POST", "/me/photos", u.Token, "p.jpg", testJPEG(400, 400, uint8(i*40))), http.StatusCreated)
	}
}

func testJPEG(w, h int, shade uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: shade, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func testPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: uint8(x % 255), B: 10, A: 128})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// pair onboards a compatible woman/man in a fresh area.
func (e *env) pair() (a, b *user, ar area) {
	ar = newArea()
	a, b = e.register(""), e.register("")
	e.onboard(a, ar, ar.persona("Alice", "woman", 28, "man"))
	e.onboard(b, ar, ar.persona("Bob", "man", 30, "woman"))
	return
}

func (e *env) discoverIDs(u *user) []string {
	e.t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", nil, u.Token), http.StatusOK)
	var out struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	r.json(e.t, &out)
	ids := make([]string, len(out.Profiles))
	for i, p := range out.Profiles {
		ids[i] = p.ID
	}
	return ids
}

func (e *env) swipe(u *user, target, action string) map[string]any {
	e.t.Helper()
	return e.expect(e.do("POST", "/swipes", map[string]any{"userId": target, "action": action}, u.Token), http.StatusOK).m(e.t)
}

// matchPair creates a matched pair and returns the conversation id.
func (e *env) matchPair() (a, b *user, convID string) {
	a, b, _ = e.pair()
	e.swipe(a, b.ID, "like")
	res := e.swipe(b, a.ID, "like")
	if res["matched"] != true {
		e.t.Fatalf("expected a match, got %v", res)
	}
	return a, b, res["match"].(map[string]any)["conversationId"].(string)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
