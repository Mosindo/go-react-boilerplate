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
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard)
	middleware.SetRateLimiting(false)
	os.Exit(m.Run())
}

type captureMailer struct {
	mu   sync.Mutex
	last string
	to   string
}

func (m *captureMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.to, m.last = to, body
	return nil
}

func (m *captureMailer) lastBody() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last
}

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	store  *storage.Disk
	mail   *captureMailer
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
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	store, err := storage.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mailer := &captureMailer{}
	e := &env{t: t, pool: pool, store: store, mail: mailer}
	e.router = setupRouter(&app{
		dbPool:    pool,
		jwtSecret: []byte("integration-test-secret-0123456789"),
		store:     store,
		mailer:    mailer,
	})
	t.Cleanup(pool.Close)
	return e
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
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

func (e *env) upload(method, path, token, field string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, "photo.bin")
	if err != nil {
		e.t.Fatal(err)
	}
	part.Write(data)
	w.Close()
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

func pngBytes(w, h int, c color.RGBA) []byte {
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

type geo struct{ lat, lon float64 }

// uniqueGeo gives every test its own corner of the world so discovery results
// are not polluted by users created by other tests in the shared database.
func uniqueGeo() geo {
	return geo{
		lat: float64(rand.Intn(11000)-5500) / 100,
		lon: float64(rand.Intn(34000)-17000) / 100,
	}
}

type user struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

var emailCounter int64
var emailMu sync.Mutex

func uniqueEmail(prefix string) string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailCounter++
	return fmt.Sprintf("%s-%d-%d@test.invalid", prefix, time.Now().UnixNano(), emailCounter)
}

const testPassword = "CorrectHorse42"

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := uniqueEmail(prefix)
	r := e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": testPassword}, ""), http.StatusCreated)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.decode(e.t, &out)
	e.t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})
	return user{ID: out.User.ID, Email: email, Token: out.AccessToken, Refresh: out.RefreshToken}
}

type profileSpec struct {
	name         string
	gender       string
	birth        string
	interestedIn []string
	minAge       int
	maxAge       int
	maxDistance  int
	interests    []string
	at           geo
}

func (e *env) onboard(prefix string, spec profileSpec) user {
	e.t.Helper()
	u := e.register(prefix)
	if spec.birth == "" {
		spec.birth = "1992-04-10"
	}
	if spec.minAge == 0 {
		spec.minAge, spec.maxAge = 18, 99
	}
	if spec.maxDistance == 0 {
		spec.maxDistance = 50
	}
	e.expect(e.do("PUT", "/profile", map[string]any{
		"displayName": spec.name, "birthDate": spec.birth, "gender": spec.gender, "bio": "bio de " + spec.name,
		"city": "Testville", "latitude": spec.at.lat, "longitude": spec.at.lon, "interests": spec.interests,
	}, u.Token), http.StatusOK)
	e.expect(e.do("PUT", "/preferences", map[string]any{
		"interestedIn": spec.interestedIn, "minAge": spec.minAge, "maxAge": spec.maxAge, "maxDistanceKm": spec.maxDistance,
	}, u.Token), http.StatusOK)
	e.expect(e.upload("POST", "/photos", u.Token, "file", pngBytes(64, 64, color.RGBA{180, 90, 60, 255})), http.StatusCreated)
	return u
}

type publicProfile struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Age         int    `json:"age"`
	DistanceKm  *int   `json:"distanceKm"`
	Photos      []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	Interests []struct {
		Slug string `json:"slug"`
	} `json:"interests"`
}

func (e *env) discover(u user) []publicProfile {
	e.t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", nil, u.Token), http.StatusOK)
	var out struct {
		Profiles []publicProfile `json:"profiles"`
	}
	r.decode(e.t, &out)
	return out.Profiles
}

func containsProfile(list []publicProfile, id string) bool {
	for _, p := range list {
		if p.UserID == id {
			return true
		}
	}
	return false
}

type swipeOut struct {
	Action    string `json:"action"`
	Matched   bool   `json:"matched"`
	Duplicate bool   `json:"duplicate"`
	Match     *struct {
		MatchID        string `json:"matchId"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (e *env) swipe(from user, target user, action string) swipeOut {
	e.t.Helper()
	r := e.expect(e.do("POST", "/swipes", map[string]string{"targetUserId": target.ID, "action": action}, from.Token), http.StatusOK)
	var out swipeOut
	r.decode(e.t, &out)
	return out
}

func (e *env) forceMatch(a, b user) string {
	e.t.Helper()
	e.swipe(a, b, "like")
	out := e.swipe(b, a, "like")
	if !out.Matched || out.Match == nil {
		e.t.Fatalf("expected a match, got %+v", out)
	}
	return out.Match.ConversationID
}

func pair(at geo) (profileSpec, profileSpec) {
	return profileSpec{name: "Alice", gender: "woman", interestedIn: []string{"man"}, at: at, interests: []string{"voyage", "cuisine"}},
		profileSpec{name: "Bob", gender: "man", interestedIn: []string{"woman"}, at: at, interests: []string{"voyage"}}
}

func containsStr(s, sub string) bool { return strings.Contains(s, sub) }

func itoa(n int64) string { return fmt.Sprintf("%d", n) }
