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
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "integration-secret-0123456789abcdef-xyz"

type testEnv struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	app    *app
	emails []string
	orgID  string
}

type testResponse struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r testResponse) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode response: %v body=%s", err, string(r.Body))
	}
}

func generousLimits() limits {
	l := func() *ratelimit.Limiter { return ratelimit.New(10_000, time.Minute, 10_000) }
	return limits{anonymous: l(), swipes: l(), messages: l(), uploads: l(), reports: l(), tickets: l(), sensitive: l(), location: l()}
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWithLimits(t, generousLimits())
}

func newTestEnvWithLimits(t *testing.T, l limits) *testEnv {
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
	storage, err := media.NewDiskStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &app{
		dbPool:    pool,
		jwtSecret: []byte(testJWTSecret),
		storage:   storage,
		signer:    media.NewSigner(testJWTSecret),
		mailer:    mailer.DevMailer{},
		hub:       realtime.NewHub(),
		limits:    l,
	}
	env := &testEnv{t: t, router: setupRouter(a), pool: pool, app: a}
	// Each environment lives in its own organisation so tests never see each
	// other's members (or demo data) in discovery.
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO organizations (slug, name) VALUES ($1, 'Integration test') RETURNING id
	`, fmt.Sprintf("test-%d-%d", time.Now().UnixNano(), emailCounter.Add(1))).Scan(&env.orgID); err != nil {
		t.Fatalf("create test organization: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1) OR organization_id = $2`, env.emails, env.orgID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, env.orgID)
		pool.Close()
	})
	return env
}

func (e *testEnv) do(method, path string, payload any, token string) testResponse {
	e.t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			e.t.Fatalf("marshal payload: %v", err)
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
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *testEnv) expect(resp testResponse, status int, context string) {
	e.t.Helper()
	if resp.Status != status {
		e.t.Fatalf("%s: expected %d, got %d body=%s", context, status, resp.Status, string(resp.Body))
	}
}

var emailCounter atomic.Int64

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d_%d@lueur.test", prefix, time.Now().UnixNano(), emailCounter.Add(1))
}

type session struct {
	Token        string
	RefreshToken string
	UserID       string
	Email        string
}

type authPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

const testPassword = "Password123"

func (e *testEnv) register(prefix string) session {
	e.t.Helper()
	email := uniqueEmail(prefix)
	e.emails = append(e.emails, email)
	resp := e.do(http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, "")
	e.expect(resp, http.StatusCreated, "register")
	var p authPayload
	resp.decode(e.t, &p)
	if _, err := e.pool.Exec(context.Background(), `UPDATE users SET organization_id = $2 WHERE id = $1`, p.User.ID, e.orgID); err != nil {
		e.t.Fatalf("move user to test organization: %v", err)
	}
	return session{Token: p.AccessToken, RefreshToken: p.RefreshToken, UserID: p.User.ID, Email: email}
}

type member struct {
	session
	firstName string
}

// onboard creates a complete, discoverable profile.
func (e *testEnv) onboard(prefix, firstName, gender string, interestedIn []string, lat, lng float64) member {
	e.t.Helper()
	s := e.register(prefix)
	e.expect(e.do(http.MethodPut, "/me/profile", map[string]any{
		"firstName": firstName, "birthdate": "1995-04-12", "gender": gender, "bio": "Bonjour", "relationshipGoal": "long_term",
	}, s.Token), http.StatusOK, "profile")
	e.expect(e.do(http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": 100,
	}, s.Token), http.StatusOK, "preferences")
	e.expect(e.do(http.MethodPut, "/me/location", map[string]any{"latitude": lat, "longitude": lng, "city": "Paris"}, s.Token), http.StatusOK, "location")
	e.expect(e.uploadPhoto(s.Token, http.MethodPost, "/me/photos", testImage(e.t, 400, 500)), http.StatusCreated, "photo")
	return member{session: s, firstName: firstName}
}

func testImage(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 180, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *testEnv) uploadPhoto(token, method, path string, data []byte) testResponse {
	e.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		e.t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		e.t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

type cardPayload struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	IsMatch    bool   `json:"isMatch"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
}

func (e *testEnv) discover(token string) []cardPayload {
	e.t.Helper()
	resp := e.do(http.MethodGet, "/discovery?limit=20", nil, token)
	e.expect(resp, http.StatusOK, "discovery")
	var payload struct {
		Profiles []cardPayload `json:"profiles"`
	}
	resp.decode(e.t, &payload)
	return payload.Profiles
}

func containsCard(cards []cardPayload, userID string) bool {
	for _, c := range cards {
		if c.UserID == userID {
			return true
		}
	}
	return false
}

func (e *testEnv) swipe(token, target, decision string) testResponse {
	e.t.Helper()
	return e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "decision": decision}, token)
}

// match makes two members like each other.
func (e *testEnv) match(a, b member) {
	e.t.Helper()
	e.expect(e.swipe(a.Token, b.UserID, "like"), http.StatusOK, "first like")
	e.expect(e.swipe(b.Token, a.UserID, "like"), http.StatusCreated, "mutual like")
}
