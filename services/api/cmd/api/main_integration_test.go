package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

// captureMailer records outgoing mail so the password-reset flow can be tested.
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

func (m *captureMailer) last() mailer.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sent[len(m.sent)-1]
}

type testEnv struct {
	t         *testing.T
	router    *gin.Engine
	pool      *pgxpool.Pool
	mail      *captureMailer
	uploadDir string
	suffix    string
}

type session struct {
	ID           string
	Email        string
	Password     string
	AccessToken  string
	RefreshToken string
	IP           string
}

func setupEnv(t *testing.T) *testEnv {
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
	uploadDir := t.TempDir()
	store, err := storage.NewLocalStore(uploadDir)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	mail := &captureMailer{}
	a := &app{
		cfg: config.Config{
			Env:       config.EnvTest,
			JWTSecret: "integration-tests-signing-key-0123456789abcdef",
			UploadDir: uploadDir,
		},
		dbPool: pool,
		store:  store,
		mailer: mail,
		hub:    realtime.NewHub(),
	}
	router, err := setupRouter(a)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	env := &testEnv{t: t, router: router, pool: pool, mail: mail, uploadDir: uploadDir, suffix: randomHex(4)}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email LIKE $1`, "%+"+env.suffix+"@it.test")
		pool.Close()
	})
	return env
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *testEnv) do(method, path string, s *session, body any) response {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return e.send(req, s)
}

func (e *testEnv) send(req *http.Request, s *session) response {
	if s != nil {
		if s.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+s.AccessToken)
		}
		req.RemoteAddr = s.IP + ":40000"
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func expectStatus(t *testing.T, res response, want int, context string) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("%s: expected %d, got %d: %s", context, want, res.Status, res.Body)
	}
}

var ipCounter struct {
	sync.Mutex
	n int
}

func nextIP() string {
	ipCounter.Lock()
	defer ipCounter.Unlock()
	ipCounter.n++
	return fmt.Sprintf("10.%d.%d.%d", (ipCounter.n>>16)&255, (ipCounter.n>>8)&255, ipCounter.n&255)
}

func (e *testEnv) register(name string) *session {
	e.t.Helper()
	s := &session{Email: fmt.Sprintf("%s+%s@it.test", name, e.suffix), Password: "Password123", IP: nextIP()}
	res := e.do(http.MethodPost, "/auth/register", s, map[string]string{"email": strings.ToUpper(s.Email), "password": s.Password})
	expectStatus(e.t, res, http.StatusCreated, "register "+name)
	var payload struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	res.decode(e.t, &payload)
	if payload.User.Email != s.Email {
		e.t.Fatalf("expected normalised email %q, got %q", s.Email, payload.User.Email)
	}
	s.ID, s.AccessToken, s.RefreshToken = payload.User.ID, payload.AccessToken, payload.RefreshToken
	return s
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
		t.Fatalf("png: %v", err)
	}
	return buf.Bytes()
}

func (e *testEnv) upload(method, path string, s *session, data []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "picture.png")
	if err != nil {
		e.t.Fatalf("form file: %v", err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return e.send(req, s)
}

type profileSpec struct {
	FirstName    string
	Birthdate    string
	Gender       string
	InterestedIn []string
	Lat, Lng     float64
	InterestIDs  []int
}

func (e *testEnv) completeProfile(s *session, spec profileSpec) {
	e.t.Helper()
	res := e.do(http.MethodPatch, "/me/profile", s, map[string]any{
		"firstName": spec.FirstName, "birthdate": spec.Birthdate, "gender": spec.Gender,
		"bio": "Bio de test", "interestIds": spec.InterestIDs,
	})
	expectStatus(e.t, res, http.StatusOK, "patch profile")
	res = e.do(http.MethodPut, "/me/preferences", s, map[string]any{
		"interestedIn": spec.InterestedIn, "ageMin": 18, "ageMax": 60, "maxDistanceKm": 50,
	})
	expectStatus(e.t, res, http.StatusOK, "preferences")
	res = e.do(http.MethodPut, "/me/location", s, map[string]any{"latitude": spec.Lat, "longitude": spec.Lng, "city": "Paris"})
	expectStatus(e.t, res, http.StatusOK, "location")
	res = e.upload(http.MethodPost, "/me/photos", s, pngBytes(e.t, 320, 400, 10))
	expectStatus(e.t, res, http.StatusCreated, "upload photo")

	res = e.do(http.MethodGet, "/me/profile", s, nil)
	var me struct {
		IsComplete bool     `json:"isComplete"`
		Missing    []string `json:"missing"`
	}
	res.decode(e.t, &me)
	if !me.IsComplete {
		e.t.Fatalf("expected complete profile, missing %v", me.Missing)
	}
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
	SharedInterests int `json:"sharedInterests"`
}

func (e *testEnv) discover(s *session) []card {
	e.t.Helper()
	res := e.do(http.MethodGet, "/discover?limit=20", s, nil)
	expectStatus(e.t, res, http.StatusOK, "discover")
	var feed struct {
		Profiles []card `json:"profiles"`
	}
	res.decode(e.t, &feed)
	return feed.Profiles
}

func containsCard(cards []card, userID string) bool {
	for _, c := range cards {
		if c.UserID == userID {
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
		OtherUserID    string `json:"otherUserId"`
	} `json:"match"`
}

func (e *testEnv) swipe(s *session, targetID, action string, wantStatus int) swipeResult {
	e.t.Helper()
	res := e.do(http.MethodPost, "/swipes", s, map[string]string{"targetUserId": targetID, "action": action})
	expectStatus(e.t, res, wantStatus, "swipe "+action)
	var out swipeResult
	if wantStatus < 300 {
		res.decode(e.t, &out)
	}
	return out
}

func adultBirthdate(years int) string {
	return time.Now().AddDate(-years, 0, -10).Format("2006-01-02")
}

func (e *testEnv) interestIDs(s *session, n int) []int {
	res := e.do(http.MethodGet, "/interests", s, nil)
	expectStatus(e.t, res, http.StatusOK, "interests")
	var payload struct {
		Interests []struct {
			ID int `json:"id"`
		} `json:"interests"`
	}
	res.decode(e.t, &payload)
	if len(payload.Interests) < n {
		e.t.Fatalf("expected at least %d interests", n)
	}
	ids := make([]int, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, payload.Interests[i].ID)
	}
	return ids
}

func TestHealth(t *testing.T) {
	env := setupEnv(t)
	res := env.do(http.MethodGet, "/health", nil, nil)
	expectStatus(t, res, http.StatusOK, "health")
}

func TestAuthLifecycle(t *testing.T) {
	env := setupEnv(t)
	a := env.register("auth")

	res := env.do(http.MethodPost, "/auth/register", &session{IP: nextIP()}, map[string]string{"email": a.Email, "password": "Password123"})
	expectStatus(t, res, http.StatusConflict, "duplicate register")
	res = env.do(http.MethodPost, "/auth/register", &session{IP: nextIP()}, map[string]string{"email": "weak+" + env.suffix + "@it.test", "password": "short"})
	expectStatus(t, res, http.StatusBadRequest, "weak password")
	res = env.do(http.MethodPost, "/auth/register", &session{IP: nextIP()}, map[string]string{"email": "not-an-email", "password": "Password123"})
	expectStatus(t, res, http.StatusBadRequest, "invalid email")

	res = env.do(http.MethodGet, "/me", a, nil)
	expectStatus(t, res, http.StatusOK, "me")
	res = env.do(http.MethodGet, "/me", &session{AccessToken: "garbage", IP: a.IP}, nil)
	expectStatus(t, res, http.StatusUnauthorized, "invalid token")
	res = env.do(http.MethodGet, "/me", nil, nil)
	expectStatus(t, res, http.StatusUnauthorized, "missing token")

	res = env.do(http.MethodPost, "/auth/login", a, map[string]string{"email": a.Email, "password": "WrongPassword1"})
	expectStatus(t, res, http.StatusUnauthorized, "wrong password")
	res = env.do(http.MethodPost, "/auth/login", a, map[string]string{"email": "  " + strings.ToUpper(a.Email), "password": a.Password})
	expectStatus(t, res, http.StatusOK, "login")

	res = env.do(http.MethodPost, "/auth/refresh", a, map[string]string{"refreshToken": a.RefreshToken})
	expectStatus(t, res, http.StatusOK, "refresh")
	var refreshed struct {
		RefreshToken string `json:"refreshToken"`
	}
	res.decode(t, &refreshed)
	res = env.do(http.MethodPost, "/auth/refresh", a, map[string]string{"refreshToken": a.RefreshToken})
	expectStatus(t, res, http.StatusUnauthorized, "rotated refresh token must not be reusable")

	res = env.do(http.MethodPost, "/auth/logout", a, map[string]string{"refreshToken": refreshed.RefreshToken})
	expectStatus(t, res, http.StatusNoContent, "logout")
	res = env.do(http.MethodPost, "/auth/refresh", a, map[string]string{"refreshToken": refreshed.RefreshToken})
	expectStatus(t, res, http.StatusUnauthorized, "refresh after logout")
}

func TestPasswordResetAndChange(t *testing.T) {
	env := setupEnv(t)
	a := env.register("reset")

	res := env.do(http.MethodPost, "/auth/password/forgot", &session{IP: nextIP()}, map[string]string{"email": "nobody+" + env.suffix + "@it.test"})
	expectStatus(t, res, http.StatusAccepted, "forgot unknown email (no enumeration)")
	res = env.do(http.MethodPost, "/auth/password/forgot", a, map[string]string{"email": a.Email})
	expectStatus(t, res, http.StatusAccepted, "forgot")
	code := regexp.MustCompile(`code de réinitialisation est : ([A-Z0-9]{8})`).FindStringSubmatch(env.mail.last().Body)
	if code == nil {
		t.Fatalf("reset code not found in mail: %q", env.mail.last().Body)
	}

	res = env.do(http.MethodPost, "/auth/password/reset", a, map[string]string{"email": a.Email, "code": "ZZZZZZZZ", "newPassword": "NewPassword456"})
	expectStatus(t, res, http.StatusBadRequest, "wrong code")
	res = env.do(http.MethodPost, "/auth/password/reset", a, map[string]string{"email": a.Email, "code": strings.ToLower(code[1]), "newPassword": "NewPassword456"})
	expectStatus(t, res, http.StatusNoContent, "reset")
	res = env.do(http.MethodPost, "/auth/password/reset", a, map[string]string{"email": a.Email, "code": code[1], "newPassword": "Another789x"})
	expectStatus(t, res, http.StatusBadRequest, "code is single-use")

	res = env.do(http.MethodPost, "/auth/refresh", a, map[string]string{"refreshToken": a.RefreshToken})
	expectStatus(t, res, http.StatusUnauthorized, "reset revokes sessions")
	res = env.do(http.MethodPost, "/auth/login", a, map[string]string{"email": a.Email, "password": "NewPassword456"})
	expectStatus(t, res, http.StatusOK, "login with new password")
	var login struct {
		AccessToken string `json:"accessToken"`
	}
	res.decode(t, &login)
	a.AccessToken = login.AccessToken

	res = env.do(http.MethodPost, "/me/password", a, map[string]string{"currentPassword": "bad", "newPassword": "Changed123x"})
	expectStatus(t, res, http.StatusForbidden, "change password wrong current")
	res = env.do(http.MethodPost, "/me/password", a, map[string]string{"currentPassword": "NewPassword456", "newPassword": "Changed123x"})
	expectStatus(t, res, http.StatusNoContent, "change password")
}

func TestProfileValidation(t *testing.T) {
	env := setupEnv(t)
	a := env.register("validate")

	res := env.do(http.MethodPatch, "/me/profile", a, map[string]any{"birthdate": time.Now().AddDate(-17, 0, 0).Format("2006-01-02")})
	expectStatus(t, res, http.StatusBadRequest, "underage")
	res = env.do(http.MethodPatch, "/me/profile", a, map[string]any{"firstName": "<b>x</b>"})
	expectStatus(t, res, http.StatusBadRequest, "invalid name")
	res = env.do(http.MethodPatch, "/me/profile", a, map[string]any{"gender": "robot"})
	expectStatus(t, res, http.StatusBadRequest, "invalid gender")
	res = env.do(http.MethodPatch, "/me/profile", a, map[string]any{"interestIds": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}})
	expectStatus(t, res, http.StatusBadRequest, "too many interests")
	res = env.do(http.MethodPatch, "/me/profile", a, map[string]any{"interestIds": []int{32000}})
	expectStatus(t, res, http.StatusBadRequest, "unknown interest")
	res = env.do(http.MethodPut, "/me/preferences", a, map[string]any{"interestedIn": []string{"man"}, "ageMin": 30, "ageMax": 20, "maxDistanceKm": 10})
	expectStatus(t, res, http.StatusBadRequest, "invalid preferences")
	res = env.do(http.MethodPut, "/me/location", a, map[string]any{"latitude": 120, "longitude": 0})
	expectStatus(t, res, http.StatusBadRequest, "invalid location")
	res = env.do(http.MethodGet, "/discover", a, nil)
	expectStatus(t, res, http.StatusConflict, "discover requires a complete profile")
	res = env.do(http.MethodGet, "/discover?limit=abc", a, nil)
	expectStatus(t, res, http.StatusBadRequest, "invalid limit")

	res = env.do(http.MethodPut, "/me/location", a, map[string]any{"latitude": 48.856613, "longitude": 2.352222})
	expectStatus(t, res, http.StatusOK, "location")
	var stored struct{ Lat, Lng float64 }
	if err := env.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&stored.Lat, &stored.Lng); err != nil {
		t.Fatalf("read location: %v", err)
	}
	if stored.Lat != 48.86 || stored.Lng != 2.35 {
		t.Fatalf("location must be stored rounded, got %v,%v", stored.Lat, stored.Lng)
	}
	if strings.Contains(string(res.Body), "48.8") || strings.Contains(string(res.Body), "latitude") {
		t.Fatalf("coordinates must never be returned: %s", res.Body)
	}
}

func TestPhotosLifecycle(t *testing.T) {
	env := setupEnv(t)
	a := env.register("photos")
	other := env.register("photos_other")

	res := env.upload(http.MethodPost, "/me/photos", a, []byte("<?php echo 'pwned'; ?>"))
	expectStatus(t, res, http.StatusUnsupportedMediaType, "reject non-image")
	res = env.upload(http.MethodPost, "/me/photos", a, pngBytes(t, 50, 50, 1))
	expectStatus(t, res, http.StatusBadRequest, "reject tiny image")

	var photos struct {
		Photos []struct {
			ID       string `json:"id"`
			URL      string `json:"url"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	for i := 0; i < 6; i++ {
		res = env.upload(http.MethodPost, "/me/photos", a, pngBytes(t, 300, 300, uint8(i*30)))
		expectStatus(t, res, http.StatusCreated, "upload")
	}
	res.decode(t, &photos)
	if len(photos.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(photos.Photos))
	}
	res = env.upload(http.MethodPost, "/me/photos", a, pngBytes(t, 300, 300, 200))
	expectStatus(t, res, http.StatusConflict, "max 6 photos")

	files, _ := filepath.Glob(filepath.Join(env.uploadDir, "photos", "*.jpg"))
	if len(files) != 6 {
		t.Fatalf("expected 6 stored files, got %d", len(files))
	}

	// Signed URLs.
	res = env.do(http.MethodGet, photos.Photos[0].URL, nil, nil)
	expectStatus(t, res, http.StatusOK, "serve signed photo")
	if res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("unexpected content type %q", res.Header.Get("Content-Type"))
	}
	tampered := photos.Photos[0].URL[:len(photos.Photos[0].URL)-4] + "0000"
	res = env.do(http.MethodGet, tampered, nil, nil)
	expectStatus(t, res, http.StatusNotFound, "tampered signature")
	res = env.do(http.MethodGet, "/media/photos/"+photos.Photos[0].ID, nil, nil)
	expectStatus(t, res, http.StatusNotFound, "missing signature")

	// Reorder: last becomes primary.
	ids := []string{}
	for i := len(photos.Photos) - 1; i >= 0; i-- {
		ids = append(ids, photos.Photos[i].ID)
	}
	res = env.do(http.MethodPost, "/me/photos/reorder", a, map[string]any{"photoIds": ids})
	expectStatus(t, res, http.StatusOK, "reorder")
	res.decode(t, &photos)
	if photos.Photos[0].ID != ids[0] || photos.Photos[0].Position != 0 {
		t.Fatalf("reorder not applied: %+v", photos.Photos)
	}
	res = env.do(http.MethodPost, "/me/photos/reorder", a, map[string]any{"photoIds": ids[:2]})
	expectStatus(t, res, http.StatusBadRequest, "partial reorder rejected")

	// Another member cannot touch my photos.
	res = env.do(http.MethodDelete, "/me/photos/"+photos.Photos[0].ID, other, nil)
	expectStatus(t, res, http.StatusNotFound, "foreign delete")
	res = env.upload(http.MethodPut, "/me/photos/"+photos.Photos[0].ID, other, pngBytes(t, 300, 300, 3))
	expectStatus(t, res, http.StatusNotFound, "foreign replace")

	// Replace and delete.
	res = env.upload(http.MethodPut, "/me/photos/"+photos.Photos[0].ID, a, pngBytes(t, 400, 300, 99))
	expectStatus(t, res, http.StatusOK, "replace")
	res = env.do(http.MethodDelete, "/me/photos/"+photos.Photos[0].ID, a, nil)
	expectStatus(t, res, http.StatusOK, "delete")
	res.decode(t, &photos)
	if len(photos.Photos) != 5 || photos.Photos[0].Position != 0 {
		t.Fatalf("positions must be compacted after delete: %+v", photos.Photos)
	}
	files, _ = filepath.Glob(filepath.Join(env.uploadDir, "photos", "*.jpg"))
	if len(files) != 5 {
		t.Fatalf("replaced/deleted files must be removed from disk, found %d", len(files))
	}
}

