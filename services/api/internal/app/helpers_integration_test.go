package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"github.com/jackc/pgx/v5/pgxpool"
)

// captureMailer records outgoing emails so recovery codes can be read by tests.
type captureMailer struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (m *captureMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

var codePattern = regexp.MustCompile(`\b(\d{6})\b`)

func (m *captureMailer) waitCode(t *testing.T, to string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		for i := len(m.sent) - 1; i >= 0; i-- {
			if m.sent[i].To == to {
				code := codePattern.FindStringSubmatch(m.sent[i].Body)
				m.mu.Unlock()
				if code == nil {
					t.Fatalf("no code in email body")
				}
				return code[1]
			}
		}
		m.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no email sent to %s", to)
	return ""
}

type env struct {
	server    *httptest.Server
	pool      *pgxpool.Pool
	mail      *captureMailer
	uploadDir string
	app       *app.App
}

var (
	sharedEnv *env
	envOnce   sync.Once
	envErr    error
	counter   atomic.Int64
)

// setup starts one API instance against DATABASE_URL_TEST. The schema of that
// database is dropped and recreated, so it must be a dedicated test database.
func setup(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		t.Skip("integration tests need DATABASE_URL_TEST (a disposable database: its schema is reset)")
	}
	envOnce.Do(func() {
		ctx := context.Background()
		pool, err := db.Connect(ctx, dsn)
		if err != nil {
			envErr = err
			return
		}
		if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
			envErr = err
			return
		}
		if err := db.RunMigrations(ctx, pool); err != nil {
			envErr = err
			return
		}
		// Migrations must be idempotent across restarts.
		if err := db.RunMigrations(ctx, pool); err != nil {
			envErr = fmt.Errorf("second migration run: %w", err)
			return
		}
		dir, err := os.MkdirTemp("", "uploads-*")
		if err != nil {
			envErr = err
			return
		}
		storage, err := media.NewLocalStorage(dir)
		if err != nil {
			envErr = err
			return
		}
		mail := &captureMailer{}
		a, err := app.New(app.Deps{
			Config: config.Config{
				Env:            config.EnvTest,
				JWTSecret:      "integration-secret-0123456789abcdefghijkl",
				TrustedProxies: []string{"127.0.0.1", "::1"},
			},
			DB:         pool,
			Mailer:     mail,
			Storage:    storage,
			BcryptCost: 4,
		})
		if err != nil {
			envErr = err
			return
		}
		sharedEnv = &env{server: httptest.NewServer(a.Engine), pool: pool, mail: mail, uploadDir: dir, app: a}
	})
	if envErr != nil {
		t.Fatalf("setup: %v", envErr)
	}
	return sharedEnv
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *env) do(t *testing.T, method, path, token string, body any) resp {
	t.Helper()
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.send(t, req)
}

// clientIP gives every test its own (trusted-proxy forwarded) address so per-IP
// rate limits apply per test, as they would per real client.
func clientIP(t *testing.T) string {
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	v := h.Sum32()
	return fmt.Sprintf("10.%d.%d.%d", byte(v>>16), byte(v>>8), byte(v))
}

func (e *env) send(t *testing.T, req *http.Request) resp {
	t.Helper()
	req.Header.Set("X-Forwarded-For", clientIP(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Body: data}
}

func expect(t *testing.T, r resp, status int) {
	t.Helper()
	if r.Status != status {
		t.Fatalf("expected %d, got %d: %s", status, r.Status, r.Body)
	}
}

type session struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type user struct {
	session
	Email    string
	Password string
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d-%d@test.invalid", prefix, time.Now().UnixNano(), counter.Add(1))
}

func (e *env) register(t *testing.T, prefix string) user {
	t.Helper()
	u := user{Email: uniqueEmail(prefix), Password: "s3cret-passw0rd"}
	r := e.do(t, http.MethodPost, "/auth/register", "", map[string]string{"email": u.Email, "password": u.Password})
	expect(t, r, http.StatusCreated)
	r.decode(t, &u.session)
	return u
}

func pngImage(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *env) upload(t *testing.T, method, path, token string, data []byte) resp {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	w.Close()
	req, _ := http.NewRequest(method, e.server.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.send(t, req)
}

type profileSpec struct {
	Name         string
	Gender       string
	Birthdate    string
	InterestedIn []string
	Lat, Lng     float64
	Radius       int
}

// onboard runs the complete onboarding of a user: profile, photo, preferences, location.
func (e *env) onboard(t *testing.T, prefix string, spec profileSpec) user {
	t.Helper()
	u := e.register(t, prefix)
	if spec.Birthdate == "" {
		spec.Birthdate = "1995-05-20"
	}
	if spec.Radius == 0 {
		spec.Radius = 50
	}
	expect(t, e.do(t, http.MethodPut, "/profile", u.AccessToken, map[string]any{
		"firstName": spec.Name, "birthdate": spec.Birthdate, "gender": spec.Gender,
		"bio": "Bio de " + spec.Name, "interestIds": []int{1, 2},
	}), http.StatusOK)
	expect(t, e.upload(t, http.MethodPost, "/photos", u.AccessToken, pngImage(t, 400, 500, color.RGBA{200, 100, 80, 255})), http.StatusCreated)
	expect(t, e.do(t, http.MethodPut, "/profile/preferences", u.AccessToken, map[string]any{
		"interestedIn": spec.InterestedIn, "ageMin": 18, "ageMax": 99, "maxDistanceKm": spec.Radius,
		"isDiscoverable": true, "showDistance": true,
	}), http.StatusOK)
	expect(t, e.do(t, http.MethodPut, "/profile/location", u.AccessToken, map[string]any{
		"latitude": spec.Lat, "longitude": spec.Lng,
	}), http.StatusNoContent)
	return u
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	Interests []struct {
		ID int `json:"id"`
	} `json:"interests"`
}

func (e *env) discover(t *testing.T, u user) []card {
	t.Helper()
	r := e.do(t, http.MethodGet, "/discovery?limit=20", u.AccessToken, nil)
	expect(t, r, http.StatusOK)
	var out struct {
		Profiles []card `json:"profiles"`
	}
	r.decode(t, &out)
	return out.Profiles
}

func containsUser(cards []card, id string) bool {
	for _, c := range cards {
		if c.UserID == id {
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
		User           card   `json:"user"`
	} `json:"match"`
}

func (e *env) like(t *testing.T, from user, to user) swipeResult {
	t.Helper()
	r := e.do(t, http.MethodPost, "/discovery/"+to.User.ID+"/like", from.AccessToken, nil)
	expect(t, r, http.StatusOK)
	var out swipeResult
	r.decode(t, &out)
	return out
}

// matchPair onboards two compatible users at the given place and makes them match.
func (e *env) matchPair(t *testing.T, lat, lng float64) (user, user, string, string) {
	t.Helper()
	a := e.onboard(t, "pair-a", profileSpec{Name: "Anna", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	b := e.onboard(t, "pair-b", profileSpec{Name: "Basile", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	if res := e.like(t, a, b); res.Matched {
		t.Fatal("one-sided like must not match")
	}
	res := e.like(t, b, a)
	if !res.Matched || res.Match == nil {
		t.Fatal("mutual like must match")
	}
	return a, b, res.Match.ID, res.Match.ConversationID
}
