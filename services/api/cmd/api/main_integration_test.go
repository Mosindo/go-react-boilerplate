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
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "correct-horse-battery"

var testJWTSecret = []byte("integration-test-secret-0123456789abcdef")

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	if url := testDatabaseURL(); url != "" {
		ctx := context.Background()
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			fmt.Fprintln(os.Stderr, "connect test db:", err)
			os.Exit(1)
		}
		// Start from an empty schema so migrations are exercised from scratch on every run.
		if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
			fmt.Fprintln(os.Stderr, "reset test db:", err)
			os.Exit(1)
		}
		if err := db.RunMigrations(ctx, pool); err != nil {
			fmt.Fprintln(os.Stderr, "migrate test db:", err)
			os.Exit(1)
		}
		pool.Close()
	}
	os.Exit(m.Run())
}

func testDatabaseURL() string {
	if v := os.Getenv("DATABASE_URL_TEST"); v != "" {
		return v
	}
	return os.Getenv("DATABASE_URL")
}

type testEnv struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mail   *mailer.Memory
	upload string
	seq    atomic.Int64
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %q: %v", string(r.Body), err)
	}
}

func (r resp) expect(t *testing.T, status int) resp {
	t.Helper()
	if r.Status != status {
		t.Fatalf("expected status %d, got %d body=%s", status, r.Status, string(r.Body))
	}
	return r
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	url := testDatabaseURL()
	if url == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	// Cascades through every table that references users; interests and migrations are kept.
	if _, err := pool.Exec(context.Background(), `TRUNCATE users, reports RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	env := &testEnv{t: t, pool: pool, mail: &mailer.Memory{}, upload: t.TempDir()}
	router, err := newRouter(deps{
		pool: pool, jwtSecret: testJWTSecret, uploadDir: env.upload,
		resetURL: "https://app.test/reset?token={token}", mailer: env.mail, syncMail: true,
		limits: limits{global: 100000, credentials: 100000, swipes: 100000, messages: 100000, uploads: 100000, reports: 100000},
	})
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	env.router = router
	return env
}

func (e *testEnv) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
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
	return resp{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *testEnv) upload2(method, path, token, field, filename string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, filename)
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
	return resp{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

type session struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

type authBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (e *testEnv) register(label string) session {
	e.t.Helper()
	email := fmt.Sprintf("%s-%d@test.local", label, e.seq.Add(1))
	var a authBody
	e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": testPassword}).expect(e.t, http.StatusCreated).json(e.t, &a)
	return session{ID: a.User.ID, Email: email, Token: a.AccessToken, Refresh: a.RefreshToken}
}

// solidJPEG builds a valid test image.
func solidJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

func solidPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 30, B: 30, A: 128})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type profileSpec struct {
	Name       string
	BirthYear  int
	Gender     string
	Interested []string
	Lat, Lng   float64
	MinAge     int
	MaxAge     int
	MaxDist    int
	Interests  []string
	NoPhoto    bool
}

func (e *testEnv) onboard(s session, spec profileSpec) {
	e.t.Helper()
	if spec.BirthYear == 0 {
		spec.BirthYear = 1992
	}
	if spec.MinAge == 0 {
		spec.MinAge = 18
	}
	if spec.MaxAge == 0 {
		spec.MaxAge = 99
	}
	if spec.MaxDist == 0 {
		spec.MaxDist = 50
	}
	if spec.Interests == nil {
		spec.Interests = []string{"travel", "music"}
	}
	e.do(http.MethodPut, "/profile/me", s.Token, map[string]any{
		"firstName": spec.Name, "birthDate": fmt.Sprintf("%d-06-15", spec.BirthYear), "gender": spec.Gender,
		"bio": "Bio de " + spec.Name, "occupation": "Testeur", "interests": spec.Interests,
	}).expect(e.t, http.StatusOK)
	e.do(http.MethodPut, "/profile/preferences", s.Token, map[string]any{
		"interestedIn": spec.Interested, "minAge": spec.MinAge, "maxAge": spec.MaxAge, "maxDistanceKm": spec.MaxDist,
	}).expect(e.t, http.StatusOK)
	e.do(http.MethodPut, "/profile/location", s.Token, map[string]any{
		"latitude": spec.Lat, "longitude": spec.Lng, "city": "Paris",
	}).expect(e.t, http.StatusOK)
	if !spec.NoPhoto {
		e.upload2(http.MethodPost, "/photos", s.Token, "photo", "me.jpg", solidJPEG(400, 500)).expect(e.t, http.StatusCreated)
	}
}

type card struct {
	ID         string `json:"id"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	City       string `json:"city"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
}

func (e *testEnv) discover(s session) []card {
	e.t.Helper()
	var out struct {
		Profiles []card `json:"profiles"`
	}
	e.do(http.MethodGet, "/discover?limit=30", s.Token, nil).expect(e.t, http.StatusOK).json(e.t, &out)
	return out.Profiles
}

func hasCard(cards []card, id string) bool {
	for _, c := range cards {
		if c.ID == id {
			return true
		}
	}
	return false
}

type swipeBody struct {
	Matched bool `json:"matched"`
	Match   *struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (e *testEnv) swipe(s session, target, action string) resp {
	e.t.Helper()
	return e.do(http.MethodPost, "/swipes", s.Token, map[string]string{"targetUserId": target, "action": action})
}

// Paris-ish coordinates used across tests.
const (
	parisLat, parisLng = 48.8566, 2.3522
	lyonLat, lyonLng   = 45.7640, 4.8357
)

func TestHealthEndpoint(t *testing.T) {
	env := newEnv(t)
	r := env.do(http.MethodGet, "/health", "", nil).expect(t, http.StatusOK)
	if !strings.Contains(string(r.Body), `"status":"ok"`) {
		t.Fatalf("unexpected health body: %s", r.Body)
	}
}