// TestDatingCriticalFlow covers: register -> profile -> discovery -> like -> match ->
// conversation -> realtime -> read receipts -> local deletion -> block -> report -> account deletion.
func TestDatingCriticalFlow(t *testing.T) {
	env := setupEnv(t)
	alice := env.register("alice")
	bob := env.register("bob")
	chloe := env.register("chloe")
	dan := env.register("dan")

	shared := env.interestIDs(alice, 4)
	env.completeProfile(alice, profileSpec{FirstName: "Alice", Birthdate: adultBirthdate(29), Gender: "woman", InterestedIn: []string{"man"}, Lat: 48.8566, Lng: 2.3522, InterestIDs: shared[:3]})
	env.completeProfile(bob, profileSpec{FirstName: "Bob", Birthdate: adultBirthdate(31), Gender: "man", InterestedIn: []string{"woman"}, Lat: 48.8700, Lng: 2.3300, InterestIDs: shared[1:4]})
	// Chloe is only interested in women: Bob must never see her, nor she him.
	env.completeProfile(chloe, profileSpec{FirstName: "Chloé", Birthdate: adultBirthdate(27), Gender: "woman", InterestedIn: []string{"woman"}, Lat: 48.86, Lng: 2.34})
	// Dan lives in Lyon (~390 km): outside the 50 km radius.
	env.completeProfile(dan, profileSpec{FirstName: "Dan", Birthdate: adultBirthdate(30), Gender: "man", InterestedIn: []string{"woman"}, Lat: 45.764, Lng: 4.8357})

	// Discovery respects mutual preferences and distance, and never leaks coordinates.
	aliceFeed := env.discover(alice)
	if !containsCard(aliceFeed, bob.ID) || containsCard(aliceFeed, chloe.ID) || containsCard(aliceFeed, dan.ID) || containsCard(aliceFeed, alice.ID) {
		t.Fatalf("unexpected alice feed: %+v", aliceFeed)
	}
	for _, c := range aliceFeed {
		if c.UserID == bob.ID {
			if c.DistanceKm == nil || *c.DistanceKm < 1 || *c.DistanceKm > 5 {
				t.Fatalf("expected approximate distance for bob, got %v", c.DistanceKm)
			}
			if c.SharedInterests != 2 {
				t.Fatalf("expected 2 shared interests, got %d", c.SharedInterests)
			}
		}
	}
	if containsCard(env.discover(bob), chloe.ID) {
		t.Fatal("bob must not see chloe (not mutual)")
	}

	// Profile view permissions.
	expectStatus(t, env.do(http.MethodGet, "/profiles/"+bob.ID, alice, nil), http.StatusOK, "view discoverable profile")
	expectStatus(t, env.do(http.MethodGet, "/profiles/not-a-uuid", alice, nil), http.StatusBadRequest, "invalid profile id")

	// Like, duplicate prevention, match.
	if r := env.swipe(alice, bob.ID, "like", http.StatusOK); r.Matched {
		t.Fatal("one-sided like must not match")
	}
	env.swipe(alice, bob.ID, "like", http.StatusConflict)
	env.swipe(alice, bob.ID, "pass", http.StatusConflict)
	env.swipe(alice, alice.ID, "like", http.StatusBadRequest)
	env.swipe(alice, bob.ID, "superlike", http.StatusBadRequest)
	if containsCard(env.discover(alice), bob.ID) {
		t.Fatal("already swiped profile must not reappear")
	}

	// Realtime: Alice listens on a WebSocket.
	server := httptest.NewServer(env.router)
	defer server.Close()
	ws := dialRealtime(t, server.URL, alice.AccessToken)
	defer ws.Close()

	bobFeed := env.discover(bob)
	if !containsCard(bobFeed, alice.ID) {
		t.Fatal("bob should see alice")
	}
	match := env.swipe(bob, alice.ID, "like", http.StatusCreated)
	if !match.Matched || match.Match == nil || match.Match.ConversationID == "" || match.Match.OtherUserID != alice.ID {
		t.Fatalf("expected match, got %+v", match)
	}
	matchEvent := expectEvent(t, ws, "match.created")
	if !strings.Contains(string(matchEvent), `"initiatedByMe":false`) {
		t.Fatalf("the member who did not complete the match must get initiatedByMe=false: %s", matchEvent)
	}
	conversationID := match.Match.ConversationID

	var matchCount int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id IN ($1::uuid, $2::uuid) AND user_b_id IN ($1::uuid, $2::uuid)`, alice.ID, bob.ID).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchCount)
	}

	// Notifications.
	res := env.do(http.MethodGet, "/notifications", alice, nil)
	expectStatus(t, res, http.StatusOK, "notifications")
	var notifs struct {
		Notifications []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	res.decode(t, &notifs)
	if len(notifs.Notifications) == 0 || notifs.Notifications[0].Type != "match.new" || notifs.UnreadCount < 1 {
		t.Fatalf("expected match notification, got %+v", notifs)
	}
	expectStatus(t, env.do(http.MethodPost, "/notifications/"+notifs.Notifications[0].ID+"/read", bob, nil), http.StatusNotFound, "cannot read others' notifications")
	expectStatus(t, env.do(http.MethodPost, "/notifications/"+notifs.Notifications[0].ID+"/read", alice, nil), http.StatusOK, "mark read")
	expectStatus(t, env.do(http.MethodPost, "/notifications/read-all", alice, nil), http.StatusNoContent, "read all")

	// Chat.
	res = env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", bob, map[string]string{"content": "  Salut Alice !  ", "clientMessageId": "c1"})
	expectStatus(t, res, http.StatusCreated, "send message")
	var sent struct {
		ID              string `json:"id"`
		Content         string `json:"content"`
		ClientMessageID string `json:"clientMessageId"`
	}
	res.decode(t, &sent)
	if sent.Content != "Salut Alice !" || sent.ClientMessageID != "c1" {
		t.Fatalf("unexpected message %+v", sent)
	}
	event := expectEvent(t, ws, "message.created")
	if !strings.Contains(string(event), "Salut Alice") {
		t.Fatalf("realtime payload missing content: %s", event)
	}
	expectStatus(t, env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", bob, map[string]string{"content": "   "}), http.StatusBadRequest, "empty message")

	// Strangers can never read or write the conversation.
	expectStatus(t, env.do(http.MethodGet, "/conversations/"+conversationID+"/messages", chloe, nil), http.StatusNotFound, "stranger read")
	expectStatus(t, env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", chloe, map[string]string{"content": "intrusion"}), http.StatusNotFound, "stranger write")
	expectStatus(t, env.do(http.MethodGet, "/conversations/"+conversationID, chloe, nil), http.StatusNotFound, "stranger get")
	expectStatus(t, env.do(http.MethodDelete, "/matches/"+match.Match.ID, chloe, nil), http.StatusNotFound, "stranger unmatch")

	res = env.do(http.MethodGet, "/conversations", alice, nil)
	expectStatus(t, res, http.StatusOK, "list conversations")
	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			Profile     struct {
				FirstName string `json:"firstName"`
			} `json:"profile"`
		} `json:"conversations"`
	}
	res.decode(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].Profile.FirstName != "Bob" {
		t.Fatalf("unexpected conversations %+v", convs)
	}
	res = env.do(http.MethodGet, "/conversations/unread-count", alice, nil)
	if !strings.Contains(string(res.Body), `"unreadCount":1`) {
		t.Fatalf("expected one unread conversation, got %s", res.Body)
	}

	for i := 0; i < 3; i++ {
		expectStatus(t, env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", alice, map[string]string{"content": fmt.Sprintf("msg %d", i)}), http.StatusCreated, "alice sends")
	}
	res = env.do(http.MethodGet, "/conversations/"+conversationID+"/messages?limit=2", bob, nil)
	expectStatus(t, res, http.StatusOK, "list messages page 1")
	var page struct {
		Messages []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"messages"`
		NextCursor *string `json:"nextCursor"`
	}
	res.decode(t, &page)
	if len(page.Messages) != 2 || page.Messages[0].Content != "msg 2" || page.NextCursor == nil {
		t.Fatalf("unexpected first page %+v", page)
	}
	res = env.do(http.MethodGet, "/conversations/"+conversationID+"/messages?limit=2&before="+*page.NextCursor, bob, nil)
	res.decode(t, &page)
	if len(page.Messages) != 2 || page.Messages[0].Content != "msg 0" || page.Messages[1].Content != "Salut Alice !" || page.NextCursor != nil {
		t.Fatalf("unexpected second page %+v", page)
	}

	// Read receipts.
	expectStatus(t, env.do(http.MethodPost, "/conversations/"+conversationID+"/read", bob, nil), http.StatusOK, "mark read")
	expectEvent(t, ws, "conversation.read")
	res = env.do(http.MethodGet, "/conversations/"+conversationID, alice, nil)
	if !strings.Contains(string(res.Body), `"otherLastReadAt":"`) {
		t.Fatalf("expected read receipt timestamp, got %s", res.Body)
	}

	// Local deletion hides history for Alice only.
	expectStatus(t, env.do(http.MethodDelete, "/conversations/"+conversationID, alice, nil), http.StatusNoContent, "hide conversation")
	res = env.do(http.MethodGet, "/conversations", alice, nil)
	res.decode(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatalf("hidden conversation must disappear, got %+v", convs)
	}
	res = env.do(http.MethodGet, "/conversations", bob, nil)
	res.decode(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("hiding must not affect the other participant")
	}
	expectStatus(t, env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", bob, map[string]string{"content": "Tu es là ?"}), http.StatusCreated, "bob writes again")
	res = env.do(http.MethodGet, "/conversations/"+conversationID+"/messages", alice, nil)
	res.decode(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Content != "Tu es là ?" {
		t.Fatalf("only messages after local deletion must be visible, got %+v", page.Messages)
	}

	// Blocking ends the match and hides everything in both directions.
	expectStatus(t, env.do(http.MethodPost, "/blocks", alice, map[string]string{"userId": bob.ID}), http.StatusNoContent, "block")
	expectEvent(t, ws, "match.ended")
	expectStatus(t, env.do(http.MethodGet, "/conversations/"+conversationID+"/messages", bob, nil), http.StatusNotFound, "blocked read")
	expectStatus(t, env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", bob, map[string]string{"content": "?"}), http.StatusNotFound, "blocked write")
	expectStatus(t, env.do(http.MethodGet, "/profiles/"+alice.ID, bob, nil), http.StatusNotFound, "blocked profile view")
	res = env.do(http.MethodGet, "/blocks", alice, nil)
	if !strings.Contains(string(res.Body), bob.ID) {
		t.Fatalf("block list should contain bob: %s", res.Body)
	}
	expectStatus(t, env.do(http.MethodPost, "/blocks", alice, map[string]string{"userId": alice.ID}), http.StatusBadRequest, "self block")

	// Reports: validated, stored, and imply a block.
	expectStatus(t, env.do(http.MethodPost, "/reports", dan, map[string]string{"userId": alice.ID, "reason": "nope"}), http.StatusBadRequest, "invalid reason")
	expectStatus(t, env.do(http.MethodPost, "/reports", dan, map[string]string{"userId": alice.ID, "reason": "fake_profile", "details": "Photos volées"}), http.StatusCreated, "report")
	expectStatus(t, env.do(http.MethodPost, "/reports", dan, map[string]string{"userId": alice.ID, "reason": "fake_profile"}), http.StatusCreated, "duplicate report is idempotent")
	var reports, blocks int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, dan.ID, alice.ID).Scan(&reports)
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, dan.ID, alice.ID).Scan(&blocks)
	if reports != 1 || blocks != 1 {
		t.Fatalf("expected one report and one block, got %d/%d", reports, blocks)
	}

	// Account deletion requires the password and erases data and files.
	expectStatus(t, env.do(http.MethodDelete, "/me", alice, map[string]string{"password": "wrong"}), http.StatusForbidden, "delete wrong password")
	expectStatus(t, env.do(http.MethodDelete, "/me", alice, map[string]string{"password": alice.Password}), http.StatusNoContent, "delete account")
	expectStatus(t, env.do(http.MethodPost, "/auth/login", alice, map[string]string{"email": alice.Email, "password": alice.Password}), http.StatusUnauthorized, "login after deletion")
	var leftovers int
	_ = env.pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM photos WHERE user_id = $1) + (SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1)
		     + (SELECT COUNT(*) FROM messages WHERE sender_id = $1) + (SELECT COUNT(*) FROM profiles WHERE user_id = $1)
	`, alice.ID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("expected all personal data removed, %d rows left", leftovers)
	}
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id IS NULL`, dan.ID).Scan(&reports)
	if reports != 1 {
		t.Fatal("reports must be retained (anonymised) after the reported account is deleted")
	}
}

