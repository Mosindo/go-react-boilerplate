package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
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
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "integration-suite-key-0123456789abcdef"
const testPassword = "Password123"

type testResponse struct {
	Status int
	Body   []byte
}

func (r testResponse) String() string {
	return fmt.Sprintf("status=%d body=%s", r.Status, string(r.Body))
}

// captureMailer records emails so tests can read password reset codes.
type captureMailer struct {
	mu   sync.Mutex
	sent map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent[to] = body
	return nil
}

var codePattern = regexp.MustCompile(`\b\d{6}\b`)

func (m *captureMailer) codeFor(t *testing.T, email string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	code := codePattern.FindString(m.sent[email])
	if code == "" {
		t.Fatalf("no reset code sent to %s", email)
	}
	return code
}

type testEnv struct {
	router *gin.Engine
	pool   *pgxpool.Pool
	mail   *captureMailer
	hub    *realtime.Hub
	emails []string
}

func generousLimits() RateLimits {
	return RateLimits{
		Global:   ratelimit.NewMemory(100000, time.Minute),
		Auth:     ratelimit.NewMemory(100000, time.Minute),
		Swipes:   ratelimit.NewMemory(100000, time.Minute),
		Messages: ratelimit.NewMemory(100000, time.Minute),
		Uploads:  ratelimit.NewMemory(100000, time.Minute),
		Reports:  ratelimit.NewMemory(100000, time.Minute),
		Location: ratelimit.NewMemory(100000, time.Minute),
	}
}

func newEnv(t *testing.T) *testEnv {
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
	storage, err := media.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env := &testEnv{pool: pool, mail: &captureMailer{sent: map[string]string{}}, hub: realtime.NewHub()}
	env.router = setupRouter(&app{
		dbPool:    pool,
		jwtSecret: []byte(testJWTSecret),
		storage:   storage,
		mailer:    env.mail,
		hub:       env.hub,
		limits:    generousLimits(),
	})
	t.Cleanup(func() {
		if len(env.emails) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, env.emails)
		}
		pool.Close()
	})
	return env
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d_%d@example.test", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
}

func (e *testEnv) do(t *testing.T, method, path string, payload any, token string) testResponse {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			t.Fatalf("marshal: %v", err)
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
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *testEnv) upload(t *testing.T, method, path string, data []byte, token string) testResponse {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes()}
}

func expect(t *testing.T, r testResponse, status int, what string) {
	t.Helper()
	if r.Status != status {
		t.Fatalf("%s: expected %d, got %s", what, status, r)
	}
}

func decode[T any](t *testing.T, r testResponse) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("decode %T: %v (%s)", v, err, r)
	}
	return v
}

func pngImage(t *testing.T, w, h int, tint uint8) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: tint, G: uint8(x), B: uint8(y), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type authPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type member struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

func (e *testEnv) register(t *testing.T, prefix string) member {
	t.Helper()
	email := uniqueEmail(prefix)
	e.emails = append(e.emails, email)
	r := e.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, "")
	expect(t, r, http.StatusCreated, "register")
	a := decode[authPayload](t, r)
	return member{ID: a.User.ID, Email: email, Token: a.AccessToken, Refresh: a.RefreshToken}
}

type profileSpec struct {
	Name         string
	Birthdate    string
	Gender       string
	InterestedIn []string
	Lat, Lng     float64
	NoLocation   bool
}

// onboard runs the whole onboarding: profile, preferences, photo, location.
func (e *testEnv) onboard(t *testing.T, prefix string, spec profileSpec) member {
	t.Helper()
	m := e.register(t, prefix)
	if spec.Birthdate == "" {
		spec.Birthdate = "1995-06-15"
	}
	expect(t, e.do(t, http.MethodPut, "/profile", map[string]any{
		"firstName": spec.Name, "birthdate": spec.Birthdate, "gender": spec.Gender,
		"bio": "Hello", "relationshipGoal": "long_term", "city": "Lyon",
	}, m.Token), http.StatusOK, "upsert profile")
	expect(t, e.do(t, http.MethodPut, "/profile/preferences", map[string]any{
		"interestedIn": spec.InterestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50,
	}, m.Token), http.StatusOK, "preferences")
	expect(t, e.upload(t, http.MethodPost, "/profile/photos", pngImage(t, 400, 500, 10), m.Token), http.StatusCreated, "upload photo")
	if !spec.NoLocation {
		lat, lng := spec.Lat, spec.Lng
		if lat == 0 && lng == 0 {
			lat, lng = 45.7640, 4.8357
		}
		expect(t, e.do(t, http.MethodPut, "/profile/location", map[string]any{"latitude": lat, "longitude": lng}, m.Token), http.StatusOK, "location")
	}
	return m
}

type publicProfile struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        *int   `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		URL string `json:"url"`
	} `json:"photos"`
}

func (e *testEnv) discover(t *testing.T, m member) []publicProfile {
	t.Helper()
	r := e.do(t, http.MethodGet, "/discover?limit=30", nil, m.Token)
	expect(t, r, http.StatusOK, "discover")
	return decode[struct {
		Profiles []publicProfile `json:"profiles"`
	}](t, r).Profiles
}

func containsProfile(list []publicProfile, id string) bool {
	for _, p := range list {
		if p.UserID == id {
			return true
		}
	}
	return false
}

type swipeResp struct {
	Matched bool `json:"matched"`
	Match   *struct {
		ID string `json:"id"`
	} `json:"match"`
}

func (e *testEnv) swipe(t *testing.T, actor member, target, decision string) testResponse {
	t.Helper()
	return e.do(t, http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "decision": decision}, actor.Token)
}

// match makes a and b like each other and returns the match id.
func (e *testEnv) match(t *testing.T, a, b member) string {
	t.Helper()
	expect(t, e.swipe(t, a, b.ID, "like"), http.StatusOK, "a likes b")
	r := e.swipe(t, b, a.ID, "like")
	expect(t, r, http.StatusOK, "b likes a")
	s := decode[swipeResp](t, r)
	if !s.Matched || s.Match == nil {
		t.Fatalf("expected a match: %s", r)
	}
	return s.Match.ID
}

func newTightLimiter(n int) *ratelimit.Memory { return ratelimit.NewMemory(n, time.Minute) }

func mustStorage(t *testing.T) media.Storage {
	t.Helper()
	s, err := media.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
