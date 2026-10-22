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
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef-integration"

// capturingMailer records the last email so tests can read the reset token.
type capturingMailer struct {
	mu   sync.Mutex
	body string
	to   string
}

func (m *capturingMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.to, m.body = to, body
	return nil
}

func (m *capturingMailer) last() (string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.to, m.body
}

type testEnv struct {
	t         *testing.T
	pool      *pgxpool.Pool
	router    *gin.Engine
	mailer    *capturingMailer
	uploadDir string
}

var resetOnce sync.Once

func newEnv(t *testing.T) *testEnv { return newEnvWithAuthLimit(t, 100000) }

func newEnvWithAuthLimit(t *testing.T, authLimit int) *testEnv {
	t.Helper()
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

	resetOnce.Do(func() {
		if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
			t.Fatalf("reset schema: %v", err)
		}
	})
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	gin.SetMode(gin.TestMode)
	env := &testEnv{t: t, pool: pool, mailer: &capturingMailer{}, uploadDir: t.TempDir()}
	router, err := setupRouter(&app{
		cfg:           config.Config{JWTSecret: testJWTSecret, UploadDir: env.uploadDir, Env: "test"},
		dbPool:        pool,
		mailer:        env.mailer,
		hub:           realtime.NewHub(),
		authRateLimit: authLimit,
	})
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	env.router = router
	return env
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (e *testEnv) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *testEnv) upload(method, path, token string, content []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("photo", "upload.bin")
	_, _ = part.Write(content)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *testEnv) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func pngBytes(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type user struct {
	ID, Email, Token, Refresh string
	Photo                     string
}

type authBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID              string `json:"id"`
		ProfileComplete bool   `json:"profileComplete"`
	} `json:"user"`
}

func (e *testEnv) register(email string) user {
	e.t.Helper()
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "correct-horse-battery"}), http.StatusCreated)
	var a authBody
	r.json(e.t, &a)
	return user{ID: a.User.ID, Email: email, Token: a.AccessToken, Refresh: a.RefreshToken}
}

type profileOpts struct {
	name         string
	gender       string
	interestedIn []string
	age          int
	lat, lon     float64
	minAge       int
	maxAge       int
	maxDistance  int
	noPhoto      bool
}

func birthdateForAge(age int) string {
	return time.Now().AddDate(-age, -1, 0).Format("2006-01-02")
}

// onboard registers a user and completes profile, preferences and photo.
func (e *testEnv) onboard(email string, o profileOpts) user {
	e.t.Helper()
	u := e.register(email)
	if o.minAge == 0 {
		o.minAge = 18
	}
	if o.maxAge == 0 {
		o.maxAge = 99
	}
	if o.maxDistance == 0 {
		o.maxDistance = 100
	}
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{
		"firstName": o.name, "birthdate": birthdateForAge(o.age), "gender": o.gender,
		"bio": "Hello", "city": "Testville", "latitude": o.lat, "longitude": o.lon, "interestIds": []int{1, 2},
	}), http.StatusOK)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{
		"interestedIn": o.interestedIn, "minAge": o.minAge, "maxAge": o.maxAge, "maxDistanceKm": o.maxDistance,
	}), http.StatusOK)
	if !o.noPhoto {
		r := e.expect(e.upload("POST", "/photos", u.Token, pngBytes(400, 500, color.RGBA{200, 80, 60, 255})), http.StatusCreated)
		var p struct {
			ID string `json:"id"`
		}
		r.json(e.t, &p)
		u.Photo = p.ID
	}
	return u
}

type profileCard struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Birthdate  string `json:"birthdate"`
	Latitude   *float64
	Photos     []struct {
		ID string `json:"id"`
	} `json:"photos"`
}

func (e *testEnv) discover(u user) []profileCard {
	e.t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", u.Token, nil), http.StatusOK)
	var body struct {
		Profiles []profileCard `json:"profiles"`
	}
	r.json(e.t, &body)
	return body.Profiles
}

