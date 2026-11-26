package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "Password123"

type captureMailer struct {
	mu    sync.Mutex
	codes map[string]string
}

var codePattern = regexp.MustCompile(`code de réinitialisation : ([A-Z0-9]+)`)

func (m *captureMailer) Send(to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if match := codePattern.FindStringSubmatch(body); match != nil {
		m.codes[to] = match[1]
	}
	return nil
}

func (m *captureMailer) code(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.codes[to]
}

type env struct {
	t       *testing.T
	server  *httptest.Server
	pool    *pgxpool.Pool
	mailer  *captureMailer
	app     *App
	uploads string
	emails  []string
	mu      sync.Mutex
}

var testPool struct {
	once sync.Once
	pool *pgxpool.Pool
	err  error
}

func sharedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("integration tests need DATABASE_URL_TEST (a disposable PostgreSQL database)")
	}
	testPool.once.Do(func() {
		ctx := context.Background()
		testPool.pool, testPool.err = db.Connect(ctx, url)
		if testPool.err == nil {
			testPool.err = db.RunMigrations(ctx, testPool.pool)
		}
	})
	if testPool.err != nil {
		t.Fatalf("test database: %v", testPool.err)
	}
	return testPool.pool
}

func newEnv(t *testing.T, opts ...func(*Options)) *env {
	t.Helper()
	pool := sharedPool(t)
	uploads := t.TempDir()
	storage, err := media.NewLocalStorage(uploads)
	if err != nil {
		t.Fatal(err)
	}
	mailer := &captureMailer{codes: map[string]string{}}
	o := Options{Mailer: mailer, Storage: storage, DisableRateLimits: true}
	for _, fn := range opts {
		fn(&o)
	}
	cfg := config.Config{Env: "test", JWTSecret: "integration-secret-0123456789abcdefghij", UploadDir: uploads}
	application, err := New(cfg, pool, o)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, server: httptest.NewServer(application.Router), pool: pool, mailer: mailer, app: application, uploads: uploads}
	t.Cleanup(func() {
		e.server.Close()
		e.mu.Lock()
		defer e.mu.Unlock()
		if len(e.emails) > 0 {
			if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, e.emails); err != nil {
				t.Errorf("cleanup: %v", err)
			}
		}
	})
	return e
}

var emailSeq atomic.Int64

func (e *env) email(prefix string) string {
	addr := fmt.Sprintf("%s_%d_%d@integration.test", prefix, time.Now().UnixNano(), emailSeq.Add(1))
	e.mu.Lock()
	e.emails = append(e.emails, addr)
	e.mu.Unlock()
	return addr
}

type response struct {
	Status int
	Body   []byte
}

func (r response) json(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %s: %v", string(r.Body), err)
	}
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
	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.send(req)
}

func (e *env) send(req *http.Request) response {
	e.t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return response{Status: res.StatusCode, Body: raw}
}

func (e *env) expect(r response, status int) response {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected HTTP %d, got %d: %s", status, r.Status, string(r.Body))
	}
	return r
}

type session struct {
	Token   string
	Refresh string
	UserID  string
	Email   string
}

func (e *env) register(prefix string) session {
	e.t.Helper()
	email := e.email(prefix)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": email, "password": testPassword, "acceptedTerms": true}), 201)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.json(e.t, &out)
	return session{Token: out.AccessToken, Refresh: out.RefreshToken, UserID: out.User.ID, Email: email}
}

type member struct {
	session
	Name string
}

type memberSpec struct {
	Name         string
	Gender       string
	Age          int
	InterestedIn []string
	MinAge       int
	MaxAge       int
	MaxDistance  int
	Lat, Lng     float64
	Interests    []string
}

func birthdateForAge(age int) string {
	return time.Now().UTC().AddDate(-age, 0, -10).Format("2006-01-02")
}

// onboard creates a complete, discoverable member.
func (e *env) onboard(spec memberSpec) member {
	e.t.Helper()
	s := e.register(strings.ToLower(spec.Name))
	if spec.MinAge == 0 {
		spec.MinAge, spec.MaxAge = 18, 99
	}
	if spec.MaxDistance == 0 {
		spec.MaxDistance = 50
	}
	if spec.Interests == nil {
		spec.Interests = []string{"hiking", "coffee"}
	}
	e.expect(e.do("PUT", "/profile", s.Token, map[string]any{
		"firstName": spec.Name, "birthdate": birthdateForAge(spec.Age), "gender": spec.Gender,
		"bio": "Bio de " + spec.Name, "interests": spec.Interests,
	}), 200)
	e.expect(e.do("PUT", "/profile/preferences", s.Token, map[string]any{
		"interestedIn": spec.InterestedIn, "minAge": spec.MinAge, "maxAge": spec.MaxAge, "maxDistanceKm": spec.MaxDistance,
	}), 200)
	if spec.Lat != 0 || spec.Lng != 0 {
		e.expect(e.do("PUT", "/profile/location", s.Token, map[string]any{"latitude": spec.Lat, "longitude": spec.Lng}), 204)
	}
	e.uploadPhoto(s.Token, testImage(e.t, 400, 500), 201)
	return member{session: s, Name: spec.Name}
}

func testImage(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *env) multipart(method, path, token string, data []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req, err := http.NewRequest(method, e.server.URL+path, &buf)
	if err != nil {
		e.t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.send(req)
}

type photoList struct {
	Photos []struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Position int    `json:"position"`
	} `json:"photos"`
}

func (e *env) uploadPhoto(token string, data []byte, status int) photoList {
	e.t.Helper()
	r := e.expect(e.multipart("POST", "/photos", token, data), status)
	var out photoList
	if status < 300 {
		r.json(e.t, &out)
	}
	return out
}

type deck struct {
	Profiles []struct {
		UserID     string `json:"userId"`
		FirstName  string `json:"firstName"`
		Age        int    `json:"age"`
		DistanceKm *int   `json:"distanceKm"`
		Photos     []struct {
			URL string `json:"url"`
		} `json:"photos"`
	} `json:"profiles"`
	Reason string `json:"reason"`
}

func (e *env) deck(token string) deck {
	e.t.Helper()
	var d deck
	e.expect(e.do("GET", "/discover?limit=20", token, nil), 200).json(e.t, &d)
	return d
}

func (d deck) has(userID string) bool {
	for _, p := range d.Profiles {
		if p.UserID == userID {
			return true
		}
	}
	return false
}

type swipeResult struct {
	Matched bool `json:"matched"`
	Match   *struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
		Profile        struct {
			UserID string `json:"userId"`
		} `json:"profile"`
	} `json:"match"`
}

func (e *env) swipe(token, target, action string, status int) swipeResult {
	e.t.Helper()
	r := e.expect(e.do("POST", "/swipes", token, map[string]any{"targetUserId": target, "action": action}), status)
	var out swipeResult
	if status == 201 {
		r.json(e.t, &out)
	}
	return out
}

// Paris coordinates and a point ~8 km away; Lyon is ~390 km away.
const (
	parisLat, parisLng   = 48.8566, 2.3522
	nearbyLat, nearbyLng = 48.92, 2.42
	lyonLat, lyonLng     = 45.764, 4.8357
)
