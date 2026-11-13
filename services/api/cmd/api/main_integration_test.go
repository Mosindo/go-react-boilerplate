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
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "Password123"

// captureMailer records outgoing mails so tests can read reset codes.
type captureMailer struct {
	mu   sync.Mutex
	last map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
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
	return regexp.MustCompile(`\d{6}`).FindString(m.last[to])
}

type testEnv struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mailer *captureMailer
	hub    *realtime.Hub
	emails []string
	// Each test run lives in its own random area so leftover rows from
	// other runs never interfere with discovery assertions.
	lat, lon float64
}

type testResponse struct {
	Status int
	Body   []byte
}

func (r testResponse) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func setupEnv(t *testing.T) *testEnv {
	t.Helper()
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
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	hub := realtime.NewHub()
	router, err := setupRouter(&app{
		cfg: config.Config{
			Env:       "test",
			JWTSecret: "integration-secret-0123456789abcdef-xyz",
		},
		dbPool: pool,
		store:  store,
		hub:    hub,
		mailer: mail,
	})
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	env := &testEnv{
		t: t, router: router, pool: pool, mailer: mail, hub: hub,
		lat: -50 + rng.Float64()*100,
		lon: -170 + rng.Float64()*340,
	}
	t.Cleanup(func() {
		if len(env.emails) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, env.emails)
		}
		pool.Close()
	})
	return env
}

func randomIP() string {
	return fmt.Sprintf("10.%d.%d.%d:1234", rand.Intn(250), rand.Intn(250), rand.Intn(250)+1)
}

func (e *testEnv) doFrom(ip, method, path string, payload any, token string) testResponse {
	e.t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			e.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.RemoteAddr = ip
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *testEnv) do(method, path string, payload any, token string) testResponse {
	e.t.Helper()
	return e.doFrom(randomIP(), method, path, payload, token)
}

func (e *testEnv) expect(res testResponse, status int, context string) {
	e.t.Helper()
	if res.Status != status {
		e.t.Fatalf("%s: expected %d, got %d (%s)", context, status, res.Status, res.Body)
	}
}

type authResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type member struct {
	ID, Email, Token, Refresh string
}

func (e *testEnv) register(prefix string) member {
	e.t.Helper()
	email := fmt.Sprintf("%s_%d_%d@dating.test", prefix, time.Now().UnixNano(), rand.Intn(1e6))
	e.emails = append(e.emails, email)
	res := e.do(http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, "")
	e.expect(res, http.StatusCreated, "register "+prefix)
	var auth authResponse
	res.decode(e.t, &auth)
	return member{ID: auth.User.ID, Email: email, Token: auth.AccessToken, Refresh: auth.RefreshToken}
}

func pngBytes(t *testing.T, w, h int, tint uint8) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: tint, G: uint8(x), B: uint8(y), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *testEnv) upload(method, path, token string, data []byte) testResponse {
	e.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &body)
	req.RemoteAddr = randomIP()
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes()}
}

type photoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type ownProfile struct {
	FirstName   string          `json:"firstName"`
	Age         *int            `json:"age"`
	IsComplete  bool            `json:"isComplete"`
	Missing     []string        `json:"missing"`
	Photos      []photoResponse `json:"photos"`
	City        string          `json:"city"`
	HasLocation bool            `json:"hasLocation"`
	Interests   []struct {
		Slug string `json:"slug"`
	} `json:"interests"`
}

type publicProfile struct {
	UserID          string   `json:"userId"`
	FirstName       string   `json:"firstName"`
	Age             int      `json:"age"`
	DistanceKm      *int     `json:"distanceKm"`
	SharedInterests []string `json:"sharedInterests"`
	Photos          []struct {
		URL string `json:"url"`
	} `json:"photos"`
}

