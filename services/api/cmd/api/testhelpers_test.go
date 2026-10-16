package main

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
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"example.com/api/internal/platform/db/dbtest"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "Password123"

type testEnv struct {
	t        *testing.T
	pool     *pgxpool.Pool
	router   *gin.Engine
	store    *storage.LocalStore
	storeDir string
	mail     *mailRecorder
	next     atomic.Int32
}

type mailRecorder struct {
	mu     sync.Mutex
	bodies []string
}

func (m *mailRecorder) Send(_ context.Context, _, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bodies = append(m.bodies, body)
	return nil
}

type user struct {
	ID      string
	Email   string
	Token   string
	Refresh string
	addr    string
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) JSON(t *testing.T, out any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, out); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (r response) Map(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	r.JSON(t, &m)
	return m
}

func gradientJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func multipartBody(t *testing.T, field, filename, contentType string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, filename))
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	w.Close()
	return &buf, w.FormDataContentType()
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pool := dbtest.NewPool(t)
	storeDir := t.TempDir()
	store, err := storage.NewLocalStore(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	rec := &mailRecorder{}
	env := &testEnv{t: t, pool: pool, store: store, storeDir: storeDir, mail: rec}
	env.router = setupRouter(&app{
		dbPool:    pool,
		jwtSecret: []byte("integration-secret-0123456789abcdef"),
		appName:   "Test",
		store:     store,
		mail:      rec,
	})
	return env
}

func (e *testEnv) do(method, path, token string, body any, addr string) response {
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
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if addr != "" {
		req.RemoteAddr = addr
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (e *testEnv) upload(token, addr string, data []byte, contentType string) response {
	e.t.Helper()
	body, ct := multipartBody(e.t, "photo", "p.jpg", contentType, data)
	req := httptest.NewRequest(http.MethodPost, "/me/photos", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	if addr != "" {
		req.RemoteAddr = addr
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

// register creates an account; every user gets a distinct client IP so per-IP rate
// limits never interfere with unrelated scenarios.
func (e *testEnv) register(name string) *user {
	e.t.Helper()
	n := e.next.Add(1)
	u := &user{Email: fmt.Sprintf("%s%d@example.test", strings.ToLower(name), n), addr: fmt.Sprintf("10.1.%d.%d:4000", n/250, n%250+1)}
	res := e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": u.Email, "password": testPassword}, u.addr)
	if res.Status != http.StatusCreated {
		e.t.Fatalf("register %s: %d %s", name, res.Status, res.Body)
	}
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	res.JSON(e.t, &out)
	u.Token, u.Refresh, u.ID = out.AccessToken, out.RefreshToken, out.User.ID
	return u
}

type profileSpec struct {
	Name         string
	BirthDate    string
	Gender       string
	InterestedIn []string
	MinAge       int
	MaxAge       int
	MaxKm        int
	Lat, Lon     float64
	Interests    []string
	Photos       int
}

// onboard performs the full onboarding: profile, preferences and photos.
func (e *testEnv) onboard(u *user, s profileSpec) {
	e.t.Helper()
	if s.MinAge == 0 {
		s.MinAge = 18
	}
	if s.MaxAge == 0 {
		s.MaxAge = 99
	}
	if s.MaxKm == 0 {
		s.MaxKm = 50
	}
	if s.Lat == 0 && s.Lon == 0 {
		s.Lat, s.Lon = 48.8566, 2.3522
	}
	res := e.do(http.MethodPut, "/me/profile", u.Token, map[string]any{
		"firstName": s.Name, "birthDate": s.BirthDate, "gender": s.Gender, "bio": "Bonjour",
		"city": "Paris", "latitude": s.Lat, "longitude": s.Lon, "interests": s.Interests,
	}, u.addr)
	if res.Status != http.StatusOK {
		e.t.Fatalf("save profile %s: %d %s", s.Name, res.Status, res.Body)
	}
	res = e.do(http.MethodPut, "/me/preferences", u.Token, map[string]any{
		"interestedIn": s.InterestedIn, "minAge": s.MinAge, "maxAge": s.MaxAge, "maxDistanceKm": s.MaxKm,
	}, u.addr)
	if res.Status != http.StatusOK {
		e.t.Fatalf("save preferences %s: %d %s", s.Name, res.Status, res.Body)
	}
	photos := s.Photos
	if photos == 0 {
		photos = 1
	}
	if photos < 0 {
		photos = 0
	}
	for i := 0; i < photos; i++ {
		if res := e.upload(u.Token, u.addr, gradientJPEG(e.t, 320, 400), "image/jpeg"); res.Status != http.StatusCreated {
			e.t.Fatalf("upload photo %s: %d %s", s.Name, res.Status, res.Body)
		}
	}
}

func (e *testEnv) discoverIDs(u *user) []string {
	e.t.Helper()
	res := e.do(http.MethodGet, "/discover?limit=20", u.Token, nil, u.addr)
	if res.Status != http.StatusOK {
		e.t.Fatalf("discover: %d %s", res.Status, res.Body)
	}
	var out struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	res.JSON(e.t, &out)
	ids := make([]string, len(out.Profiles))
	for i, p := range out.Profiles {
		ids[i] = p.ID
	}
	return ids
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func (e *testEnv) swipe(u *user, target *user, action string) map[string]any {
	e.t.Helper()
	res := e.do(http.MethodPost, "/swipes", u.Token, map[string]string{"targetId": target.ID, "action": action}, u.addr)
	if res.Status != http.StatusOK {
		e.t.Fatalf("swipe: %d %s", res.Status, res.Body)
	}
	return res.Map(e.t)
}