func TestConcurrentMutualLikesCreateSingleMatch(t *testing.T) {
	env := setupEnv(t)
	for round := 0; round < 5; round++ {
		a := env.register(fmt.Sprintf("race_a%d", round))
		b := env.register(fmt.Sprintf("race_b%d", round))
		env.completeProfile(a, profileSpec{FirstName: "Ana", Birthdate: adultBirthdate(25), Gender: "woman", InterestedIn: []string{"man", "woman", "nonbinary"}, Lat: 43.6, Lng: 1.44})
		env.completeProfile(b, profileSpec{FirstName: "Ben", Birthdate: adultBirthdate(26), Gender: "man", InterestedIn: []string{"woman"}, Lat: 43.61, Lng: 1.45})

		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]*session{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, swiper, target *session) {
				defer wg.Done()
				res := env.do(http.MethodPost, "/swipes", swiper, map[string]string{"targetUserId": target.ID, "action": "like"})
				_ = json.Unmarshal(res.Body, &results[i])
			}(i, pair[0], pair[1])
		}
		wg.Wait()

		matched := 0
		for _, r := range results {
			if r.Matched {
				matched++
			}
		}
		var rows int
		_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&rows)
		if matched != 1 || rows != 1 {
			t.Fatalf("round %d: expected exactly one match, got matched=%d rows=%d", round, matched, rows)
		}
	}
}