type swipeBody struct {
	Matched       bool `json:"matched"`
	AlreadySwiped bool `json:"alreadySwiped"`
	Match         *struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (e *testEnv) swipe(from user, to, action string) swipeBody {
	e.t.Helper()
	r := e.expect(e.do("POST", "/discover/swipes", from.Token, map[string]string{"userId": to, "action": action}), http.StatusOK)
	var s swipeBody
	r.json(e.t, &s)
	return s
}

var paris = [2]float64{48.8566, 2.3522}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), http.StatusOK)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "correct-horse-battery"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "a@example.org", "password": "short"}), http.StatusBadRequest)

	u := e.register("  Alice@Example.org ")
	if u.Email != "  Alice@Example.org " || u.Token == "" {
		t.Fatal("registration should return tokens")
	}
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "alice@example.org", "password": "correct-horse-battery"}), http.StatusConflict)

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.org", "password": "wrong-password"}), http.StatusUnauthorized)
	login := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "ALICE@example.org", "password": "correct-horse-battery"}), http.StatusOK)
	var a authBody
	login.json(t, &a)

	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", "garbage", nil), http.StatusUnauthorized)
	me := e.expect(e.do("GET", "/me", a.AccessToken, nil), http.StatusOK)
	var meBody struct {
		Email           string `json:"email"`
		ProfileComplete bool   `json:"profileComplete"`
	}
	me.json(t, &meBody)
	if meBody.Email != "alice@example.org" || meBody.ProfileComplete {
		t.Fatalf("unexpected me: %s", me.Body)
	}

	refreshed := e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": a.RefreshToken}), http.StatusOK)
	var r authBody
	refreshed.json(t, &r)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": a.RefreshToken}), http.StatusUnauthorized) // rotated

	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": r.RefreshToken}), http.StatusNoContent)
	e.expect(e.do("GET", "/me", r.AccessToken, nil), http.StatusUnauthorized) // revoked session

	// Account recovery.
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": "nobody@example.org"}), http.StatusAccepted)
	if to, _ := e.mailer.last(); to != "" {
		t.Fatal("no mail must be sent for unknown accounts")
	}
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": "alice@example.org"}), http.StatusAccepted)
	_, body := e.mailer.last()
	token := ""
	for _, line := range strings.Split(body, "\n") {
		if len(line) == 43 && !strings.Contains(line, " ") {
			token = line
		}
	}
	if token == "" {
		t.Fatalf("reset token not found in mail: %q", body)
	}
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"token": "bogus", "password": "new-password-123"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "new-password-123"}), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "another-password-1"}), http.StatusBadRequest) // single use
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.org", "password": "correct-horse-battery"}), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.org", "password": "new-password-123"}), http.StatusOK)
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t)
	u := e.register("val@example.org")

	valid := map[string]any{"firstName": "Léa", "birthdate": birthdateForAge(28), "gender": "woman", "bio": "bio",
		"city": "Paris", "latitude": 48.85661, "longitude": 2.35222, "interestIds": []int{1}}
	with := func(k string, v any) map[string]any {
		m := map[string]any{}
		for key, val := range valid {
			m[key] = val
		}
		m[k] = v
		return m
	}
	e.expect(e.do("PUT", "/profile", u.Token, with("birthdate", birthdateForAge(17))), http.StatusBadRequest)
	e.expect(e.do("PUT", "/profile", u.Token, with("birthdate", "1990-13-45")), http.StatusBadRequest)
	e.expect(e.do("PUT", "/profile", u.Token, with("gender", "robot")), http.StatusBadRequest)
	e.expect(e.do("PUT", "/profile", u.Token, with("firstName", "<script>")), http.StatusBadRequest)
	e.expect(e.do("PUT", "/profile", u.Token, with("firstName", "")), http.StatusBadRequest)
	e.expect(e.do("PUT", "/profile", u.Token, with("bio", strings.Repeat("a", 501))), http.StatusBadRequest)
	e.expect(e.do("PUT", "/profile", u.Token, with("latitude", 123.0)), http.StatusBadRequest)
	e.expect(e.do("PUT", "/profile", u.Token, with("interestIds", []int{9999})), http.StatusBadRequest)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{}, "minAge": 18, "maxAge": 30}), http.StatusBadRequest)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30}), http.StatusBadRequest)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30}), http.StatusBadRequest)

	r := e.expect(e.do("PUT", "/profile", u.Token, valid), http.StatusOK)
	var self struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Age       int     `json:"age"`
	}
	r.json(t, &self)
	if self.Latitude != 48.86 || self.Longitude != 2.35 || self.Age != 28 {
		t.Fatalf("coordinates must be stored coarsely: %+v", self)
	}
}

