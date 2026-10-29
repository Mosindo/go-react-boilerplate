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
	"testing"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "integration-tests-jwt-secret-0123456789abcdef"

// captureMailer records outgoing emails so tests can read password reset codes.
type captureMailer struct {
	mu   sync.Mutex
	mail []capturedMail
}

type capturedMail struct{ To, Body string }

func (m *captureMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mail = append(m.mail, capturedMail{To: to, Body: body})
	return nil
}

func (m *captureMailer) last() capturedMail {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.mail) == 0 {
		return capturedMail{}
	}
	return m.mail[len(m.mail)-1]
}

type testEnv struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	mailer *captureMailer
	store  storage.Storage
	dir    string
}

// newTestEnv connects to DATABASE_URL_TEST, wipes all data and returns a router
// wired exactly like production (minus rate limiting). The database is
// destroyed on every call, so it must be a dedicated test database.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST not set: skipping integration tests")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `TRUNCATE users CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	dir := t.TempDir()
	store, err := storage.NewDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	m := &captureMailer{}
	a := &app{dbPool: pool, jwtSecret: []byte(testJWTSecret), store: store, mailer: m}
	return &testEnv{t: t, pool: pool, router: setupRouter(a), mailer: m, store: store, dir: dir}
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %q: %v", string(r.Body), err)
	}
}

func (e *testEnv) do(method, path, token string, body any) response {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
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
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return response{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *testEnv) expect(r response, status int) response {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, string(r.Body))
	}
	return r
}

func (e *testEnv) upload(token string, data []byte, filename string) response {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("photo", filename)
	_, _ = part.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return response{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type testUser struct {
	ID           string
	Email        string
	Token        string
	RefreshToken string
	PhotoID      string
}

type userSpec struct {
	Name      string
	Gender    string
	BirthYear int // age ~ 2026 - BirthYear
	Lat, Lng  float64
	Interests []int16
	Into      []string
	MinAge    int
	MaxAge    int
	MaxDist   int
	NoPhoto   bool
}

func (s userSpec) withDefaults() userSpec {
	if s.Gender == "" {
		s.Gender = "woman"
	}
	if s.BirthYear == 0 {
		s.BirthYear = 1995
	}
	if s.Lat == 0 && s.Lng == 0 {
		s.Lat, s.Lng = 48.8566, 2.3522 // Paris
	}
	if len(s.Into) == 0 {
		s.Into = []string{"man", "woman", "non_binary"}
	}
	if s.MinAge == 0 {
		s.MinAge = 18
	}
	if s.MaxAge == 0 {
		s.MaxAge = 99
	}
	if s.MaxDist == 0 {
		s.MaxDist = 100
	}
	return s
}

var emailCounter int
var emailMu sync.Mutex

func uniqueEmail(prefix string) string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailCounter++
	return fmt.Sprintf("%s%d@dating.test", prefix, emailCounter)
}

// register creates an account only.
func (e *testEnv) register(prefix string) testUser {
	e.t.Helper()
	email := uniqueEmail(prefix)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123!"}), 201)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.decode(e.t, &out)
	return testUser{ID: out.User.ID, Email: email, Token: out.AccessToken, RefreshToken: out.RefreshToken}
}

// onboard registers a user and completes profile, preferences and photo.
func (e *testEnv) onboard(s userSpec) testUser {
	e.t.Helper()
	s = s.withDefaults()
	u := e.register(s.Name)
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{
		"firstName": s.Name, "birthDate": fmt.Sprintf("%d-03-15", s.BirthYear), "gender": s.Gender,
		"bio": "Hello", "city": "Somewhere", "latitude": s.Lat, "longitude": s.Lng, "interestIds": s.Interests,
	}), 200)
	e.expect(e.do("PUT", "/profile/preferences", u.Token, map[string]any{
		"interestedIn": s.Into, "minAge": s.MinAge, "maxAge": s.MaxAge, "maxDistanceKm": s.MaxDist,
	}), 200)
	if !s.NoPhoto {
		r := e.expect(e.upload(u.Token, pngBytes(400, 500), "me.png"), 201)
		var ph struct {
			ID string `json:"id"`
		}
		r.decode(e.t, &ph)
		u.PhotoID = ph.ID
	}
	return u
}

func (e *testEnv) swipe(u testUser, target testUser, action string) response {
	return e.do("POST", "/swipes", u.Token, map[string]string{"userId": target.ID, "action": action})
}

type discoverResp struct {
	Profiles []struct {
		ID         string `json:"id"`
		FirstName  string `json:"firstName"`
		Age        int    `json:"age"`
		DistanceKm *int   `json:"distanceKm"`
	} `json:"profiles"`
}

func (e *testEnv) discoverIDs(u testUser) []string {
	e.t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=30", u.Token, nil), 200)
	var d discoverResp
	r.decode(e.t, &d)
	ids := make([]string, 0, len(d.Profiles))
	for _, p := range d.Profiles {
		ids = append(ids, p.ID)
	}
	return ids
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}
