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
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (m *captureMailer) body(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last[to]
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	mailer *captureMailer
	secret []byte
	emails []string
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func newEnv(t *testing.T, rateLimits bool) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("skip integration tests: postgres unavailable (%v)", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	e := &env{t: t, pool: pool, mailer: &captureMailer{}, secret: []byte("integration-secret-0123456789abcdef")}
	router, err := setupRouter(&app{
		dbPool: pool, jwtSecret: e.secret, uploadsDir: t.TempDir(), mailer: e.mailer, rateLimits: rateLimits,
		allowedOrigins: nil,
	})
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	e.router = router
	t.Cleanup(func() {
		for _, email := range e.emails {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
		}
	})
	return e
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader *bytes.Reader
	switch b := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case []byte:
		reader = bytes.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
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
	return resp{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *env) upload(method, path, token, filename string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

type testUser struct {
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

func (e *env) register(prefix string) testUser {
	e.t.Helper()
	email := fmt.Sprintf("%s-%d-%d@example.test", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
	e.emails = append(e.emails, email)
	var body authBody
	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": "Password123"}), http.StatusCreated).json(e.t, &body)
	return testUser{ID: body.User.ID, Email: email, Token: body.AccessToken, Refresh: body.RefreshToken}
}

type profileSpec struct {
	Name         string
	Gender       string
	InterestedIn []string
	Lat, Lng     float64
	Age          int
	MaxDistance  *int
}

func (e *env) makeProfile(u testUser, spec profileSpec) {
	e.t.Helper()
	age := spec.Age
	if age == 0 {
		age = 30
	}
	birth := time.Now().AddDate(-age, 0, -10).Format("2006-01-02")
	e.expect(e.do(http.MethodPut, "/me/profile", u.Token, map[string]any{
		"firstName": spec.Name, "birthDate": birth, "gender": spec.Gender, "bio": "Hello",
		"city": "Testville", "interests": []string{"travel", "music"},
		"latitude": spec.Lat, "longitude": spec.Lng,
	}), http.StatusOK)
	maxDist := spec.MaxDistance
	if maxDist == nil {
		fifty := 50
		maxDist = &fifty
	}
	e.expect(e.do(http.MethodPut, "/me/preferences", u.Token, map[string]any{
		"interestedIn": spec.InterestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": maxDist,
	}), http.StatusOK)
}

func (e *env) addPhoto(u testUser) string {
	e.t.Helper()
	var p struct {
		ID string `json:"id"`
	}
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "p.jpg", jpegBytes(64, 48)), http.StatusCreated).json(e.t, &p)
	return p.ID
}

// completeUser registers a user with a full profile and one photo.
func (e *env) completeUser(prefix string, spec profileSpec) testUser {
	e.t.Helper()
	u := e.register(prefix)
	e.makeProfile(u, spec)
	e.addPhoto(u)
	return u
}

func jpegBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 3), G: uint8(y * 3), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func pngBytes(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, 0, color.NRGBA{R: 255, A: 128})
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// scenarioOrigin returns a random coordinate so tests sharing one database never see each
// other's users within the 50 km discovery radius.
func scenarioOrigin() (float64, float64) {
	return -60 + rand.Float64()*120, -170 + rand.Float64()*340
}

func idsOf(r resp, t *testing.T) []string {
	t.Helper()
	var out struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	r.json(t, &out)
	ids := make([]string, 0, len(out.Profiles))
	for _, p := range out.Profiles {
		ids = append(ids, p.ID)
	}
	return ids
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

var _ = strings.TrimSpace
