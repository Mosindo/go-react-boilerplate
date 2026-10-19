package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/server"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "integration-secret-0123456789abcdef-xyz"

type captureMailer struct {
	mu    sync.Mutex
	codes map[string]string
}

func (m *captureMailer) SendPasswordReset(_ context.Context, to, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[to] = code
	return nil
}

func (m *captureMailer) code(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.codes[email]
}

type env struct {
	t       *testing.T
	pool    *pgxpool.Pool
	srv     *httptest.Server
	hub     *realtime.Hub
	mailer  *captureMailer
	uploads string
}

// newEnv boots the full API against DATABASE_URL_TEST (skipped when unset) and
// starts every test from empty tables.
func newEnv(t *testing.T) *env {
	t.Helper()
	return newEnvWithLimits(t, server.Limits{AuthPerMinute: 600000, AuthBurst: 100000, RecoveryPerMinute: 600000, RecoveryBurst: 100000})
}

func newEnvWithLimits(t *testing.T, limits server.Limits) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard)
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST (or DATABASE_URL) not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, reports CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	dir := t.TempDir()
	store, err := storage.NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mailer := &captureMailer{codes: map[string]string{}}
	hub := realtime.NewHub()
	router, err := server.New(server.Deps{
		Pool: pool, JWTSecret: []byte(testSecret), Store: store, Mailer: mailer, Hub: hub, Limits: limits, BcryptCost: bcrypt.MinCost,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return &env{t: t, pool: pool, srv: srv, hub: hub, mailer: mailer, uploads: dir}
}

type user struct {
	e            *env
	ID           string
	Email        string
	Access       string
	Refresh      string
	Password     string
	Conversation string
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (r resp) code() string {
	var e struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(r.Body, &e)
	return e.Code
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Body: raw, Header: res.Header}
}

func (u *user) do(method, path string, body any) resp {
	u.e.t.Helper()
	return u.e.do(method, path, u.Access, body)
}

func (u *user) expect(status int, method, path string, body any) resp {
	u.e.t.Helper()
	r := u.do(method, path, body)
	if r.Status != status {
		u.e.t.Fatalf("%s %s: expected %d, got %d (%s)", method, path, status, r.Status, r.Body)
	}
	return r
}

var emailCounter int
var emailMu sync.Mutex

func (e *env) register(name string) *user {
	e.t.Helper()
	emailMu.Lock()
	emailCounter++
	n := emailCounter
	emailMu.Unlock()
	u := &user{e: e, Email: fmt.Sprintf("%s%d@test.invalid", strings.ToLower(name), n), Password: "Password123"}
	r := e.do("POST", "/auth/register", "", map[string]string{"email": u.Email, "password": u.Password})
	if r.Status != http.StatusCreated {
		e.t.Fatalf("register: %d %s", r.Status, r.Body)
	}
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.decode(e.t, &out)
	u.ID, u.Access, u.Refresh = out.User.ID, out.AccessToken, out.RefreshToken
	return u
}

type profileSpec struct {
	gender       string
	birth        string
	interestedIn []string
	minAge       int
	maxAge       int
	maxDist      *int
	lat, lng     *float64
	interests    []int
}

func ptr[T any](v T) *T { return &v }

// onboard registers a user and completes profile, preferences and one photo.
func (e *env) onboard(name string, s profileSpec) *user {
	e.t.Helper()
	u := e.register(name)
	body := map[string]any{
		"firstName": name, "birthDate": s.birth, "gender": s.gender, "bio": "Hello, I'm " + name,
		"city": "Lyon", "interests": s.interests,
	}
	if s.lat != nil {
		body["location"] = map[string]float64{"lat": *s.lat, "lng": *s.lng}
	}
	u.expect(200, "PUT", "/profile", body)
	minAge, maxAge := s.minAge, s.maxAge
	if minAge == 0 {
		minAge = 18
	}
	if maxAge == 0 {
		maxAge = 99
	}
	u.expect(200, "PUT", "/preferences", map[string]any{
		"interestedIn": s.interestedIn, "minAge": minAge, "maxAge": maxAge,
		"maxDistanceKm": s.maxDist, "showDistance": true,
	})
	u.upload(testJPEG(400, 500), nil, 201)
	return u
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

func (u *user) upload(data []byte, position *int, want int) resp {
	u.e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("photo", "photo.jpg")
	_, _ = fw.Write(data)
	if position != nil {
		_ = mw.WriteField("position", fmt.Sprint(*position))
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", u.e.srv.URL+"/profile/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+u.Access)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		u.e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		u.e.t.Fatalf("upload: expected %d, got %d (%s)", want, res.StatusCode, raw)
	}
	return resp{Status: res.StatusCode, Body: raw, Header: res.Header}
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID string `json:"id"`
	} `json:"photos"`
}

func (u *user) discover() []card {
	u.e.t.Helper()
	r := u.expect(200, "GET", "/discover?limit=20", nil)
	var out struct {
		Profiles []card `json:"profiles"`
	}
	r.decode(u.e.t, &out)
	return out.Profiles
}

type swipeResp struct {
	Matched   bool `json:"matched"`
	Duplicate bool `json:"duplicate"`
	Match     *struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (u *user) swipe(target *user, action string) swipeResp {
	u.e.t.Helper()
	r := u.expect(200, "POST", "/swipes", map[string]string{"userId": target.ID, "action": action})
	var out swipeResp
	r.decode(u.e.t, &out)
	return out
}

func contains(cards []card, id string) bool {
	for _, c := range cards {
		if c.UserID == id {
			return true
		}
	}
	return false
}

// match makes a and b like each other and records the conversation id on both.
func (e *env) match(a, b *user) string {
	e.t.Helper()
	a.swipe(b, "like")
	res := b.swipe(a, "like")
	if !res.Matched || res.Match == nil {
		e.t.Fatalf("expected a match")
	}
	a.Conversation, b.Conversation = res.Match.ConversationID, res.Match.ConversationID
	return res.Match.ConversationID
}

var (
	woman = profileSpec{gender: "woman", birth: "1994-05-10", interestedIn: []string{"man"}}
	man   = profileSpec{gender: "man", birth: "1992-03-01", interestedIn: []string{"woman"}}
)
