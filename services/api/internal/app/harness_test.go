package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "Passw0rd-test"

var userCounter atomic.Int64

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
	pool   *pgxpool.Pool
	t      *testing.T
	router *gin.Engine
	files  *storage.LocalStore
	dir    string
	mail   *captureMailer
}

func newEnv(t *testing.T, mutate ...func(*app.Deps)) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pool := testutil.NewTestDB(t)
	dir := t.TempDir()
	files, err := storage.NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	deps := app.Deps{
		DB:        pool,
		JWTSecret: []byte("integration-secret-0123456789abcdef0123"),
		Files:     files,
		Mailer:    mail,
		AppName:   "Lumen",
		Limits: app.Limits{
			AuthPerMinute: 1000, AuthBurst: 1000, SwipePerMinute: 1000, SwipeBurst: 1000,
			SendPerMinute: 1000, SendBurst: 1000, UploadPerMinute: 1000, UploadBurst: 1000,
			ReportPerMinute: 1000, ReportBurst: 1000,
		},
	}
	for _, m := range mutate {
		m(&deps)
	}
	router, err := app.NewRouter(deps)
	if err != nil {
		t.Fatal(err)
	}
	return &env{pool: pool, t: t, router: router, files: files, dir: dir, mail: mail}
}

type result struct {
	Status int
	Body   map[string]any
	Raw    []byte
}

func (r result) str(path ...string) string {
	var cur any = r.Body
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[p]
	}
	s, _ := cur.(string)
	return s
}

func (r result) list(key string) []any {
	l, _ := r.Body[key].([]any)
	return l
}

func (e *env) do(method, path, token string, body any) result {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.serve(req)
}

func (e *env) serve(req *http.Request) result {
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	res := result{Status: w.Code, Raw: w.Body.Bytes()}
	_ = json.Unmarshal(res.Raw, &res.Body)
	return res
}

func (e *env) upload(token string, data []byte, filename string) result {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", filename)
	_, _ = part.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.serve(req)
}

func testJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

type user struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

type profileSpec struct {
	Name         string
	Gender       string
	BirthDate    string
	InterestedIn []string
	MinAge       int
	MaxAge       int
	MaxDistance  int
	Lat, Lng     float64
	Interests    []string
}

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s%d@example.org", prefix, userCounter.Add(1))
	res := e.do("POST", "/auth/register", "", map[string]any{"email": email, "password": testPassword})
	if res.Status != http.StatusCreated {
		e.t.Fatalf("register: %d %s", res.Status, res.Raw)
	}
	return user{ID: res.str("user", "id"), Email: email, Token: res.str("accessToken"), Refresh: res.str("refreshToken")}
}

// onboard registers a user and completes profile, preferences, photo and location.
func (e *env) onboard(prefix string, spec profileSpec) user {
	e.t.Helper()
	u := e.register(prefix)
	if spec.BirthDate == "" {
		spec.BirthDate = "1994-06-15"
	}
	if spec.MinAge == 0 {
		spec.MinAge, spec.MaxAge = 18, 99
	}
	if spec.MaxDistance == 0 {
		spec.MaxDistance = 50
	}
	if len(spec.InterestedIn) == 0 {
		spec.InterestedIn = []string{"man", "woman", "non_binary"}
	}
	if spec.Interests == nil {
		spec.Interests = []string{"hiking", "music"}
	}
	if spec.Lat == 0 {
		spec.Lat, spec.Lng = 48.8566, 2.3522
	}
	steps := []struct {
		method, path string
		body         any
	}{
		{"PUT", "/profile/me", map[string]any{"firstName": spec.Name, "birthDate": spec.BirthDate, "gender": spec.Gender, "bio": "Hello there", "city": "Paris", "interests": spec.Interests}},
		{"PUT", "/profile/preferences", map[string]any{"interestedIn": spec.InterestedIn, "minAge": spec.MinAge, "maxAge": spec.MaxAge, "maxDistanceKm": spec.MaxDistance, "discoverable": true, "showDistance": true}},
		{"PUT", "/profile/location", map[string]any{"latitude": spec.Lat, "longitude": spec.Lng}},
	}
	for _, s := range steps {
		res := e.do(s.method, s.path, u.Token, s.body)
		if res.Status >= 300 {
			e.t.Fatalf("%s %s: %d %s", s.method, s.path, res.Status, res.Raw)
		}
	}
	if res := e.upload(u.Token, testJPEG(400, 300), "me.jpg"); res.Status != http.StatusCreated {
		e.t.Fatalf("upload: %d %s", res.Status, res.Raw)
	}
	return u
}

func (e *env) dbPool() *pgxpool.Pool { return e.pool }

func contains(s, sub string) bool { return strings.Contains(s, sub) }
