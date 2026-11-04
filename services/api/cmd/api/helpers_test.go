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
	"strings"
	"sync"
	"testing"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-test-secret-0123456789abcdef"

var (
	testPool  *pgxpool.Pool
	testStore *media.Store
	testMail  = &captureMailer{}
)

type captureMailer struct {
	mu   sync.Mutex
	last map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.last == nil {
		m.last = map[string]string{}
	}
	m.last[to] = body
	return nil
}

func (m *captureMailer) codeFor(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	body := m.last[to]
	for _, f := range strings.Fields(body) {
		if len(f) == 8 && strings.Trim(f, "0123456789") == "" {
			return f
		}
	}
	return ""
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		fmt.Println("TEST_DATABASE_URL not set: skipping database integration tests")
		os.Exit(0)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		fmt.Println("reset schema:", err)
		os.Exit(1)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		fmt.Println("migrate:", err)
		os.Exit(1)
	}
	// Running twice must be a no-op.
	if err := db.RunMigrations(ctx, pool); err != nil {
		fmt.Println("migrate twice:", err)
		os.Exit(1)
	}
	dir, _ := os.MkdirTemp("", "uploads")
	testStore, _ = media.NewStore(dir)
	testPool = pool
	code := m.Run()
	pool.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func newRouter() *gin.Engine {
	return setupRouter(&app{dbPool: testPool, jwtSecret: []byte(testSecret), store: testStore, mailer: testMail})
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func do(t *testing.T, r http.Handler, method, path string, body any, token string) resp {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return resp{rec.Code, rec.Body.Bytes(), rec.Header()}
}

func pngImage(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func upload(t *testing.T, r http.Handler, method, path, token string, data []byte) resp {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("photo", "x.png")
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return resp{rec.Code, rec.Body.Bytes(), rec.Header()}
}

type testUser struct {
	ID, Email, Token, Refresh string
}

var emailSeq int
var emailMu sync.Mutex

func uniqueEmail() string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailSeq++
	return fmt.Sprintf("user%d@test.example", emailSeq)
}

func register(t *testing.T, r http.Handler) testUser {
	t.Helper()
	email := uniqueEmail()
	res := do(t, r, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": "Passw0rdTest"}, "")
	if res.Status != http.StatusCreated {
		t.Fatalf("register: %d %s", res.Status, res.Body)
	}
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct{ ID string }
	}
	res.json(t, &out)
	return testUser{ID: out.User.ID, Email: email, Token: out.AccessToken, Refresh: out.RefreshToken}
}

// onboard registers a user with a complete profile: name, gender, preferences and one photo.
func onboard(t *testing.T, r http.Handler, name, gender string, interestedIn []string, lat, lng float64) testUser {
	t.Helper()
	u := register(t, r)
	res := do(t, r, http.MethodPut, "/profile/me", map[string]any{
		"firstName": name, "birthDate": "1995-04-12", "gender": gender, "bio": "Hello", "interests": []string{"travel", "music"},
	}, u.Token)
	if res.Status != http.StatusOK {
		t.Fatalf("profile: %d %s", res.Status, res.Body)
	}
	res = do(t, r, http.MethodPut, "/profile/preferences", map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": 100,
	}, u.Token)
	if res.Status != http.StatusOK {
		t.Fatalf("prefs: %d %s", res.Status, res.Body)
	}
	if lat != 0 {
		res = do(t, r, http.MethodPut, "/profile/location", map[string]any{"latitude": lat, "longitude": lng, "city": "Paris"}, u.Token)
		if res.Status != http.StatusOK {
			t.Fatalf("location: %d %s", res.Status, res.Body)
		}
	}
	res = upload(t, r, http.MethodPost, "/photos", u.Token, pngImage(400, 500))
	if res.Status != http.StatusCreated {
		t.Fatalf("photo: %d %s", res.Status, res.Body)
	}
	return u
}

type deckProfile struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID, URL string
	} `json:"photos"`
}

func deck(t *testing.T, r http.Handler, u testUser) []deckProfile {
	t.Helper()
	res := do(t, r, http.MethodGet, "/discover?limit=20", nil, u.Token)
	if res.Status != http.StatusOK {
		t.Fatalf("discover: %d %s", res.Status, res.Body)
	}
	var out struct{ Profiles []deckProfile }
	res.json(t, &out)
	return out.Profiles
}

func hasProfile(list []deckProfile, id string) bool {
	for _, p := range list {
		if p.UserID == id {
			return true
		}
	}
	return false
}

type swipeResult struct {
	Result           string `json:"result"`
	AlreadyProcessed bool   `json:"alreadyProcessed"`
	Match            *struct {
		MatchID        string `json:"matchId"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func swipe(t *testing.T, r http.Handler, from testUser, to, action string) (int, swipeResult) {
	t.Helper()
	res := do(t, r, http.MethodPost, "/swipes", map[string]string{"userId": to, "action": action}, from.Token)
	var out swipeResult
	if res.Status == http.StatusOK {
		res.json(t, &out)
	}
	return res.Status, out
}