// completeProfile runs the onboarding API calls for a member.
func (e *testEnv) completeProfile(m member, name, gender string, interestedIn []string, birthdate string, dLat float64) {
	e.t.Helper()
	e.expect(e.do(http.MethodPatch, "/me/profile", map[string]any{
		"firstName": name,
		"birthdate": birthdate,
		"gender":    gender,
		"bio":       "Bio de " + name,
		"interests": []string{"hiking", "coffee", "jazz"},
	}, m.Token), http.StatusOK, "profile "+name)
	e.expect(e.do(http.MethodPatch, "/me/preferences", map[string]any{
		"interestedIn": interestedIn, "ageMin": 18, "ageMax": 60, "maxDistanceKm": 20,
	}, m.Token), http.StatusOK, "preferences "+name)
	e.expect(e.do(http.MethodPut, "/me/location", map[string]any{
		"latitude": e.lat + dLat, "longitude": e.lon, "city": "Testville",
	}, m.Token), http.StatusOK, "location "+name)
	e.expect(e.upload(http.MethodPost, "/me/photos", m.Token, pngBytes(e.t, 400, 500, 10)), http.StatusCreated, "photo "+name)

	var p ownProfile
	res := e.do(http.MethodGet, "/me/profile", nil, m.Token)
	e.expect(res, http.StatusOK, "get profile "+name)
	res.decode(e.t, &p)
	if !p.IsComplete {
		e.t.Fatalf("profile %s should be complete, missing %v", name, p.Missing)
	}
}

func (e *testEnv) discover(m member) []publicProfile {
	e.t.Helper()
	res := e.do(http.MethodGet, "/discovery?limit=20", nil, m.Token)
	e.expect(res, http.StatusOK, "discovery")
	var out struct {
		Profiles []publicProfile `json:"profiles"`
	}
	res.decode(e.t, &out)
	return out.Profiles
}

func containsProfile(list []publicProfile, id string) bool {
	for _, p := range list {
		if p.UserID == id {
			return true
		}
	}
	return false
}

type swipeResult struct {
	Matched bool `json:"matched"`
	Match   *struct {
		ID   string        `json:"id"`
		User publicProfile `json:"user"`
	} `json:"match"`
}

func (e *testEnv) swipe(m member, target, action string, status int) swipeResult {
	e.t.Helper()
	res := e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "action": action}, m.Token)
	e.expect(res, status, "swipe "+action)
	var out swipeResult
	if status == http.StatusOK {
		res.decode(e.t, &out)
	}
	return out
}

func TestHealth(t *testing.T) {
	env := setupEnv(t)
	env.expect(env.do(http.MethodGet, "/health", nil, ""), http.StatusOK, "health")
}