func TestPhotoUploadSecurity(t *testing.T) {
	e := newEnv(t)
	u := e.register("photos@example.org")

	e.expect(e.upload("POST", "/photos", u.Token, []byte("<?php echo 1; ?>")), http.StatusUnsupportedMediaType)
	e.expect(e.upload("POST", "/photos", u.Token, append([]byte("GIF89a"), make([]byte, 100)...)), http.StatusUnsupportedMediaType)
	e.expect(e.upload("POST", "/photos", u.Token, pngBytes(50, 50, color.White)), http.StatusBadRequest) // too small
	e.expect(e.upload("POST", "/photos", u.Token, append(pngBytes(300, 300, color.White)[:60], 0xff, 0xff)), http.StatusBadRequest)
	e.expect(e.upload("POST", "/photos", u.Token, make([]byte, 9<<20)), http.StatusRequestEntityTooLarge)
	e.expect(e.do("POST", "/photos", u.Token, nil), http.StatusBadRequest) // no multipart body

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.expect(e.upload("POST", "/photos", u.Token, pngBytes(300, 300, color.RGBA{uint8(i * 40), 10, 10, 255})), http.StatusCreated)
		var p struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		}
		r.json(t, &p)
		if p.Position != i {
			t.Fatalf("expected position %d got %d", i, p.Position)
		}
		ids = append(ids, p.ID)
	}
	e.expect(e.upload("POST", "/photos", u.Token, pngBytes(300, 300, color.White)), http.StatusConflict) // limit

	// Reorder and primary.
	rev := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	e.expect(e.do("PUT", "/photos/order", u.Token, map[string]any{"photoIds": rev}), http.StatusOK)
	e.expect(e.do("PUT", "/photos/order", u.Token, map[string]any{"photoIds": ids[:3]}), http.StatusBadRequest)
	r := e.expect(e.do("PUT", "/photos/"+ids[2]+"/primary", u.Token, nil), http.StatusOK)
	var list struct {
		Photos []struct{ ID string } `json:"photos"`
	}
	r.json(t, &list)
	if list.Photos[0].ID != ids[2] || len(list.Photos) != 6 {
		t.Fatalf("primary not first: %s", r.Body)
	}

	// Files are stored as re-encoded JPEG and served with safe headers.
	file := e.do("GET", "/photos/"+ids[2]+"/file", u.Token, nil)
	if file.Status != http.StatusOK || !bytes.HasPrefix(file.Body, []byte{0xff, 0xd8}) {
		t.Fatalf("expected JPEG, status %d", file.Status)
	}
	e.expect(e.do("GET", "/photos/"+ids[2]+"/file", "", nil), http.StatusUnauthorized)

	// Replace and delete (positions close up, files disappear).
	e.expect(e.upload("PUT", "/photos/"+ids[2], u.Token, pngBytes(320, 320, color.RGBA{1, 2, 3, 255})), http.StatusOK)
	e.expect(e.do("DELETE", "/photos/"+ids[2], u.Token, nil), http.StatusNoContent)
	e.expect(e.do("DELETE", "/photos/"+ids[2], u.Token, nil), http.StatusNotFound)

	// The last remaining photo cannot be deleted: a profile always has a picture.
	solo := e.register("solo@example.org")
	var only struct {
		ID string `json:"id"`
	}
	e.expect(e.upload("POST", "/photos", solo.Token, pngBytes(300, 300, color.White)), http.StatusCreated).json(t, &only)
	e.expect(e.do("DELETE", "/photos/"+only.ID, solo.Token, nil), http.StatusConflict)
	e.expect(e.do("DELETE", "/photos/"+ids[0], solo.Token, nil), http.StatusNotFound) // someone else's photo
	var count int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*), COALESCE(max(position), -1) FROM photos WHERE user_id = $1`, u.ID).Scan(&count, new(int))
	if count != 5 {
		t.Fatalf("expected 5 photos, got %d", count)
	}
	var maxPos int
	_ = e.pool.QueryRow(context.Background(), `SELECT max(position) FROM photos WHERE user_id = $1`, u.ID).Scan(&maxPos)
	if maxPos != 4 {
		t.Fatalf("positions must stay contiguous, max=%d", maxPos)
	}
	jpgs := 0
	_ = filepath.Walk(e.uploadDir, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			jpgs++
		}
		return nil
	})
	if jpgs != 12 { // (5 + 1 solo) photos x (full + thumb)
		t.Fatalf("expected 12 stored files, got %d", jpgs)
	}
}

func TestPhotoAccessControl(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("a@example.org", profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
	b := e.onboard("b@example.org", profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}, age: 32, lat: paris[0], lon: paris[1]})
	hidden := e.onboard("h@example.org", profileOpts{name: "Hid", gender: "man", interestedIn: []string{"woman"}, age: 32, lat: paris[0], lon: paris[1]})
	e.expect(e.do("PUT", "/profile", hidden.Token, map[string]any{"firstName": "Hid", "birthdate": birthdateForAge(32), "gender": "man", "latitude": paris[0], "longitude": paris[1], "isDiscoverable": false}), http.StatusOK)

	e.expect(e.do("GET", "/photos/"+b.Photo+"/file", a.Token, nil), http.StatusOK)
	e.expect(e.do("GET", "/photos/"+hidden.Photo+"/file?size=thumb", a.Token, nil), http.StatusNotFound) // hidden profile
	e.expect(e.do("GET", "/profiles/"+hidden.ID, a.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/photos/not-a-uuid/file", a.Token, nil), http.StatusNotFound)

	e.expect(e.do("POST", "/blocks", b.Token, map[string]string{"userId": a.ID}), http.StatusNoContent)
	e.expect(e.do("GET", "/photos/"+b.Photo+"/file", a.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/photos/"+a.Photo+"/file", b.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), http.StatusNotFound)
}

func TestDiscoveryFilters(t *testing.T) {
	e := newEnv(t)
	me := e.onboard("me@example.org", profileOpts{name: "Me", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1], minAge: 25, maxAge: 40, maxDistance: 50})

	ok := e.onboard("ok@example.org", profileOpts{name: "Ok", gender: "man", interestedIn: []string{"woman"}, age: 33, lat: 48.90, lon: 2.40})
	e.onboard("woman@example.org", profileOpts{name: "Wrong", gender: "woman", interestedIn: []string{"woman"}, age: 33, lat: paris[0], lon: paris[1]})
	e.onboard("far@example.org", profileOpts{name: "Far", gender: "man", interestedIn: []string{"woman"}, age: 33, lat: 45.76, lon: 4.83, maxDistance: 500}) // Lyon
	e.onboard("young@example.org", profileOpts{name: "Young", gender: "man", interestedIn: []string{"woman"}, age: 22, lat: paris[0], lon: paris[1]})
	e.onboard("picky@example.org", profileOpts{name: "Picky", gender: "man", interestedIn: []string{"woman"}, age: 33, lat: paris[0], lon: paris[1], minAge: 18, maxAge: 25}) // wants younger
	e.onboard("nophoto@example.org", profileOpts{name: "NoPhoto", gender: "man", interestedIn: []string{"woman"}, age: 33, lat: paris[0], lon: paris[1], noPhoto: true})
	e.onboard("notme@example.org", profileOpts{name: "NotMe", gender: "man", interestedIn: []string{"man"}, age: 33, lat: paris[0], lon: paris[1]}) // not interested in women
	incomplete := e.register("incomplete@example.org")
	_ = incomplete

	cards := e.discover(me)
	if len(cards) != 1 || cards[0].UserID != ok.ID {
		t.Fatalf("expected only the compatible nearby profile, got %d: %+v", len(cards), cards)
	}
	if cards[0].DistanceKm == nil || *cards[0].DistanceKm%5 != 0 || *cards[0].DistanceKm < 5 || *cards[0].DistanceKm > 15 {
		t.Fatalf("distance must be a coarse 5km bucket, got %v", cards[0].DistanceKm)
	}
	raw := e.do("GET", "/discover", me.Token, nil).Body
	for _, forbidden := range []string{"latitude", "longitude", "birthdate", "email"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("discovery leaks %q: %s", forbidden, raw)
		}
	}

	// A user without a finished profile cannot browse.
	e.expect(e.do("GET", "/discover", incomplete.Token, nil), http.StatusConflict)

	// Passing hides the profile for good, and repeating the swipe is a no-op.
	s := e.swipe(me, ok.ID, "pass")
	if s.Matched || s.AlreadySwiped {
		t.Fatalf("unexpected first pass result %+v", s)
	}
	if again := e.swipe(me, ok.ID, "like"); !again.AlreadySwiped || again.Matched {
		t.Fatalf("a processed profile must not change state: %+v", again)
	}
	if got := e.discover(me); len(got) != 0 {
		t.Fatalf("processed profile reappeared: %+v", got)
	}

	// Hiding the profile removes it from discovery.
	e.expect(e.do("PUT", "/profile", ok.Token, map[string]any{"firstName": "Ok", "birthdate": birthdateForAge(33), "gender": "man", "latitude": 48.90, "longitude": 2.40, "isDiscoverable": false}), http.StatusOK)
}

func TestSwipeRules(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("a@example.org", profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
	far := e.onboard("far@example.org", profileOpts{name: "Far", gender: "man", interestedIn: []string{"woman"}, age: 30, lat: 43.30, lon: 5.37}) // Marseille

	for _, bad := range []map[string]string{
		{"userId": a.ID, "action": "like"},
		{"userId": far.ID, "action": "superlike"},
		{"userId": "nope", "action": "like"},
	} {
		if r := e.do("POST", "/discover/swipes", a.Token, bad); r.Status != http.StatusBadRequest && r.Status != http.StatusNotFound {
			t.Fatalf("expected 400/404 for %v, got %d", bad, r.Status)
		}
	}
	// Not shown by discovery (too far) => cannot be liked with a crafted request either.
	e.expect(e.do("POST", "/discover/swipes", a.Token, map[string]string{"userId": far.ID, "action": "like"}), http.StatusNotFound)
	e.expect(e.do("POST", "/discover/swipes", a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "action": "like"}), http.StatusNotFound)
	e.expect(e.do("POST", "/discover/swipes", "", map[string]string{"userId": far.ID, "action": "like"}), http.StatusUnauthorized)
}

func TestMatchAndChatJourney(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("a@example.org", profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
	b := e.onboard("b@example.org", profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}, age: 32, lat: 48.87, lon: 2.36})
	c := e.onboard("c@example.org", profileOpts{name: "Cleo", gender: "woman", interestedIn: []string{"man"}, age: 29, lat: paris[0], lon: paris[1]})

	// Complete flag flips once profile + preferences + photo exist.
	var me struct {
		ProfileComplete bool `json:"profileComplete"`
	}
	e.expect(e.do("GET", "/me", a.Token, nil), http.StatusOK).json(t, &me)
	if !me.ProfileComplete {
		t.Fatal("profile should be complete")
	}

	// A likes B: no match yet, and B is told nothing (no leak of who liked whom).
	first := e.swipe(a, b.ID, "like")
	if first.Matched {
		t.Fatal("one-sided like must not match")
	}
	if e.swipe(a, b.ID, "like"); true {
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM swipes WHERE from_user_id = $1`, a.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("duplicate like stored: %d", n)
		}
	}
	// B sees A (liked-you profiles are prioritised), then likes back => match.
	cards := e.discover(b)
	if len(cards) != 2 || cards[0].UserID != a.ID && cards[1].UserID != a.ID {
		t.Fatalf("B should see A: %+v", cards)
	}
	second := e.swipe(b, a.ID, "like")
	if !second.Matched || second.Match == nil {
		t.Fatal("reciprocal like must create a match")
	}
	convID := second.Match.ConversationID
	if repeat := e.swipe(b, a.ID, "like"); !repeat.AlreadySwiped || repeat.Match == nil || repeat.Match.ID != second.Match.ID {
		t.Fatalf("repeat should return the same match: %+v", repeat)
	}
	var matchCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches`).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match, got %d", matchCount)
	}

	// Matches list for both sides; matched profile stays visible even if hidden later.
	for _, u := range []user{a, b} {
		var list struct {
			Matches []struct {
				ConversationID string `json:"conversationId"`
				HasMessages    bool   `json:"hasMessages"`
			} `json:"matches"`
		}
		e.expect(e.do("GET", "/matches", u.Token, nil), http.StatusOK).json(t, &list)
		if len(list.Matches) != 1 || list.Matches[0].ConversationID != convID || list.Matches[0].HasMessages {
			t.Fatalf("unexpected matches %+v", list)
		}
	}
	// Both received a match notification.
	for _, u := range []user{a, b} {
		var n struct {
			UnreadCount   int                     `json:"unreadCount"`
			Notifications []struct{ Type string } `json:"notifications"`
		}
		e.expect(e.do("GET", "/notifications", u.Token, nil), http.StatusOK).json(t, &n)
		if n.UnreadCount != 1 || n.Notifications[0].Type != "match" {
			t.Fatalf("expected a match notification: %+v", n)
		}
	}

	// Chat: send, list, unread, read.
	msgPath := "/conversations/" + convID + "/messages"
	e.expect(e.do("POST", msgPath, a.Token, map[string]string{"content": "   "}), http.StatusBadRequest)
	e.expect(e.do("POST", msgPath, a.Token, map[string]string{"content": strings.Repeat("x", 2001)}), http.StatusBadRequest)
	sent := e.expect(e.do("POST", msgPath, a.Token, map[string]string{"content": "  Salut Ben !  "}), http.StatusCreated)
	var msg struct {
		ID      string  `json:"id"`
		Content string  `json:"content"`
		ReadAt  *string `json:"readAt"`
	}
	sent.json(t, &msg)
	if msg.Content != "Salut Ben !" || msg.ReadAt != nil {
		t.Fatalf("unexpected message %+v", msg)
	}
	e.expect(e.do("POST", msgPath, a.Token, map[string]string{"content": "Tu es là ?"}), http.StatusCreated)

	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage struct {
				Content string `json:"content"`
			} `json:"lastMessage"`
			Peer struct {
				FirstName string `json:"firstName"`
			} `json:"peer"`
		} `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", b.Token, nil), http.StatusOK).json(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 2 || convs.Conversations[0].Peer.FirstName != "Ana" || convs.Conversations[0].LastMessage.Content != "Tu es là ?" {
		t.Fatalf("unexpected conversations for B: %+v", convs)
	}
	e.expect(e.do("GET", "/conversations", a.Token, nil), http.StatusOK).json(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("sender has nothing unread")
	}

	// Message notifications are coalesced into one per conversation.
	var notif struct {
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do("GET", "/notifications", b.Token, nil), http.StatusOK).json(t, &notif)
	if notif.UnreadCount != 2 { // match + one message notification
		t.Fatalf("expected 2 unread notifications, got %d", notif.UnreadCount)
	}

	// Outsider C cannot touch the conversation, whatever the verb.
	e.expect(e.do("GET", msgPath, c.Token, nil), http.StatusNotFound)
	e.expect(e.do("POST", msgPath, c.Token, map[string]string{"content": "intrus"}), http.StatusNotFound)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", c.Token, nil), http.StatusNotFound)
	e.expect(e.do("DELETE", "/conversations/"+convID, c.Token, nil), http.StatusNotFound)
	e.expect(e.do("DELETE", "/matches/"+second.Match.ID, c.Token, nil), http.StatusNotFound)

	// B reads: A's messages get read state, unread drops to 0.
	read := e.expect(e.do("POST", "/conversations/"+convID+"/read", b.Token, nil), http.StatusOK)
	if !strings.Contains(string(read.Body), `"read":2`) {
		t.Fatalf("unexpected read: %s", read.Body)
	}
	var thread struct {
		Messages []struct {
			Content string  `json:"content"`
			ReadAt  *string `json:"readAt"`
		} `json:"messages"`
	}
	e.expect(e.do("GET", msgPath, a.Token, nil), http.StatusOK).json(t, &thread)
	if len(thread.Messages) != 2 || thread.Messages[0].Content != "Salut Ben !" || thread.Messages[0].ReadAt == nil || thread.Messages[1].ReadAt == nil {
		t.Fatalf("read receipts missing: %+v", thread)
	}

	// Pagination by cursor.
	var page struct {
		Messages   []struct{ Content string } `json:"messages"`
		NextCursor *string                    `json:"nextCursor"`
	}
	e.expect(e.do("GET", msgPath+"?limit=1", a.Token, nil), http.StatusOK).json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Content != "Tu es là ?" || page.NextCursor == nil {
		t.Fatalf("bad page: %+v", page)
	}
	e.expect(e.do("GET", msgPath+"?limit=1&cursor="+*page.NextCursor, a.Token, nil), http.StatusOK).json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Content != "Salut Ben !" {
		t.Fatalf("bad older page: %+v", page)
	}

	// Local deletion hides history for B only, and a new message brings the thread back.
	e.expect(e.do("DELETE", "/conversations/"+convID, b.Token, nil), http.StatusNoContent)
	e.expect(e.do("GET", "/conversations", b.Token, nil), http.StatusOK).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("hidden conversation should disappear for B")
	}
	e.expect(e.do("GET", "/conversations", a.Token, nil), http.StatusOK).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("A must keep the conversation")
	}
	e.expect(e.do("POST", msgPath, a.Token, map[string]string{"content": "Coucou"}), http.StatusCreated)
	e.expect(e.do("GET", msgPath, b.Token, nil), http.StatusOK).json(t, &thread)
	if len(thread.Messages) != 1 || thread.Messages[0].Content != "Coucou" {
		t.Fatalf("B should only see messages after clearing: %+v", thread)
	}

	// Unmatch removes the conversation for everyone.
	e.expect(e.do("DELETE", "/matches/"+second.Match.ID, b.Token, nil), http.StatusNoContent)
	e.expect(e.do("GET", msgPath, a.Token, nil), http.StatusNotFound)
	e.expect(e.do("POST", msgPath, b.Token, map[string]string{"content": "x"}), http.StatusNotFound)
}

