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

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "Password123"

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

var resetCodePattern = regexp.MustCompile(`: ([A-Z0-9]{8})`)

func (m *captureMailer) lastCodeFor(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.sent) - 1; i >= 0; i-- {
		if m.sent[i].To == email {
			if match := resetCodePattern.FindStringSubmatch(m.sent[i].Body); match != nil {
				return match[1]
			}
		}
	}
	return ""
}

type testEnv struct {
	t       *testing.T
	router  *gin.Engine
	pool    *pgxpool.Pool
	mail    *captureMailer
	hub     *realtime.Hub
	storage *storage.Local
	uploads string
	emails  []string
	// Each test gets its own geographic area so runs do not interfere.
	baseLat float64
	baseLon float64
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("integration tests need DATABASE_URL_TEST or DATABASE_URL")
	}
	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("migrations: %v", err)
	}

	uploads := t.TempDir()
	store, err := storage.NewLocal(uploads)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	hub := realtime.NewHub()
	router, err := setupRouter(Dependencies{
		Config: config.Config{
			Env:       config.EnvTest,
			JWTSecret: "integration-secret-0123456789abcdef-0123456789",
			UploadDir: uploads,
		},
		DB:      pool,
		Storage: store,
		Mailer:  mail,
		Hub:     hub,
	})
	if err != nil {
		t.Fatal(err)
	}

	env := &testEnv{
		t: t, router: router, pool: pool, mail: mail, hub: hub, storage: store, uploads: uploads,
		baseLat: -50 + rand.Float64()*100,
		baseLon: -170 + rand.Float64()*340,
	}
	t.Cleanup(func() {
		if len(env.emails) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM conversations WHERE id IN (
				SELECT cp.conversation_id FROM conversation_participants cp JOIN users u ON u.id = cp.user_id WHERE u.email = ANY($1))`, env.emails)
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, env.emails)
		}
		pool.Close()
	})
	return env
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", string(r.Body), err)
	}
}

func (r response) code(t *testing.T) string {
	t.Helper()
	var payload struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(r.Body, &payload)
	return payload.Code
}

func (e *testEnv) do(method, path string, payload any, token string) response {
	e.t.Helper()
	var body *bytes.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatal(err)
		}
		body = bytes.NewReader(raw)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, body)
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", rand.Intn(255), rand.Intn(255), rand.Intn(255))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *testEnv) upload(method, path string, data []byte, token string) response {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "photo.png")
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

func (e *testEnv) expect(r response, status int) {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, string(r.Body))
	}
}

func testImage(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type authResult struct {
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

func (e *testEnv) register(prefix string) member {
	e.t.Helper()
	email := fmt.Sprintf("%s_%d_%d@hearth.test", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
	e.emails = append(e.emails, email)
	r := e.do(http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, "")
	e.expect(r, http.StatusCreated)
	var auth authResult
	r.decode(e.t, &auth)
	return member{ID: auth.User.ID, Email: email, Token: auth.AccessToken, Refresh: auth.RefreshToken}
}

type persona struct {
	Name         string
	Gender       string
	Birthdate    string
	InterestedIn []string
	MinAge       int
	MaxAge       int
	Distance     int
	LatOffset    float64
	Interests    []string
}

func (e *testEnv) onboard(m member, p persona) {
	e.t.Helper()
	if p.MinAge == 0 {
		p.MinAge, p.MaxAge = 18, 99
	}
	if p.Distance == 0 {
		p.Distance = 50
	}
	interests := p.Interests
	if interests == nil {
		interests = []string{"hiking", "coffee"}
	}
	e.expect(e.do(http.MethodPatch, "/profile", map[string]any{
		"firstName": p.Name, "birthdate": p.Birthdate, "gender": p.Gender,
		"bio": "Bio de test", "interests": interests,
	}, m.Token), http.StatusOK)
	e.expect(e.do(http.MethodPut, "/profile/preferences", map[string]any{
		"interestedIn": p.InterestedIn, "minAge": p.MinAge, "maxAge": p.MaxAge, "maxDistanceKm": p.Distance,
	}, m.Token), http.StatusOK)
	e.expect(e.do(http.MethodPut, "/profile/location", map[string]any{
		"latitude": e.baseLat + p.LatOffset, "longitude": e.baseLon, "city": "Testville",
	}, m.Token), http.StatusOK)
	e.expect(e.upload(http.MethodPost, "/profile/photos", testImage(e.t, 400, 400, color.RGBA{200, 90, 70, 255}), m.Token), http.StatusCreated)
}

type publicProfile struct {
	UserID     string `json:"userId"`
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

func (e *testEnv) discover(m member) []publicProfile {
	e.t.Helper()
	r := e.do(http.MethodGet, "/discover?limit=30", nil, m.Token)
	e.expect(r, http.StatusOK)
	var deck struct {
		Profiles []publicProfile `json:"profiles"`
	}
	r.decode(e.t, &deck)
	return deck.Profiles
}

func containsProfile(list []publicProfile, userID string) bool {
	for _, p := range list {
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
		User           struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"user"`
	} `json:"match"`
}

func (e *testEnv) swipe(m member, target, action string) (response, swipeResult) {
	e.t.Helper()
	r := e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "action": action}, m.Token)
	var res swipeResult
	if r.Status == http.StatusOK {
		r.decode(e.t, &res)
	}
	return r, res
}

// pair onboards two compatible people and makes them match.
func (e *testEnv) pair(prefixA, prefixB string) (member, member, string, string) {
	e.t.Helper()
	a, b := e.register(prefixA), e.register(prefixB)
	e.onboard(a, persona{Name: "Alice", Gender: "woman", Birthdate: "1995-04-10", InterestedIn: []string{"man"}})
	e.onboard(b, persona{Name: "Bruno", Gender: "man", Birthdate: "1993-08-21", InterestedIn: []string{"woman"}, LatOffset: 0.05})
	r, _ := e.swipe(a, b.ID, "like")
	e.expect(r, http.StatusOK)
	r, res := e.swipe(b, a.ID, "like")
	e.expect(r, http.StatusOK)
	if !res.Matched || res.Match == nil {
		e.t.Fatalf("expected a match, got %s", string(r.Body))
	}
	return a, b, res.Match.ID, res.Match.ConversationID
}