func TestAuthLifecycle(t *testing.T) {
	env := setupEnv(t)
	email := fmt.Sprintf("Auth_%d@Dating.test", time.Now().UnixNano())
	normalized := strings.ToLower(email)
	env.emails = append(env.emails, normalized)

	res := env.do(http.MethodPost, "/auth/register", map[string]string{"email": "  " + email + " ", "password": testPassword}, "")
	env.expect(res, http.StatusCreated, "register")
	var auth authResponse
	res.decode(t, &auth)
	if auth.User.Email != normalized {
		t.Fatalf("email should be normalized, got %q", auth.User.Email)
	}

	env.expect(env.do(http.MethodPost, "/auth/register", map[string]string{"email": normalized, "password": testPassword}, ""), http.StatusConflict, "duplicate")
	env.expect(env.do(http.MethodPost, "/auth/register", map[string]string{"email": "short@dating.test", "password": "123"}, ""), http.StatusBadRequest, "weak password")
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": normalized, "password": "nope-nope"}, ""), http.StatusUnauthorized, "bad login")
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": "unknown@dating.test", "password": testPassword}, ""), http.StatusUnauthorized, "unknown login")
	env.expect(env.do(http.MethodGet, "/me", nil, ""), http.StatusUnauthorized, "me without token")
	env.expect(env.do(http.MethodGet, "/me", nil, auth.AccessToken), http.StatusOK, "me")

	// Refresh rotates; logout revokes the session for the access token too.
	res = env.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": auth.RefreshToken}, "")
	env.expect(res, http.StatusOK, "refresh")
	var refreshed authResponse
	res.decode(t, &refreshed)
	env.expect(env.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": auth.RefreshToken}, ""), http.StatusUnauthorized, "old refresh token")
	env.expect(env.do(http.MethodPost, "/auth/logout", map[string]string{"refreshToken": refreshed.RefreshToken}, ""), http.StatusNoContent, "logout")
	env.expect(env.do(http.MethodGet, "/me", nil, refreshed.AccessToken), http.StatusUnauthorized, "access token after logout")

	// Password recovery.
	res = env.do(http.MethodPost, "/auth/login", map[string]string{"email": normalized, "password": testPassword}, "")
	env.expect(res, http.StatusOK, "login")
	var session authResponse
	res.decode(t, &session)

	env.expect(env.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": "nobody@dating.test"}, ""), http.StatusAccepted, "forgot unknown")
	env.expect(env.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": normalized}, ""), http.StatusAccepted, "forgot")
	code := env.mailer.codeFor(normalized)
	if len(code) != 6 {
		t.Fatalf("expected a 6-digit code in the mail, got %q", code)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": normalized, "code": wrong, "newPassword": "NewPassword456"}, ""), http.StatusBadRequest, "wrong code")
	env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": normalized, "code": code, "newPassword": "NewPassword456"}, ""), http.StatusNoContent, "reset")
	env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": normalized, "code": code, "newPassword": "Another789xx"}, ""), http.StatusBadRequest, "code reuse")
	env.expect(env.do(http.MethodGet, "/me", nil, session.AccessToken), http.StatusUnauthorized, "sessions revoked after reset")
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": normalized, "password": testPassword}, ""), http.StatusUnauthorized, "old password")
	res = env.do(http.MethodPost, "/auth/login", map[string]string{"email": normalized, "password": "NewPassword456"}, "")
	env.expect(res, http.StatusOK, "login new password")
	res.decode(t, &session)

	// Account deletion requires the password and removes everything.
	env.expect(env.do(http.MethodDelete, "/me", map[string]string{"password": "wrong-password"}, session.AccessToken), http.StatusForbidden, "delete wrong password")
	env.expect(env.do(http.MethodDelete, "/me", map[string]string{"password": "NewPassword456"}, session.AccessToken), http.StatusNoContent, "delete")
	env.expect(env.do(http.MethodGet, "/me", nil, session.AccessToken), http.StatusUnauthorized, "me after delete")
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": normalized, "password": "NewPassword456"}, ""), http.StatusUnauthorized, "login after delete")
	var count int
	if err := env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM profiles WHERE user_id = $1`, auth.User.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("profile should be deleted with the account (count=%d, err=%v)", count, err)
	}
}

func TestLoginRateLimit(t *testing.T) {
	env := setupEnv(t)
	ip := randomIP()
	limited := false
	for i := 0; i < 25; i++ {
		res := env.doFrom(ip, http.MethodPost, "/auth/login", map[string]string{"email": "nobody@dating.test", "password": "whatever1"}, "")
		if res.Status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("expected login attempts from one IP to be rate limited")
	}
}

func TestProfileOnboardingAndPhotos(t *testing.T) {
	env := setupEnv(t)
	m := env.register("onboard")

	var p ownProfile
	res := env.do(http.MethodGet, "/me/profile", nil, m.Token)
	env.expect(res, http.StatusOK, "empty profile")
	res.decode(t, &p)
	if p.IsComplete || len(p.Missing) == 0 {
		t.Fatalf("new profile must be incomplete, got %+v", p)
	}

	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	env.expect(env.do(http.MethodPatch, "/me/profile", map[string]any{"birthdate": minor}, m.Token), http.StatusBadRequest, "underage")
	env.expect(env.do(http.MethodPatch, "/me/profile", map[string]any{"firstName": "<script>"}, m.Token), http.StatusBadRequest, "bad name")
	env.expect(env.do(http.MethodPatch, "/me/profile", map[string]any{"gender": "robot"}, m.Token), http.StatusBadRequest, "bad gender")
	env.expect(env.do(http.MethodPatch, "/me/profile", map[string]any{"interests": []string{"not-an-interest"}}, m.Token), http.StatusBadRequest, "unknown interest")
	env.expect(env.do(http.MethodPatch, "/me/preferences", map[string]any{"ageMin": 40, "ageMax": 30}, m.Token), http.StatusBadRequest, "bad age range")
	env.expect(env.do(http.MethodPut, "/me/location", map[string]any{"latitude": 120, "longitude": 2}, m.Token), http.StatusBadRequest, "bad latitude")

	// Exact coordinates are never stored: they are rounded to ~1 km.
	env.expect(env.do(http.MethodPut, "/me/location", map[string]any{"latitude": 48.856613, "longitude": 2.352222, "city": "Paris"}, m.Token), http.StatusOK, "location")
	var lat, lon float64
	if err := env.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, m.ID).Scan(&lat, &lon); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lon != 2.35 {
		t.Fatalf("coordinates must be rounded, got %v,%v", lat, lon)
	}

	// Uploads: type sniffing, size, count, reorder, replace, delete.
	env.expect(env.upload(http.MethodPost, "/me/photos", m.Token, []byte("<html>not an image</html>")), http.StatusBadRequest, "html upload")
	env.expect(env.upload(http.MethodPost, "/me/photos", m.Token, pngBytes(t, 100, 100, 1)), http.StatusBadRequest, "tiny image")

	var photos []photoResponse
	for i := 0; i < 6; i++ {
		res := env.upload(http.MethodPost, "/me/photos", m.Token, pngBytes(t, 400, 400, uint8(i*30)))
		env.expect(res, http.StatusCreated, "upload")
		var ph photoResponse
		res.decode(t, &ph)
		if ph.Position != i {
			t.Fatalf("expected position %d, got %d", i, ph.Position)
		}
		photos = append(photos, ph)
	}
	env.expect(env.upload(http.MethodPost, "/me/photos", m.Token, pngBytes(t, 400, 400, 99)), http.StatusConflict, "7th photo")

	order := []string{photos[5].ID, photos[0].ID, photos[1].ID, photos[2].ID, photos[3].ID, photos[4].ID}
	res = env.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order}, m.Token)
	env.expect(res, http.StatusOK, "reorder")
	var reordered struct {
		Photos []photoResponse `json:"photos"`
	}
	res.decode(t, &reordered)
	if reordered.Photos[0].ID != photos[5].ID {
		t.Fatalf("expected last photo to become the main one")
	}
	env.expect(env.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order[:2]}, m.Token), http.StatusBadRequest, "partial order")

	env.expect(env.upload(http.MethodPut, "/me/photos/"+photos[1].ID, m.Token, pngBytes(t, 500, 400, 200)), http.StatusOK, "replace")
	env.expect(env.do(http.MethodDelete, "/me/photos/"+photos[5].ID, nil, m.Token), http.StatusNoContent, "delete photo")

	res = env.do(http.MethodGet, "/me/photos", nil, m.Token)
	var listed struct {
		Photos []photoResponse `json:"photos"`
	}
	res.decode(t, &listed)
	if len(listed.Photos) != 5 || listed.Photos[0].ID != photos[0].ID || listed.Photos[0].Position != 0 {
		t.Fatalf("positions must stay contiguous after delete: %+v", listed.Photos)
	}

	// Signed media URLs: valid, tampered and unauthenticated access.
	env.expect(env.do(http.MethodGet, listed.Photos[0].URL, nil, ""), http.StatusOK, "signed photo")
	env.expect(env.do(http.MethodGet, strings.Replace(listed.Photos[0].URL, "sig=", "sig=00", 1), nil, ""), http.StatusForbidden, "tampered photo")
	env.expect(env.do(http.MethodGet, "/media/photos/"+listed.Photos[0].ID, nil, ""), http.StatusForbidden, "unsigned photo")

	// Another member cannot touch my photos.
	other := env.register("intruder")
	env.expect(env.do(http.MethodDelete, "/me/photos/"+photos[0].ID, nil, other.Token), http.StatusNotFound, "foreign delete")
}

func TestDiscoveryLikeMatchAndChat(t *testing.T) {
	env := setupEnv(t)
	alice := env.register("alice")
	bob := env.register("bob")
	carl := env.register("carl")
	far := env.register("far")

	env.completeProfile(alice, "Alice", "woman", []string{"man"}, "1995-04-12", 0)
	env.completeProfile(bob, "Bob", "man", []string{"woman"}, "1992-08-30", 0.02)
	env.completeProfile(carl, "Carl", "man", []string{"man"}, "1990-01-01", 0.01)
	env.completeProfile(far, "Fabien", "man", []string{"woman"}, "1991-01-01", 3) // ~330 km away

	feed := env.discover(alice)
	if !containsProfile(feed, bob.ID) {
		t.Fatal("alice should discover bob")
	}
	if containsProfile(feed, carl.ID) {
		t.Fatal("carl is not interested in women and must not be shown to alice")
	}
	if containsProfile(feed, far.ID) {
		t.Fatal("profiles beyond the distance preference must be excluded")
	}
	for _, p := range feed {
		if p.UserID == bob.ID {
			if p.DistanceKm == nil || *p.DistanceKm < 1 || *p.DistanceKm > 5 {
				t.Fatalf("expected an approximate distance for bob, got %v", p.DistanceKm)
			}
			if len(p.SharedInterests) != 3 {
				t.Fatalf("expected 3 shared interests, got %v", p.SharedInterests)
			}
		}
	}

	// Chat is impossible before a match.
	env.expect(env.do(http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": "hi"}, alice.Token), http.StatusNotFound, "message before match")

	if r := env.swipe(alice, bob.ID, "like", http.StatusOK); r.Matched {
		t.Fatal("a single like must not match")
	}
	if r := env.swipe(alice, bob.ID, "like", http.StatusOK); r.Matched {
		t.Fatal("repeated like is idempotent")
	}
	env.swipe(alice, bob.ID, "pass", http.StatusConflict)
	env.swipe(alice, alice.ID, "like", http.StatusBadRequest)
	env.swipe(alice, carl.ID, "pass", http.StatusOK)
	if containsProfile(env.discover(alice), bob.ID) {
		t.Fatal("already-liked profiles must not reappear")
	}

	if !containsProfile(env.discover(bob), alice.ID) {
		t.Fatal("bob should discover alice")
	}
	r := env.swipe(bob, alice.ID, "like", http.StatusOK)
	if !r.Matched || r.Match == nil || r.Match.User.UserID != alice.ID {
		t.Fatalf("mutual like must create a match, got %+v", r)
	}
	var matches int
	if err := env.pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
	`, alice.ID, bob.ID).Scan(&matches); err != nil || matches != 1 {
		t.Fatalf("expected exactly one match row, got %d (%v)", matches, err)
	}

	// Both receive a match notification.
	for _, m := range []member{alice, bob} {
		res := env.do(http.MethodGet, "/notifications", nil, m.Token)
		env.expect(res, http.StatusOK, "notifications")
		var n struct {
			Notifications []struct {
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		res.decode(t, &n)
		if n.UnreadCount < 1 || len(n.Notifications) == 0 || n.Notifications[0].Type != "match" {
			t.Fatalf("expected a match notification, got %s", res.Body)
		}
	}

	// Conversation.
	res := env.do(http.MethodGet, "/chats", nil, alice.Token)
	env.expect(res, http.StatusOK, "chats")
	var chats struct {
		Chats []struct {
			User struct {
				UserID   string `json:"userId"`
				PhotoURL string `json:"photoUrl"`
			} `json:"user"`
			UnreadCount int `json:"unreadCount"`
			LastMessage *struct {
				Content string `json:"content"`
			} `json:"lastMessage"`
		} `json:"chats"`
	}
	res.decode(t, &chats)
	if len(chats.Chats) != 1 || chats.Chats[0].User.UserID != bob.ID || chats.Chats[0].LastMessage != nil || chats.Chats[0].User.PhotoURL == "" {
		t.Fatalf("alice should see one new match without messages: %s", res.Body)
	}

	env.expect(env.do(http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": "   "}, alice.Token), http.StatusBadRequest, "empty message")
	env.expect(env.do(http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": strings.Repeat("a", 2001)}, alice.Token), http.StatusBadRequest, "long message")
	for i := 0; i < 5; i++ {
		env.expect(env.do(http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": fmt.Sprintf("message %d", i)}, alice.Token), http.StatusCreated, "send")
	}

	// Outsiders can neither read nor write this conversation.
	env.expect(env.do(http.MethodGet, "/chats/"+bob.ID+"/messages", nil, carl.Token), http.StatusNotFound, "outsider read")
	env.expect(env.do(http.MethodGet, "/chats/"+alice.ID+"/messages", nil, carl.Token), http.StatusNotFound, "outsider read 2")
	env.expect(env.do(http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "hey"}, carl.Token), http.StatusNotFound, "outsider write")

	res = env.do(http.MethodGet, "/chats", nil, bob.Token)
	res.decode(t, &chats)
	if len(chats.Chats) != 1 || chats.Chats[0].UnreadCount != 5 || chats.Chats[0].LastMessage == nil || chats.Chats[0].LastMessage.Content != "message 4" {
		t.Fatalf("bob should have 5 unread messages: %s", res.Body)
	}

	type page struct {
		Messages []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"messages"`
		HasMore        bool       `json:"hasMore"`
		PeerLastReadAt *time.Time `json:"peerLastReadAt"`
	}
	var p1, p2 page
	res = env.do(http.MethodGet, "/chats/"+alice.ID+"/messages?limit=3", nil, bob.Token)
	env.expect(res, http.StatusOK, "page 1")
	res.decode(t, &p1)
	if len(p1.Messages) != 3 || !p1.HasMore || p1.Messages[2].Content != "message 4" {
		t.Fatalf("unexpected first page: %s", res.Body)
	}
	res = env.do(http.MethodGet, "/chats/"+alice.ID+"/messages?limit=3&before="+p1.Messages[0].ID, nil, bob.Token)
	res.decode(t, &p2)
	if len(p2.Messages) != 2 || p2.HasMore || p2.Messages[0].Content != "message 0" {
		t.Fatalf("unexpected second page: %s", res.Body)
	}

	env.expect(env.do(http.MethodPost, "/chats/"+alice.ID+"/read", nil, bob.Token), http.StatusOK, "mark read")
	res = env.do(http.MethodGet, "/chats/"+bob.ID+"/messages", nil, alice.Token)
	var fromAlice page
	res.decode(t, &fromAlice)
	if fromAlice.PeerLastReadAt == nil {
		t.Fatal("alice should see that bob read the conversation")
	}
	res = env.do(http.MethodGet, "/chats", nil, bob.Token)
	res.decode(t, &chats)
	if chats.Chats[0].UnreadCount != 0 {
		t.Fatalf("unread count should reset after reading, got %d", chats.Chats[0].UnreadCount)
	}

	// Local clear only affects the caller.
	env.expect(env.do(http.MethodDelete, "/chats/"+bob.ID, nil, alice.Token), http.StatusNoContent, "clear")
	res = env.do(http.MethodGet, "/chats/"+bob.ID+"/messages", nil, alice.Token)
	res.decode(t, &fromAlice)
	if len(fromAlice.Messages) != 0 {
		t.Fatal("cleared conversation must be empty for alice")
	}
	res = env.do(http.MethodGet, "/chats/"+alice.ID+"/messages", nil, bob.Token)
	res.decode(t, &p1)
	if len(p1.Messages) != 5 {
		t.Fatal("bob keeps his copy of the conversation")
	}

	// Full profile is visible to a match; unmatch closes the conversation.
	env.expect(env.do(http.MethodGet, "/profiles/"+alice.ID, nil, bob.Token), http.StatusOK, "matched profile")
	env.expect(env.do(http.MethodDelete, "/matches/"+bob.ID, nil, alice.Token), http.StatusNoContent, "unmatch")
	env.expect(env.do(http.MethodDelete, "/matches/"+bob.ID, nil, alice.Token), http.StatusNotFound, "unmatch twice")
	env.expect(env.do(http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "still there?"}, bob.Token), http.StatusNotFound, "message after unmatch")
	res = env.do(http.MethodGet, "/chats", nil, bob.Token)
	res.decode(t, &chats)
	if len(chats.Chats) != 0 {
		t.Fatal("unmatched conversation must disappear")
	}
	if containsProfile(env.discover(bob), alice.ID) {
		t.Fatal("unmatched profiles must not come back in discovery")
	}
}

func TestBlockAndReport(t *testing.T) {
	env := setupEnv(t)
	eve := env.register("eve")
	fred := env.register("fred")
	gus := env.register("gus")
	env.completeProfile(eve, "Eve", "woman", []string{"man"}, "1994-02-02", 0)
	env.completeProfile(fred, "Fred", "man", []string{"woman"}, "1993-03-03", 0.01)
	env.completeProfile(gus, "Gus", "man", []string{"woman"}, "1996-06-06", 0.01)

	env.swipe(eve, fred.ID, "like", http.StatusOK)
	if r := env.swipe(fred, eve.ID, "like", http.StatusOK); !r.Matched {
		t.Fatal("expected match")
	}

	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": eve.ID}, eve.Token), http.StatusBadRequest, "self block")
	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": fred.ID}, eve.Token), http.StatusNoContent, "block")
	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": fred.ID}, eve.Token), http.StatusNoContent, "block idempotent")
	env.expect(env.do(http.MethodPost, "/chats/"+eve.ID+"/messages", map[string]string{"content": "hello?"}, fred.Token), http.StatusNotFound, "blocked chat")
	env.expect(env.do(http.MethodGet, "/profiles/"+eve.ID, nil, fred.Token), http.StatusNotFound, "blocked profile view")
	env.expect(env.do(http.MethodGet, "/profiles/"+fred.ID, nil, eve.Token), http.StatusNotFound, "blocker profile view")

	res := env.do(http.MethodGet, "/blocks", nil, eve.Token)
	env.expect(res, http.StatusOK, "list blocks")
	if !strings.Contains(string(res.Body), fred.ID) {
		t.Fatalf("block list should contain fred: %s", res.Body)
	}
	env.expect(env.do(http.MethodDelete, "/blocks/"+fred.ID, nil, eve.Token), http.StatusNoContent, "unblock")
	env.expect(env.do(http.MethodPost, "/chats/"+eve.ID+"/messages", map[string]string{"content": "hello?"}, fred.Token), http.StatusNotFound, "block ended the match for good")

	// Report: invalid reason, then an underage report hides the profile and
	// blocks it for the reporter.
	if !containsProfile(env.discover(eve), gus.ID) {
		t.Fatal("eve should discover gus before reporting")
	}
	env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": gus.ID, "reason": "boring"}, eve.Token), http.StatusBadRequest, "bad reason")
	env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": gus.ID, "reason": "underage", "details": "Says he is 16"}, eve.Token), http.StatusCreated, "report")
	env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": gus.ID, "reason": "underage"}, eve.Token), http.StatusCreated, "report twice is idempotent")
	if containsProfile(env.discover(eve), gus.ID) {
		t.Fatal("reported profile must disappear for the reporter")
	}
	var hidden bool
	if err := env.pool.QueryRow(context.Background(), `SELECT moderation_hidden FROM profiles WHERE user_id = $1`, gus.ID).Scan(&hidden); err != nil || !hidden {
		t.Fatalf("underage report must hide the profile pending review (hidden=%v err=%v)", hidden, err)
	}
	var reports int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reported_user_id = $1`, gus.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("expected one open report, got %d", reports)
	}
}

func TestIncompleteProfileCannotDiscoverOrSwipe(t *testing.T) {
	env := setupEnv(t)
	newbie := env.register("newbie")
	res := env.do(http.MethodGet, "/discovery", nil, newbie.Token)
	env.expect(res, http.StatusConflict, "discovery incomplete")
	if !strings.Contains(string(res.Body), "profile_incomplete") {
		t.Fatalf("expected profile_incomplete code: %s", res.Body)
	}
	env.swipe(newbie, newbie.ID, "like", http.StatusBadRequest)
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	env := setupEnv(t)
	for round := 0; round < 3; round++ {
		a := env.register("race_a")
		b := env.register("race_b")
		env.completeProfile(a, "Ana", "woman", []string{"man"}, "1990-05-05", float64(round)*0.001)
		env.completeProfile(b, "Ben", "man", []string{"woman"}, "1990-06-06", float64(round)*0.001)

		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]member{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to member) {
				defer wg.Done()
				res := env.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": to.ID, "action": "like"}, from.Token)
				if res.Status == http.StatusOK {
					_ = json.Unmarshal(res.Body, &results[i])
				}
			}(i, pair[0], pair[1])
		}
		wg.Wait()

		var count int
		if err := env.pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		`, a.ID, b.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, count)
		}
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: one of the two likes must report the match", round)
		}
	}
}

func TestRealtimeMessageDelivery(t *testing.T) {
	env := setupEnv(t)
	ines := env.register("ines")
	jules := env.register("jules")
	env.completeProfile(ines, "Inès", "woman", []string{"man", "woman"}, "1997-07-07", 0)
	env.completeProfile(jules, "Jules", "man", []string{"woman"}, "1995-09-09", 0.01)

	server := httptest.NewServer(env.router)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime"

	// Unauthenticated sockets are closed.
	bad, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "garbage"})
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("invalid token must close the socket")
	}
	bad.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": jules.Token}); err != nil {
		t.Fatal(err)
	}
	readEvent := func() map[string]any {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var ev map[string]any
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read event: %v", err)
		}
		return ev
	}
	if ev := readEvent(); ev["type"] != "ready" {
		t.Fatalf("expected ready event, got %v", ev)
	}

	env.swipe(jules, ines.ID, "like", http.StatusOK)
	env.swipe(ines, jules.ID, "like", http.StatusOK)
	env.expect(env.do(http.MethodPost, "/chats/"+jules.ID+"/messages", map[string]string{"content": "Salut Jules !"}, ines.Token), http.StatusCreated, "send")

	seen := map[string]bool{}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !(seen["match.created"] && seen["message.created"]) {
		ev := readEvent()
		seen[ev["type"].(string)] = true
		if ev["type"] == "message.created" {
			data := ev["data"].(map[string]any)
			if data["content"] != "Salut Jules !" {
				t.Fatalf("unexpected message payload %v", data)
			}
		}
	}
	if !seen["match.created"] || !seen["message.created"] {
		t.Fatalf("expected match and message events, got %v", seen)
	}
}

func TestPausedProfileIsHiddenAndNotificationsReadAll(t *testing.T) {
	env := setupEnv(t)
	kim := env.register("kim")
	leo := env.register("leo")
	env.completeProfile(kim, "Kim", "woman", []string{"man"}, "1991-11-11", 0)
	env.completeProfile(leo, "Léo", "man", []string{"woman"}, "1992-12-12", 0.01)

	if !containsProfile(env.discover(leo), kim.ID) {
		t.Fatal("leo should discover kim")
	}
	env.expect(env.do(http.MethodPatch, "/me/profile", map[string]any{"isPaused": true}, kim.Token), http.StatusOK, "pause")
	if containsProfile(env.discover(leo), kim.ID) {
		t.Fatal("a paused profile must not be shown in discovery")
	}
	env.expect(env.do(http.MethodGet, "/profiles/"+kim.ID, nil, leo.Token), http.StatusNotFound, "paused profile view")
	env.swipe(leo, kim.ID, "like", http.StatusNotFound)

	// Hidden distance: the card keeps the city but drops the distance.
	env.expect(env.do(http.MethodPatch, "/me/profile", map[string]any{"isPaused": false, "showDistance": false}, kim.Token), http.StatusOK, "unpause")
	for _, p := range env.discover(leo) {
		if p.UserID == kim.ID && p.DistanceKm != nil {
			t.Fatal("distance must be hidden when showDistance is false")
		}
	}

	env.swipe(leo, kim.ID, "like", http.StatusOK)
	env.swipe(kim, leo.ID, "like", http.StatusOK)
	env.expect(env.do(http.MethodPost, "/notifications/read-all", nil, kim.Token), http.StatusNoContent, "read all")
	res := env.do(http.MethodGet, "/notifications", nil, kim.Token)
	var n struct {
		UnreadCount int `json:"unreadCount"`
	}
	res.decode(t, &n)
	if n.UnreadCount != 0 {
		t.Fatalf("expected no unread notification, got %d", n.UnreadCount)
	}

	res = env.do(http.MethodGet, "/interests", nil, kim.Token)
	env.expect(res, http.StatusOK, "interests")
	if !strings.Contains(string(res.Body), `"slug":"hiking"`) {
		t.Fatalf("interest catalog should be seeded: %s", res.Body)
	}
}