func TestUndoPassAndUnmatch(t *testing.T) {
	env := setupEnv(t)
	a := env.register("undo_a")
	b := env.register("undo_b")
	env.completeProfile(a, profileSpec{FirstName: "Inès", Birthdate: adultBirthdate(33), Gender: "woman", InterestedIn: []string{"nonbinary"}, Lat: 50.63, Lng: 3.06})
	env.completeProfile(b, profileSpec{FirstName: "Sam", Birthdate: adultBirthdate(34), Gender: "nonbinary", InterestedIn: []string{"woman"}, Lat: 50.64, Lng: 3.07})

	expectStatus(t, env.do(http.MethodPost, "/swipes/undo", a, nil), http.StatusNotFound, "nothing to undo")
	env.swipe(a, b.ID, "pass", http.StatusOK)
	if containsCard(env.discover(b), a.ID) {
		t.Fatal("someone who passed on you is not proposed")
	}
	expectStatus(t, env.do(http.MethodPost, "/swipes/undo", a, nil), http.StatusOK, "undo pass")
	if !containsCard(env.discover(a), b.ID) {
		t.Fatal("undone profile must come back")
	}
	env.swipe(a, b.ID, "like", http.StatusOK)
	expectStatus(t, env.do(http.MethodPost, "/swipes/undo", a, nil), http.StatusNotFound, "likes cannot be undone")
	m := env.swipe(b, a.ID, "like", http.StatusCreated)

	expectStatus(t, env.do(http.MethodDelete, "/matches/"+m.Match.ID, a, nil), http.StatusNoContent, "unmatch")
	expectStatus(t, env.do(http.MethodDelete, "/matches/"+m.Match.ID, a, nil), http.StatusNotFound, "unmatch twice")
	expectStatus(t, env.do(http.MethodGet, "/conversations/"+m.Match.ConversationID+"/messages", b, nil), http.StatusNotFound, "conversation closed after unmatch")
	if containsCard(env.discover(a), b.ID) || containsCard(env.discover(b), a.ID) {
		t.Fatal("unmatched members must not reappear in discovery")
	}

	// Pausing the profile removes it from discovery.
	c := env.register("undo_c")
	env.completeProfile(c, profileSpec{FirstName: "Léa", Birthdate: adultBirthdate(30), Gender: "woman", InterestedIn: []string{"nonbinary"}, Lat: 50.63, Lng: 3.06})
	if !containsCard(env.discover(b), c.ID) {
		t.Fatal("c should be visible")
	}
	expectStatus(t, env.do(http.MethodPatch, "/me/settings", c, map[string]bool{"discoverable": false}), http.StatusOK, "pause")
	if containsCard(env.discover(b), c.ID) {
		t.Fatal("paused profile must be hidden")
	}
	expectStatus(t, env.do(http.MethodPatch, "/me/settings", c, map[string]bool{"discoverable": true, "showDistance": false}), http.StatusOK, "unpause hide distance")
	for _, card := range env.discover(b) {
		if card.UserID == c.ID && card.DistanceKm != nil {
			t.Fatal("distance must be hidden when showDistance=false")
		}
	}
}

