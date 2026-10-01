package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
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
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type testResponse struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r testResponse) decode(t *testing.T, out any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, out); err != nil {
		t.Fatalf("decode %q: %v", string(r.Body), err)
	}
}

type captureMailer struct {
	mu   sync.Mutex
	mail []string
}

func (m *captureMailer) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mail = append(m.mail, to+"\n"+subject+"\n"+body)
	return nil
}

func (m *captureMailer) waitFor(t *testing.T, to string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		for _, mail := range m.mail {
			if strings.HasPrefix(mail, to+"\n") {
				m.mu.Unlock()
				return mail
			}
		}
		m.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no mail sent to %s", to)
	return ""
}

type testEnv struct {
	router *gin.Engine
	pool   *pgxpool.Pool
	app    *app
	dir    string
	mailer *captureMailer
	emails []string
}

func newTestEnv(t *testing.T) *testEnv {
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
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	store, err := storage.NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	a := &app{
		dbPool: pool, jwtSecret: []byte("integration-test-secret-0123456789"),
		store: store, mailer: mail, appName: "Lumen", rateLimit: false, hub: realtime.NewHub(),
	}
	env := &testEnv{router: setupRouter(a), pool: pool, app: a, dir: dir, mailer: mail}
	t.Cleanup(func() {
		ctx := context.Background()
		if len(env.emails) > 0 {
			_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = ANY($1)`, env.emails)
		}
		pool.Close()
	})
	return env
}

// rebuild recreates the router after tweaking env.app (e.g. enabling rate limits).
func (e *testEnv) rebuild() { e.router = setupRouter(e.app) }

func (e *testEnv) do(t *testing.T, method, path string, payload any, token string) testResponse {
	t.Helper()
	var body *bytes.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
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
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *testEnv) upload(t *testing.T, method, path, token string, data []byte) testResponse {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "photo.jpg")
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
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *testEnv) expect(t *testing.T, r testResponse, status int, what string) testResponse {
	t.Helper()
	if r.Status != status {
		t.Fatalf("%s: expected %d, got %d body=%s", what, status, r.Status, string(r.Body))
	}
	return r
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type testUser struct {
	ID           string
	Email        string
	Token        string
	RefreshToken string
	Password     string
}

type userOpts struct {
	Name         string
	Gender       string
	InterestedIn []string
	Age          int
	AgeMin       int
	AgeMax       int
	Lat, Lng     float64
	NoPhoto      bool
	Hidden       bool
	Interests    []string
}

// area gives each test its own geographic bubble so discovery results are not
// polluted by other tests or leftover rows: everyone within it is ~0 km apart,
// and max distance is small.
type area struct{ lat, lng float64 }

func newArea() area {
	return area{lat: -60 + rand.Float64()*120, lng: -170 + rand.Float64()*340}
}

func (e *testEnv) register(t *testing.T, prefix string) testUser {
	t.Helper()
	email := fmt.Sprintf("%s-%d-%d@test.invalid", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
	e.emails = append(e.emails, email)
	u := testUser{Email: email, Password: "Password123"}
	resp := e.expect(t, e.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": u.Password}, ""), http.StatusCreated, "register")
	var auth struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	resp.decode(t, &auth)
	u.ID, u.Token, u.RefreshToken = auth.User.ID, auth.AccessToken, auth.RefreshToken
	return u
}

// onboard registers a user and completes profile, preferences, location and photo.
func (e *testEnv) onboard(t *testing.T, ar area, o userOpts) testUser {
	t.Helper()
	u := e.register(t, strings.ToLower(o.Name))
	if o.Age == 0 {
		o.Age = 30
	}
	if o.Gender == "" {
		o.Gender = "woman"
	}
	if o.InterestedIn == nil {
		o.InterestedIn = []string{"woman", "man", "non_binary", "other"}
	}
	if o.AgeMin == 0 {
		o.AgeMin = 18
	}
	if o.AgeMax == 0 {
		o.AgeMax = 99
	}
	birth := time.Now().AddDate(-o.Age, 0, -2).Format("2006-01-02")
	profile := map[string]any{"firstName": o.Name, "birthDate": birth, "gender": o.Gender, "bio": "Bio de " + o.Name, "city": "Testville"}
	if o.Interests != nil {
		profile["interests"] = o.Interests
	}
	if o.Hidden {
		profile["discoverable"] = false
	}
	e.expect(t, e.do(t, http.MethodPut, "/me/profile", profile, u.Token), http.StatusOK, "put profile")
	e.expect(t, e.do(t, http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": o.InterestedIn, "ageMin": o.AgeMin, "ageMax": o.AgeMax, "maxDistanceKm": 5,
	}, u.Token), http.StatusOK, "put preferences")
	e.expect(t, e.do(t, http.MethodPut, "/me/location", map[string]any{"latitude": ar.lat, "longitude": ar.lng}, u.Token), http.StatusNoContent, "put location")
	if !o.NoPhoto {
		e.expect(t, e.upload(t, http.MethodPost, "/me/photos", u.Token, jpegBytes(t, 600, 800)), http.StatusCreated, "upload photo")
	}
	return u
}

type profileView struct {
	ID         string `json:"id"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	Interests []struct {
		Slug string `json:"slug"`
	} `json:"interests"`
}

func (e *testEnv) discover(t *testing.T, u testUser) []profileView {
	t.Helper()
	resp := e.expect(t, e.do(t, http.MethodGet, "/discover?limit=20", nil, u.Token), http.StatusOK, "discover")
	var out struct {
		Profiles []profileView `json:"profiles"`
	}
	resp.decode(t, &out)
	return out.Profiles
}

func containsProfile(list []profileView, id string) bool {
	for _, p := range list {
		if p.ID == id {
			return true
		}
	}
	return false
}

type swipeResult struct {
	Action        string `json:"action"`
	Matched       bool   `json:"matched"`
	AlreadySwiped bool   `json:"alreadySwiped"`
	Match         *struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (e *testEnv) swipe(t *testing.T, from testUser, to testUser, action string) swipeResult {
	t.Helper()
	resp := e.expect(t, e.do(t, http.MethodPost, "/swipes", map[string]string{"userId": to.ID, "action": action}, from.Token), http.StatusOK, "swipe "+action)
	var out swipeResult
	resp.decode(t, &out)
	return out
}

// matchUsers makes a and b like each other and returns the conversation id.
func (e *testEnv) matchUsers(t *testing.T, a, b testUser) string {
	t.Helper()
	if r := e.swipe(t, a, b, "like"); r.Matched {
		t.Fatal("first like must not match")
	}
	r := e.swipe(t, b, a, "like")
	if !r.Matched || r.Match == nil {
		t.Fatal("reciprocal like must match")
	}
	return r.Match.ConversationID
}