func TestBlockAndReport(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("a@example.org", profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
	b := e.onboard("b@example.org", profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}, age: 32, lat: paris[0], lon: paris[1]})
	e.swipe(a, b.ID, "like")
	m := e.swipe(b, a.ID, "like")
	if !m.Matched {
		t.Fatal("expected match")
	}
	msgPath := "/conversations/" + m.Match.ConversationID + "/messages"
	e.expect(e.do("POST", msgPath, a.Token, map[string]string{"content": "hi"}), http.StatusCreated)

	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": a.ID}), http.StatusBadRequest)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}), http.StatusNotFound)
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "nonsense"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "rude", "alsoBlock": true}), http.StatusCreated)
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "spam"}), http.StatusCreated) // duplicate open report is a no-op

	var reports int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("expected one open report, got %d", reports)
	}

	// The block dissolves the match and conversation and hides both users from each other.
	e.expect(e.do("GET", msgPath, a.Token, nil), http.StatusNotFound)
	e.expect(e.do("POST", msgPath, b.Token, map[string]string{"content": "hello?"}), http.StatusNotFound)
	if len(e.discover(a)) != 0 || len(e.discover(b)) != 0 {
		t.Fatal("blocked users must not see each other")
	}
	if again := e.swipe(b, a.ID, "like"); again.Matched {
		t.Fatal("a block must not be bypassed by swiping again")
	}
	c := e.onboard("c@example.org", profileOpts{name: "Cleo", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
	e.expect(e.do("POST", "/blocks", c.Token, map[string]string{"userId": b.ID}), http.StatusNoContent)
	e.expect(e.do("POST", "/discover/swipes", b.Token, map[string]string{"userId": c.ID, "action": "like"}), http.StatusNotFound)

	var blocked struct {
		Blocks []struct{ UserID string } `json:"blocks"`
	}
	e.expect(e.do("GET", "/blocks", a.Token, nil), http.StatusOK).json(t, &blocked)
	if len(blocked.Blocks) != 1 || blocked.Blocks[0].UserID != b.ID {
		t.Fatalf("unexpected blocks %+v", blocked)
	}
	e.expect(e.do("DELETE", "/blocks/"+b.ID, a.Token, nil), http.StatusNoContent)
	e.expect(e.do("GET", "/blocks", a.Token, nil), http.StatusOK).json(t, &blocked)
	if len(blocked.Blocks) != 0 {
		t.Fatal("unblock failed")
	}
}

func TestAccountDeletion(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("a@example.org", profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
	b := e.onboard("b@example.org", profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}, age: 32, lat: paris[0], lon: paris[1]})
	e.swipe(a, b.ID, "like")
	m := e.swipe(b, a.ID, "like")
	e.expect(e.do("POST", "/conversations/"+m.Match.ConversationID+"/messages", a.Token, map[string]string{"content": "hi"}), http.StatusCreated)
	e.expect(e.do("POST", "/reports", b.Token, map[string]any{"userId": a.ID, "reason": "spam"}), http.StatusCreated)

	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong-password"}), http.StatusUnauthorized)
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "correct-horse-battery"}), http.StatusNoContent)

	e.expect(e.do("GET", "/me", a.Token, nil), http.StatusUnauthorized) // token dies with the account
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "a@example.org", "password": "correct-horse-battery"}), http.StatusUnauthorized)
	for table, col := range map[string]string{
		"profiles": "user_id", "preferences": "user_id", "photos": "user_id", "user_interests": "user_id",
		"sessions": "user_id", "notifications": "user_id", "reports": "reported_id", "messages": "sender_id",
	} {
		var n int
		_ = e.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, table, col), a.ID).Scan(&n)
		if n != 0 {
			t.Fatalf("%s still holds %d rows of the deleted user", table, n)
		}
	}
	var matches, convs int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches`).Scan(&matches)
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM conversations`).Scan(&convs)
	if matches != 0 || convs != 0 {
		t.Fatalf("match/conversation must be deleted: %d %d", matches, convs)
	}
	files := 0
	_ = filepath.Walk(e.uploadDir, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			files++
		}
		return nil
	})
	// Only Ben's photo (full + thumb) remains.
	if files != 2 {
		t.Fatalf("photo files of the deleted user must be removed, %d left", files)
	}
	if len(e.discover(b)) != 0 {
		t.Fatal("deleted user must not be discoverable")
	}
}

func TestConcurrentMutualLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	const pairs = 12
	type pair struct{ a, b user }
	var ps []pair
	for i := 0; i < pairs; i++ {
		a := e.onboard(fmt.Sprintf("a%d@example.org", i), profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
		b := e.onboard(fmt.Sprintf("b%d@example.org", i), profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}, age: 30, lat: paris[0], lon: paris[1]})
		ps = append(ps, pair{a, b})
	}
	// Everybody is compatible with everybody; each pair likes each other at the same instant.
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, p := range ps {
		p := p
		wg.Add(2)
		go func() { defer wg.Done(); <-start; e.swipe(p.a, p.b.ID, "like") }()
		go func() { defer wg.Done(); <-start; e.swipe(p.b, p.a.ID, "like") }()
	}
	close(start)
	wg.Wait()

	for _, p := range ps {
		var n int
		_ = e.pool.QueryRow(context.Background(), `
			SELECT count(*) FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		`, p.a.ID, p.b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("pair %s/%s: expected exactly 1 match, got %d", p.a.ID, p.b.ID, n)
		}
	}
}

func TestWebSocketRealtime(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("a@example.org", profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}, age: 30, lat: paris[0], lon: paris[1]})
	b := e.onboard("b@example.org", profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}, age: 32, lat: paris[0], lon: paris[1]})

	srv := httptest.NewServer(e.router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	// Unauthenticated sockets are closed.
	bad, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "garbage"})
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("expected the socket to be closed for an invalid token")
	}

	connect := func(u user) *websocket.Conn {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		_ = c.WriteJSON(map[string]string{"type": "auth", "token": u.Token})
		var ready struct{ Type string }
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := c.ReadJSON(&ready); err != nil || ready.Type != "ready" {
			t.Fatalf("expected ready, got %v %v", ready, err)
		}
		return c
	}
	wa, wb := connect(a), connect(b)
	defer wa.Close()
	defer wb.Close()

	next := func(c *websocket.Conn, want string) json.RawMessage {
		t.Helper()
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			var ev struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := c.ReadJSON(&ev); err != nil {
				t.Fatalf("waiting for %s: %v", want, err)
			}
			if ev.Type == want {
				return ev.Data
			}
		}
	}

	e.swipe(a, b.ID, "like")
	m := e.swipe(b, a.ID, "like")
	next(wa, "match.new")
	next(wb, "match.new")

	e.expect(e.do("POST", "/conversations/"+m.Match.ConversationID+"/messages", a.Token, map[string]string{"content": "temps réel"}), http.StatusCreated)
	var got struct{ Content string }
	if err := json.Unmarshal(next(wb, "message.new"), &got); err != nil || got.Content != "temps réel" {
		t.Fatalf("B should receive the message live: %v %v", got, err)
	}
	next(wb, "notification.new")

	e.expect(e.do("POST", "/conversations/"+m.Match.ConversationID+"/read", b.Token, nil), http.StatusOK)
	next(wa, "message.read")

	// Deleting the account ends the live connection.
	e.expect(e.do("DELETE", "/me", b.Token, map[string]string{"password": "correct-horse-battery"}), http.StatusNoContent)
	_ = wb.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := wb.ReadMessage(); err != nil {
			break
		}
	}
}

func TestRateLimitOnAuth(t *testing.T) {
	e := newEnvWithAuthLimit(t, 20)
	limited := false
	for i := 0; i < 40; i++ {
		r := e.do("POST", "/auth/login", "", map[string]string{"email": fmt.Sprintf("x%d@example.org", i), "password": "whatever-1234"})
		if r.Status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("expected the auth endpoints to be rate limited per IP")
	}
}