func TestRateLimitOnRegister(t *testing.T) {
	env := setupEnv(t)
	ip := nextIP()
	var last response
	for i := 0; i < 6; i++ {
		last = env.do(http.MethodPost, "/auth/register", &session{IP: ip}, map[string]string{
			"email": fmt.Sprintf("rl%d+%s@it.test", i, env.suffix), "password": "Password123",
		})
	}
	expectStatus(t, last, http.StatusTooManyRequests, "6th register from same IP")
	if last.Header.Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestOversizedJSONBodyIsRejected(t *testing.T) {
	env := setupEnv(t)
	a := env.register("bigbody")
	huge := strings.Repeat("a", 2<<20)
	res := env.do(http.MethodPatch, "/me/profile", a, map[string]string{"bio": huge})
	if res.Status != http.StatusBadRequest && res.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversized body to be rejected, got %d", res.Status)
	}
}

func TestRealtimeRejectsInvalidToken(t *testing.T) {
	env := setupEnv(t)
	server := httptest.NewServer(env.router)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/realtime", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": "invalid"})
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected connection to be closed for invalid token")
	}
}

func dialRealtime(t *testing.T, baseURL, token string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(strings.Replace(baseURL, "http", "ws", 1)+"/realtime", nil)
	if err != nil {
		t.Fatalf("dial realtime: %v", err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		t.Fatalf("auth realtime: %v", err)
	}
	expectEvent(t, conn, "ready")
	return conn
}

func expectEvent(t *testing.T, conn *websocket.Conn, eventType string) []byte {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for %s: %v", eventType, err)
		}
		var envelope struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(data, &envelope)
		if envelope.Type == eventType {
			return data
		}
	}
	t.Fatalf("event %s not received", eventType)
	return nil
}

func TestMain(m *testing.M) {
	if os.Getenv("TEST_VERBOSE_LOGS") == "" {
		log.SetOutput(io.Discard)
	}
	os.Exit(m.Run())
}
