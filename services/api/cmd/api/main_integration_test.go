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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
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

// captureMailer records e-mails so password reset codes can be read.
type captureMailer struct {
	mu    sync.Mutex
	sent  map[string]string
	count int
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sent == nil {
		m.sent = map[string]string{}
	}
	m.sent[to] = body
	m.count++
	return nil
}

func (m *captureMailer) last(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sent[to]
}

type env struct {
	t         *testing.T
	router    *gin.Engine
	pool      *pgxpool.Pool
	mail      *captureMailer
	uploadDir string
	emails    []string
}

var emailSeq atomic.Int64

func setup(t *testing.T) *env {
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
		t.Fatal(err)
	}
	mail := &captureMailer{}
	a := &app{
		cfg: config.Config{
			AppEnv:    "test",
			JWTSecret: "integration-test-secret-0123456789abcdef",
			UploadDir: uploadDir,
		},
		dbPool: pool,
		store:  store,
		mailer: mail,
		hub:    realtime.NewHub(),
	}
	e := &env{t: t, router: setupRouter(a), pool: pool, mail: mail, uploadDir: uploadDir}
	t.Cleanup(func() {
		if len(e.emails) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, e.emails)
		}
		pool.Close()
	})
	return e
}

type response struct {
	Status int
	Body   []byte
}

func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %s: %v", string(r.Body), err)
	}
}

func (e *env) do(method, path string, payload any, token string) response {
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
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", emailSeq.Load()%250, time.Now().Nanosecond()%250, time.Now().UnixNano()%250)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *env) expect(r response, status int, context string) {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("%s: expected %d, got %d: %s", context, status, r.Status, string(r.Body))
	}
}

type user struct {
	ID           string
	Email        string
	Token        string
	RefreshToken string
}

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s_%d_%d@lueur.test", prefix, time.Now().UnixNano(), emailSeq.Add(1))
	e.emails = append(e.emails, email)
	r := e.do("POST", "/auth/register", map[string]string{"email": email, "password": testPassword}, "")
	e.expect(r, http.StatusCreated, "register")
	var auth struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.decode(e.t, &auth)
	return user{ID: auth.User.ID, Email: email, Token: auth.AccessToken, RefreshToken: auth.RefreshToken}
}

type profileOpts struct {
	name         string
	gender       string
	interestedIn []string
	birthdate    string
	lat, lng     float64
	noLocation   bool
}

func testJPEG(t *testing.T, seed uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 480, 640))
	for y := 0; y < 640; y++ {
		for x := 0; x < 480; x++ {
			img.Set(x, y, color.RGBA{R: seed, G: uint8(x), B: uint8(y), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *env) upload(method, path, token string, data []byte) response {
	e.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", "photo.jpg")
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes()}
}

// onboard creates a complete, discoverable profile.
func (e *env) onboard(u user, o profileOpts) {
	e.t.Helper()
	if o.birthdate == "" {
		o.birthdate = "1995-04-12"
	}
	e.expect(e.do("PUT", "/me/profile", map[string]any{
		"firstName": o.name, "birthdate": o.birthdate, "gender": o.gender, "bio": "Bonjour", "city": "Paris",
	}, u.Token), http.StatusOK, "upsert profile")
	e.expect(e.do("PUT", "/me/preferences", map[string]any{
		"interestedIn": o.interestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50,
	}, u.Token), http.StatusOK, "preferences")
	if !o.noLocation {
		lat, lng := o.lat, o.lng
		if lat == 0 && lng == 0 {
			lat, lng = 48.8566, 2.3522
		}
		e.expect(e.do("PUT", "/me/location", map[string]any{"latitude": lat, "longitude": lng}, u.Token), http.StatusOK, "location")
	}
	e.expect(e.upload("POST", "/me/photos", u.Token, testJPEG(e.t, 10)), http.StatusCreated, "upload photo")
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
}

func (e *env) discover(u user) []card {
	e.t.Helper()
	r := e.do("GET", "/discovery?limit=20", nil, u.Token)
	e.expect(r, http.StatusOK, "discovery")
	var resp struct {
		Profiles []card `json:"profiles"`
	}
	r.decode(e.t, &resp)
	return resp.Profiles
}

func containsCard(cards []card, userID string) bool {
	for _, c := range cards {
		if c.UserID == userID {
			return true
		}
	}
	return false
}

type swipeResp struct {
	Action        string `json:"action"`
	AlreadySwiped bool   `json:"alreadySwiped"`
	Matched       bool   `json:"matched"`
	Match         *struct {
		ID string `json:"id"`
	} `json:"match"`
}

func (e *env) swipe(u user, target, action string) (int, swipeResp) {
	e.t.Helper()
	r := e.do("POST", "/discovery/swipes", map[string]string{"targetUserId": target, "action": action}, u.Token)
	var s swipeResp
	if r.Status < 300 {
		r.decode(e.t, &s)
	}
	return r.Status, s
}

// matchPair onboards two compatible users and makes them match.
func (e *env) matchPair() (user, user, string) {
	e.t.Helper()
	a := e.register("alice")
	b := e.register("bruno")
	e.onboard(a, profileOpts{name: "Alice", gender: "woman", interestedIn: []string{"man"}})
	e.onboard(b, profileOpts{name: "Bruno", gender: "man", interestedIn: []string{"woman"}})
	if status, _ := e.swipe(a, b.ID, "like"); status != http.StatusCreated {
		e.t.Fatalf("like a->b: %d", status)
	}
	status, s := e.swipe(b, a.ID, "like")
	if status != http.StatusCreated || !s.Matched || s.Match == nil {
		e.t.Fatalf("expected match, got %d %+v", status, s)
	}
	return a, b, s.Match.ID
}

func TestHealthEndpoint(t *testing.T) {
	e := setup(t)
	r := e.do("GET", "/health", nil, "")
	e.expect(r, http.StatusOK, "health")
	if !strings.Contains(string(r.Body), `"ok"`) {
		t.Fatalf("unexpected body %s", r.Body)
	}
}

func TestAuthFlow(t *testing.T) {
	e := setup(t)
	email := fmt.Sprintf("auth_%d@lueur.test", time.Now().UnixNano())
	e.emails = append(e.emails, email)

	e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": "short"}, ""), 400, "too short")
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": "onlyletters"}, ""), 400, "weak")
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": "not-an-email", "password": testPassword}, ""), 400, "invalid email")
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": testPassword}, ""), 201, "register trims and lowercases")
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": testPassword}, ""), 409, "duplicate (normalized)")

	e.expect(e.do("POST", "/auth/login", map[string]string{"email": email, "password": "Wrong12345"}, ""), 401, "wrong password")
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": "nobody@lueur.test", "password": testPassword}, ""), 401, "unknown email")
	r := e.do("POST", "/auth/login", map[string]string{"email": email, "password": testPassword}, "")
	e.expect(r, 200, "login")
	var auth struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	r.decode(t, &auth)

	me := e.do("GET", "/me", nil, auth.AccessToken)
	e.expect(me, 200, "me")
	if !strings.Contains(string(me.Body), email) {
		t.Fatalf("me should return normalized email: %s", me.Body)
	}
	e.expect(e.do("GET", "/me", nil, ""), 401, "me without token")
	e.expect(e.do("GET", "/me", nil, "garbage"), 401, "me with garbage token")

	refreshed := e.do("POST", "/auth/refresh", map[string]string{"refreshToken": auth.RefreshToken}, "")
	e.expect(refreshed, 200, "refresh")
	var next struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	refreshed.decode(t, &next)
	if next.RefreshToken == auth.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": auth.RefreshToken}, ""), 401, "old refresh token reuse")

	e.expect(e.do("POST", "/auth/logout", map[string]string{"refreshToken": next.RefreshToken}, ""), 204, "logout")
	e.expect(e.do("GET", "/me", nil, next.AccessToken), 401, "access token is revoked with its session")
}

func TestPasswordReset(t *testing.T) {
	e := setup(t)
	u := e.register("reset")

	e.expect(e.do("POST", "/auth/password/forgot", map[string]string{"email": "unknown@lueur.test"}, ""), 202, "unknown email still 202")
	e.expect(e.do("POST", "/auth/password/forgot", map[string]string{"email": u.Email}, ""), 202, "forgot")
	body := e.mail.last(u.Email)
	code := regexp.MustCompile(`: ([A-Z0-9]{8})`).FindStringSubmatch(body)
	if len(code) != 2 {
		t.Fatalf("no code in mail %q", body)
	}

	e.expect(e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": "AAAAAAAA", "password": "NewPassw0rd"}, ""), 400, "wrong code")
	e.expect(e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": strings.ToLower(code[1]), "password": "NewPassw0rd"}, ""), 204, "reset")
	e.expect(e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": code[1], "password": "Another1pass"}, ""), 400, "code is single use")

	e.expect(e.do("GET", "/me", nil, u.Token), 401, "sessions revoked after reset")
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": testPassword}, ""), 401, "old password")
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "NewPassw0rd"}, ""), 200, "new password")
}

func TestPasswordResetLocksAfterTooManyAttempts(t *testing.T) {
	e := setup(t)
	u := e.register("bruteforce")
	e.expect(e.do("POST", "/auth/password/forgot", map[string]string{"email": u.Email}, ""), 202, "forgot")
	code := regexp.MustCompile(`: ([A-Z0-9]{8})`).FindStringSubmatch(e.mail.last(u.Email))[1]
	for i := 0; i < 5; i++ {
		e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": "ZZZZZZZZ", "password": "NewPassw0rd"}, "")
	}
	e.expect(e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "password": "NewPassw0rd"}, ""), 400, "code burned after 5 failures")
}

func TestProfileValidationAndCompletion(t *testing.T) {
	e := setup(t)
	u := e.register("profile")

	r := e.do("GET", "/me/profile", nil, u.Token)
	e.expect(r, 200, "empty profile")
	var mine struct {
		Exists     bool     `json:"exists"`
		IsComplete bool     `json:"isComplete"`
		Missing    []string `json:"missing"`
	}
	r.decode(t, &mine)
	if mine.Exists || mine.IsComplete {
		t.Fatalf("new account must not have a profile: %+v", mine)
	}

	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Zoé", "birthdate": minor, "gender": "woman"}, u.Token), 400, "minor rejected")
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Zoé", "birthdate": "1990-01-01", "gender": "robot"}, u.Token), 400, "bad gender")
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "<b>", "birthdate": "1990-01-01", "gender": "woman"}, u.Token), 400, "bad name")
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Zoé", "birthdate": "1990-01-01", "gender": "woman", "bio": strings.Repeat("x", 501)}, u.Token), 400, "bio too long")
	e.expect(e.upload("POST", "/me/photos", u.Token, testJPEG(t, 1)), 409, "photo before profile")
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": " Zoé ", "birthdate": "1990-01-01", "gender": "woman", "bio": "Hello"}, u.Token), 200, "valid profile")

	var interests struct {
		Interests []struct {
			ID int `json:"id"`
		} `json:"interests"`
	}
	ir := e.do("GET", "/interests", nil, u.Token)
	e.expect(ir, 200, "interests")
	ir.decode(t, &interests)
	if len(interests.Interests) < 20 {
		t.Fatalf("expected interest catalog, got %d", len(interests.Interests))
	}
	ids := []int{}
	for _, i := range interests.Interests[:11] {
		ids = append(ids, i.ID)
	}
	e.expect(e.do("PUT", "/me/interests", map[string]any{"interestIds": ids}, u.Token), 400, "too many interests")
	e.expect(e.do("PUT", "/me/interests", map[string]any{"interestIds": []int{999999}}, u.Token), 400, "unknown interest")
	e.expect(e.do("PUT", "/me/interests", map[string]any{"interestIds": ids[:3]}, u.Token), 200, "interests")
	e.expect(e.do("PUT", "/me/location", map[string]any{"latitude": 123.0, "longitude": 2.0}, u.Token), 400, "bad latitude")
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 10}, u.Token), 400, "bad age range")

	// Photos: upload, reject non-images, enforce limit, reorder, replace, delete.
	e.expect(e.upload("POST", "/me/photos", u.Token, []byte("#!/bin/sh\nrm -rf /")), 415, "script rejected")
	var photos struct {
		Photos []struct {
			ID       string `json:"id"`
			URL      string `json:"url"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	for i := 0; i < 6; i++ {
		r := e.upload("POST", "/me/photos", u.Token, testJPEG(t, uint8(i)))
		e.expect(r, 201, "upload")
		r.decode(t, &photos)
	}
	if len(photos.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(photos.Photos))
	}
	e.expect(e.upload("POST", "/me/photos", u.Token, testJPEG(t, 7)), 409, "7th photo")

	reversed := []string{}
	for i := len(photos.Photos) - 1; i >= 0; i-- {
		reversed = append(reversed, photos.Photos[i].ID)
	}
	e.expect(e.do("PUT", "/me/photos/order", map[string]any{"photoIds": reversed[:5]}, u.Token), 400, "partial order")
	or := e.do("PUT", "/me/photos/order", map[string]any{"photoIds": reversed}, u.Token)
	e.expect(or, 200, "reorder")
	or.decode(t, &photos)
	if photos.Photos[0].ID != reversed[0] || photos.Photos[0].Position != 0 {
		t.Fatal("reorder must make the chosen photo the main one")
	}

	main := photos.Photos[0]
	media := e.do("GET", main.URL, nil, "")
	e.expect(media, 200, "signed media")
	if !bytes.HasPrefix(media.Body, []byte{0xFF, 0xD8}) {
		t.Fatal("media must be a JPEG")
	}
	e.expect(e.do("GET", "/media/photos/"+main.ID+"?exp=9999999999&sig=forged", nil, ""), 404, "forged signature")
	e.expect(e.do("GET", "/media/photos/"+main.ID, nil, ""), 404, "unsigned")

	rr := e.upload("PUT", "/me/photos/"+main.ID, u.Token, testJPEG(t, 99))
	e.expect(rr, 200, "replace")
	rr.decode(t, &photos)
	if photos.Photos[0].ID != main.ID {
		t.Fatal("replace keeps id and position")
	}

	other := e.register("intruder")
	e.expect(e.do("DELETE", "/me/photos/"+main.ID, nil, other.Token), 404, "cannot delete someone else's photo")
	dr := e.do("DELETE", "/me/photos/"+main.ID, nil, u.Token)
	e.expect(dr, 200, "delete photo")
	dr.decode(t, &photos)
	if len(photos.Photos) != 5 || photos.Photos[0].Position != 0 {
		t.Fatal("positions must stay contiguous after delete")
	}

	r = e.do("GET", "/me/profile", nil, u.Token)
	r.decode(t, &mine)
	if !mine.IsComplete {
		t.Fatalf("profile should be complete: %s", r.Body)
	}
	if strings.Contains(string(r.Body), "latitude") || strings.Contains(string(r.Body), "longitude") {
		t.Fatal("coordinates must never be returned")
	}
}

func TestDiscoveryLikeMatchAndChat(t *testing.T) {
	e := setup(t)
	alice := e.register("alice")
	bruno := e.register("bruno")
	carl := e.register("carl")   // man interested in men: not compatible with alice
	david := e.register("david") // compatible but in Lyon (beyond 50 km)
	emile := e.register("emile") // compatible, passed by alice
	e.onboard(alice, profileOpts{name: "Alice", gender: "woman", interestedIn: []string{"man"}})
	e.onboard(bruno, profileOpts{name: "Bruno", gender: "man", interestedIn: []string{"woman"}, lat: 48.8666, lng: 2.3333})
	e.onboard(carl, profileOpts{name: "Carl", gender: "man", interestedIn: []string{"man"}})
	e.onboard(david, profileOpts{name: "David", gender: "man", interestedIn: []string{"woman"}, lat: 45.764, lng: 4.8357})
	e.onboard(emile, profileOpts{name: "Emile", gender: "man", interestedIn: []string{"woman"}})

	incomplete := e.register("incomplete")
	e.expect(e.do("GET", "/discovery", nil, incomplete.Token), 409, "discovery requires complete profile")

	feed := e.discover(alice)
	if !containsCard(feed, bruno.ID) || !containsCard(feed, emile.ID) {
		t.Fatal("compatible nearby profiles must be discoverable")
	}
	if containsCard(feed, carl.ID) {
		t.Fatal("gender preferences must be mutual")
	}
	if containsCard(feed, david.ID) {
		t.Fatal("profiles beyond max distance must be excluded")
	}
	if containsCard(feed, alice.ID) {
		t.Fatal("viewer must never see themselves")
	}
	for _, c := range feed {
		if c.UserID == bruno.ID {
			if c.DistanceKm == nil || *c.DistanceKm < 1 || *c.DistanceKm > 3 {
				t.Fatalf("expected approximate distance ~1-2 km, got %v", c.DistanceKm)
			}
			if len(c.Photos) == 0 || !strings.HasPrefix(c.Photos[0].URL, "/media/photos/") {
				t.Fatal("cards must expose signed photo URLs")
			}
		}
	}

	// Pass: never shown again.
	if status, s := e.swipe(alice, emile.ID, "pass"); status != 201 || s.Matched {
		t.Fatalf("pass failed: %d", status)
	}
	if containsCard(e.discover(alice), emile.ID) {
		t.Fatal("passed profiles must not reappear")
	}
	if status, s := e.swipe(alice, emile.ID, "like"); status != 200 || !s.AlreadySwiped || s.Action != "pass" {
		t.Fatalf("a processed profile cannot be re-swiped: %d %+v", status, s)
	}

	if status, _ := e.swipe(alice, alice.ID, "like"); status != 400 {
		t.Fatal("self-like must be rejected")
	}
	if status, _ := e.swipe(alice, bruno.ID, "superlike"); status != 400 {
		t.Fatal("unknown action must be rejected")
	}
	if status, _ := e.swipe(alice, "00000000-0000-0000-0000-000000000000", "like"); status != 404 {
		t.Fatal("unknown target must be 404")
	}

	// Like without reciprocity: no match. Repeated like is idempotent.
	if status, s := e.swipe(alice, bruno.ID, "like"); status != 201 || s.Matched {
		t.Fatalf("first like: %d %+v", status, s)
	}
	if status, s := e.swipe(alice, bruno.ID, "like"); status != 200 || !s.AlreadySwiped || s.Matched {
		t.Fatalf("repeated like: %d %+v", status, s)
	}
	if !containsCard(e.discover(bruno), alice.ID) {
		t.Fatal("bruno should discover alice")
	}

	// Reciprocal like: match.
	status, s := e.swipe(bruno, alice.ID, "like")
	if status != 201 || !s.Matched || s.Match == nil {
		t.Fatalf("expected match: %d %+v", status, s)
	}
	matchID := s.Match.ID
	if status, s := e.swipe(bruno, alice.ID, "like"); status != 200 || !s.Matched || s.Match.ID != matchID {
		t.Fatal("re-like after match must return the same match")
	}
	var matchCount int
	if err := e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE id = $1`, matchID).Scan(&matchCount); err != nil || matchCount != 1 {
		t.Fatalf("exactly one match row expected, got %d (%v)", matchCount, err)
	}
	if containsCard(e.discover(alice), bruno.ID) || containsCard(e.discover(bruno), alice.ID) {
		t.Fatal("matched profiles leave discovery")
	}

	// Both get a match notification.
	for _, u := range []user{alice, bruno} {
		r := e.do("GET", "/notifications/unread-count", nil, u.Token)
		e.expect(r, 200, "unread count")
		if !strings.Contains(string(r.Body), `"unread":1`) {
			t.Fatalf("expected 1 unread notification: %s", r.Body)
		}
	}

	// Matches list.
	r := e.do("GET", "/matches", nil, alice.Token)
	e.expect(r, 200, "matches")
	if !strings.Contains(string(r.Body), matchID) || !strings.Contains(string(r.Body), "Bruno") {
		t.Fatalf("match should be listed: %s", r.Body)
	}
	if strings.Contains(string(r.Body), bruno.Email) {
		t.Fatal("emails must never leak")
	}

	// Chat: only members can read or write.
	e.expect(e.do("GET", "/matches/"+matchID+"/messages", nil, carl.Token), 404, "outsider cannot read")
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "hi"}, carl.Token), 404, "outsider cannot write")
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "   "}, alice.Token), 400, "empty message")
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": strings.Repeat("a", 2001)}, alice.Token), 400, "too long")

	sent := e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "Salut Bruno !", "clientId": "c-1"}, alice.Token)
	e.expect(sent, 201, "send")
	retry := e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "Salut Bruno !", "clientId": "c-1"}, alice.Token)
	e.expect(retry, 201, "idempotent retry")
	var m1, m2 struct {
		ID     string `json:"id"`
		IsRead bool   `json:"isRead"`
	}
	sent.decode(t, &m1)
	retry.decode(t, &m2)
	if m1.ID != m2.ID {
		t.Fatal("same clientId must not create a duplicate")
	}

	r = e.do("GET", "/matches", nil, bruno.Token)
	if !strings.Contains(string(r.Body), `"unreadCount":1`) || !strings.Contains(string(r.Body), "Salut Bruno") {
		t.Fatalf("bruno should see 1 unread message: %s", r.Body)
	}
	e.expect(e.do("POST", "/matches/"+matchID+"/read", nil, bruno.Token), 200, "mark read")
	r = e.do("GET", "/matches", nil, bruno.Token)
	if !strings.Contains(string(r.Body), `"unreadCount":0`) {
		t.Fatalf("unread must reset: %s", r.Body)
	}

	r = e.do("GET", "/matches/"+matchID+"/messages", nil, alice.Token)
	e.expect(r, 200, "alice messages")
	var msgs struct {
		Messages []struct {
			ID     string `json:"id"`
			IsRead bool   `json:"isRead"`
		} `json:"messages"`
	}
	r.decode(t, &msgs)
	if len(msgs.Messages) != 1 || !msgs.Messages[0].IsRead {
		t.Fatalf("alice should see her message as read: %s", r.Body)
	}

	// Pagination over messages.
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": fmt.Sprintf("msg %d", i)}, bruno.Token), 201, "send")
	}
	r = e.do("GET", "/matches/"+matchID+"/messages?limit=4", nil, alice.Token)
	var page struct {
		Messages   []json.RawMessage `json:"messages"`
		NextCursor string            `json:"nextCursor"`
	}
	r.decode(t, &page)
	if len(page.Messages) != 4 || page.NextCursor == "" {
		t.Fatalf("expected a full first page with a cursor: %s", r.Body)
	}
	r = e.do("GET", "/matches/"+matchID+"/messages?limit=4&cursor="+page.NextCursor, nil, alice.Token)
	r.decode(t, &page)
	if len(page.Messages) != 2 {
		t.Fatalf("expected the 2 remaining messages, got %d", len(page.Messages))
	}

	// Public profile visibility.
	e.expect(e.do("GET", "/profiles/"+bruno.ID, nil, alice.Token), 200, "matched profile visible")

	// Unmatch removes the conversation for both.
	e.expect(e.do("DELETE", "/matches/"+matchID, nil, carl.Token), 404, "outsider cannot unmatch")
	e.expect(e.do("DELETE", "/matches/"+matchID, nil, alice.Token), 204, "unmatch")
	e.expect(e.do("GET", "/matches/"+matchID+"/messages", nil, bruno.Token), 404, "conversation gone")
	if containsCard(e.discover(alice), bruno.ID) {
		t.Fatal("unmatched profiles do not resurface")
	}
}

func TestConcurrentMutualLikesCreateExactlyOneMatch(t *testing.T) {
	e := setup(t)
	for round := 0; round < 5; round++ {
		a := e.register("ca")
		b := e.register("cb")
		e.onboard(a, profileOpts{name: "Ana", gender: "woman", interestedIn: []string{"man"}})
		e.onboard(b, profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}})

		var wg sync.WaitGroup
		results := make([]swipeResp, 2)
		wg.Add(2)
		go func() { defer wg.Done(); _, results[0] = e.swipe(a, b.ID, "like") }()
		go func() { defer wg.Done(); _, results[1] = e.swipe(b, a.ID, "like") }()
		wg.Wait()

		var count int
		if err := e.pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
		`, a.ID, b.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, count)
		}
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: one of the swipes must report the match", round)
		}
	}
}

func TestBlockAndReport(t *testing.T) {
	e := setup(t)
	alice, bruno, matchID := e.matchPair()
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "Message déplacé"}, bruno.Token), 201, "send")

	e.expect(e.do("POST", "/blocks", map[string]string{"userId": alice.ID}, alice.Token), 400, "cannot block self")
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}, alice.Token), 404, "unknown user")

	// Report (auto-blocks): evidence is snapshotted before the conversation is deleted.
	e.expect(e.do("POST", "/reports", map[string]any{"userId": bruno.ID, "reason": "nope"}, alice.Token), 400, "invalid reason")
	r := e.do("POST", "/reports", map[string]any{"userId": bruno.ID, "reason": "harassment", "details": "insistant"}, alice.Token)
	e.expect(r, 201, "report")
	if !strings.Contains(string(r.Body), `"blocked":true`) {
		t.Fatalf("report should block by default: %s", r.Body)
	}
	var evidence string
	if err := e.pool.QueryRow(context.Background(), `SELECT evidence::text FROM reports WHERE reporter_user_id = $1 AND reported_user_id = $2`, alice.ID, bruno.ID).Scan(&evidence); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(evidence, "Message déplacé") || !strings.Contains(evidence, `"reported"`) {
		t.Fatalf("evidence should contain the conversation: %s", evidence)
	}

	e.expect(e.do("GET", "/matches/"+matchID+"/messages", nil, bruno.Token), 404, "blocked: conversation gone")
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "encore"}, bruno.Token), 404, "blocked: cannot write")
	e.expect(e.do("GET", "/profiles/"+alice.ID, nil, bruno.Token), 404, "blocked: profile hidden")
	e.expect(e.do("GET", "/profiles/"+bruno.ID, nil, alice.Token), 404, "blocker does not see either")
	if containsCard(e.discover(bruno), alice.ID) {
		t.Fatal("blocked users never see each other")
	}

	lr := e.do("GET", "/blocks", nil, alice.Token)
	e.expect(lr, 200, "list blocks")
	if !strings.Contains(string(lr.Body), bruno.ID) {
		t.Fatalf("block list should contain bruno: %s", lr.Body)
	}
	e.expect(e.do("DELETE", "/blocks/"+bruno.ID, nil, alice.Token), 204, "unblock")
	e.expect(e.do("DELETE", "/blocks/"+bruno.ID, nil, alice.Token), 404, "unblock twice")

	// Three distinct reporters hide a profile from discovery.
	target := e.register("spammer")
	e.onboard(target, profileOpts{name: "Spam", gender: "man", interestedIn: []string{"woman"}})
	viewer := e.register("viewer")
	e.onboard(viewer, profileOpts{name: "Vera", gender: "woman", interestedIn: []string{"man"}})
	if !containsCard(e.discover(viewer), target.ID) {
		t.Fatal("target should be visible before reports")
	}
	// Throwaway accounts (no profile, never interacted) cannot hide anyone.
	for i := 0; i < 3; i++ {
		throwaway := e.register("throwaway")
		e.expect(e.do("POST", "/reports", map[string]any{"userId": target.ID, "reason": "spam", "block": false}, throwaway.Token), 201, "report")
	}
	if !containsCard(e.discover(viewer), target.ID) {
		t.Fatal("reports from throwaway accounts must not hide a profile")
	}
	// Three credible reporters (complete profile + real interaction) do.
	for i := 0; i < 3; i++ {
		reporter := e.register("reporter")
		e.onboard(reporter, profileOpts{name: "Rita", gender: "woman", interestedIn: []string{"man"}})
		if status, _ := e.swipe(reporter, target.ID, "pass"); status != 201 {
			t.Fatalf("reporter swipe: %d", status)
		}
		e.expect(e.do("POST", "/reports", map[string]any{"userId": target.ID, "reason": "spam", "block": false}, reporter.Token), 201, "report")
	}
	if containsCard(e.discover(viewer), target.ID) {
		t.Fatal("profile must be hidden after 3 credible reports")
	}
	// The hidden user is restricted too: no discovery.
	e.expect(e.do("GET", "/discovery", nil, target.Token), 409, "moderated user cannot discover")
}

func TestPrivacySettings(t *testing.T) {
	e := setup(t)
	a := e.register("priv-a")
	b := e.register("priv-b")
	e.onboard(a, profileOpts{name: "Anna", gender: "woman", interestedIn: []string{"man"}})
	e.onboard(b, profileOpts{name: "Basile", gender: "man", interestedIn: []string{"woman"}})

	e.expect(e.do("PUT", "/me/privacy", map[string]any{"isVisible": true, "showDistance": false}, b.Token), 200, "hide distance")
	for _, c := range e.discover(a) {
		if c.UserID == b.ID && c.DistanceKm != nil {
			t.Fatal("distance must be hidden when the user opts out")
		}
	}
	e.expect(e.do("PUT", "/me/privacy", map[string]any{"isVisible": false, "showDistance": false}, b.Token), 200, "pause profile")
	if containsCard(e.discover(a), b.ID) {
		t.Fatal("paused profiles leave discovery")
	}
	if status, _ := e.swipe(a, b.ID, "like"); status != 404 {
		t.Fatalf("cannot like a paused profile, got %d", status)
	}
	e.expect(e.do("DELETE", "/me/location", nil, a.Token), 200, "clear location")
}

func TestAgePreferencesAreMutual(t *testing.T) {
	e := setup(t)
	young := e.register("young")
	older := e.register("older")
	e.onboard(young, profileOpts{name: "Lina", gender: "woman", interestedIn: []string{"man"}, birthdate: time.Now().AddDate(-22, 0, 0).Format("2006-01-02")})
	e.onboard(older, profileOpts{name: "Marc", gender: "man", interestedIn: []string{"woman"}, birthdate: time.Now().AddDate(-45, 0, 0).Format("2006-01-02")})
	if !containsCard(e.discover(young), older.ID) {
		t.Fatal("open preferences should match")
	}
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 50}, young.Token), 200, "narrow ages")
	if containsCard(e.discover(young), older.ID) {
		t.Fatal("viewer age range must be applied")
	}
	if containsCard(e.discover(older), young.ID) {
		t.Fatal("candidate age range must also be applied (mutual)")
	}
}

func TestDeleteAccountRemovesData(t *testing.T) {
	e := setup(t)
	alice, bruno, matchID := e.matchPair()
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "coucou"}, alice.Token), 201, "send")

	files := 0
	_ = filepath.Walk(e.uploadDir, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			files++
		}
		return nil
	})
	if files != 2 {
		t.Fatalf("expected 2 stored photos, got %d", files)
	}

	e.expect(e.do("DELETE", "/me", map[string]string{"password": "Wrong12345"}, alice.Token), 403, "wrong password")
	e.expect(e.do("DELETE", "/me", map[string]string{"password": testPassword}, alice.Token), 204, "delete account")

	e.expect(e.do("GET", "/me", nil, alice.Token), 401, "token unusable after deletion")
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": alice.Email, "password": testPassword}, ""), 401, "cannot login")
	e.expect(e.do("GET", "/matches/"+matchID+"/messages", nil, bruno.Token), 404, "conversation deleted")
	r := e.do("GET", "/matches", nil, bruno.Token)
	if strings.Contains(string(r.Body), matchID) {
		t.Fatal("match must be deleted")
	}
	var remaining int
	for _, q := range []string{
		`SELECT COUNT(*) FROM profiles WHERE user_id = $1`,
		`SELECT COUNT(*) FROM photos WHERE user_id = $1`,
		`SELECT COUNT(*) FROM swipes WHERE swiper_user_id = $1 OR target_user_id = $1`,
		`SELECT COUNT(*) FROM messages WHERE sender_user_id = $1`,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`,
		`SELECT COUNT(*) FROM sessions WHERE user_id = $1`,
	} {
		if err := e.pool.QueryRow(context.Background(), q, alice.ID).Scan(&remaining); err != nil || remaining != 0 {
			t.Fatalf("%s: expected 0 rows, got %d (%v)", q, remaining, err)
		}
	}
	files = 0
	_ = filepath.Walk(e.uploadDir, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			files++
		}
		return nil
	})
	if files != 1 {
		t.Fatalf("alice's photo file must be deleted from storage, %d files remain", files)
	}
}

func TestNotificationsFlow(t *testing.T) {
	e := setup(t)
	alice, _, _ := e.matchPair()
	r := e.do("GET", "/notifications?limit=10", nil, alice.Token)
	e.expect(r, 200, "list")
	var list struct {
		Notifications []struct {
			ID     string            `json:"id"`
			Type   string            `json:"type"`
			IsRead bool              `json:"isRead"`
			Data   map[string]string `json:"data"`
		} `json:"notifications"`
	}
	r.decode(t, &list)
	if len(list.Notifications) != 1 || list.Notifications[0].Type != "match" || list.Notifications[0].Data["matchId"] == "" {
		t.Fatalf("expected one match notification: %s", r.Body)
	}
	other := e.register("other")
	e.expect(e.do("POST", "/notifications/"+list.Notifications[0].ID+"/read", nil, other.Token), 404, "cannot read other's notification")
	e.expect(e.do("POST", "/notifications/"+list.Notifications[0].ID+"/read", nil, alice.Token), 200, "mark read")
	e.expect(e.do("POST", "/notifications/read-all", nil, alice.Token), 204, "read all")
	r = e.do("GET", "/notifications/unread-count", nil, alice.Token)
	if !strings.Contains(string(r.Body), `"unread":0`) {
		t.Fatalf("expected 0 unread: %s", r.Body)
	}
	e.expect(e.do("POST", "/notifications", map[string]string{"type": "x", "title": "x", "body": "x"}, alice.Token), 404, "clients cannot create notifications")
	e.expect(e.do("GET", "/notifications?limit=0", nil, alice.Token), 400, "invalid limit")
}

func TestRealtimeMessageDelivery(t *testing.T) {
	e := setup(t)
	alice, bruno, matchID := e.matchPair()

	server := httptest.NewServer(e.router)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Unauthenticated connections are closed.
	bad, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "forged"})
	_ = bad.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("forged token must not receive events")
	}
	bad.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": bruno.Token}); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ready realtime.Event
	if err := conn.ReadJSON(&ready); err != nil || ready.Type != "ready" {
		t.Fatalf("expected ready event, got %+v (%v)", ready, err)
	}

	e.expect(e.do("POST", "/matches/"+matchID+"/messages", map[string]string{"content": "en direct"}, alice.Token), 201, "send")
	var evt struct {
		Type    string `json:"type"`
		Payload struct {
			MatchID string `json:"matchId"`
			Content string `json:"content"`
		} `json:"payload"`
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := conn.ReadJSON(&evt); err != nil {
		t.Fatal(err)
	}
	if evt.Type != "message.new" || evt.Payload.Content != "en direct" || evt.Payload.MatchID != matchID {
		t.Fatalf("unexpected event %+v", evt)
	}
}

func TestSwipeCannotBypassRules(t *testing.T) {
	e := setup(t)
	alice := e.register("alice")
	carl := e.register("carl")
	e.onboard(alice, profileOpts{name: "Alice", gender: "woman", interestedIn: []string{"man"}})
	e.onboard(carl, profileOpts{name: "Carl", gender: "man", interestedIn: []string{"man"}})

	// Incompatible preferences: a like sent by id is rejected like discovery.
	if status, _ := e.swipe(carl, alice.ID, "like"); status != 404 {
		t.Fatalf("like must respect mutual preferences, got %d", status)
	}
	// Uppercase ids are normalised (no self-like, no 500).
	if status, _ := e.swipe(alice, strings.ToUpper(alice.ID), "like"); status != 400 {
		t.Fatalf("uppercase self-like must be rejected with 400, got %d", status)
	}

	// Mutual like with uppercase ids still produces exactly one match.
	bruno := e.register("bruno")
	e.onboard(bruno, profileOpts{name: "Bruno", gender: "man", interestedIn: []string{"woman"}})
	if status, _ := e.swipe(alice, strings.ToUpper(bruno.ID), "like"); status != 201 {
		t.Fatalf("uppercase like: %d", status)
	}
	if status, s := e.swipe(bruno, strings.ToUpper(alice.ID), "like"); status != 201 || !s.Matched {
		t.Fatalf("uppercase mutual like must match: %d %+v", status, s)
	}

	// Hiding one's location does not bypass the other person's radius.
	far := e.register("far")
	e.onboard(far, profileOpts{name: "Fanny", gender: "woman", interestedIn: []string{"man"}, lat: 43.2965, lng: 5.3698})
	david := e.register("david")
	e.onboard(david, profileOpts{name: "David", gender: "man", interestedIn: []string{"woman"}, noLocation: true})
	if containsCard(e.discover(david), far.ID) {
		t.Fatal("a viewer without location must not see someone with a limited radius")
	}
	if status, _ := e.swipe(david, far.ID, "like"); status != 404 {
		t.Fatalf("liking outside the other's radius must fail, got %d", status)
	}
}

func TestRefreshTokenReuseRevokesSession(t *testing.T) {
	e := setup(t)
	u := e.register("reuse")
	first := e.do("POST", "/auth/refresh", map[string]string{"refreshToken": u.RefreshToken}, "")
	e.expect(first, 200, "legit refresh")
	var next struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	first.decode(t, &next)
	// Replaying the rotated token (e.g. stolen copy) ends the session.
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": u.RefreshToken}, ""), 401, "replay")
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": next.RefreshToken}, ""), 401, "session revoked after replay")
	e.expect(e.do("GET", "/me", nil, next.AccessToken), 401, "access token revoked too")
}

func TestRateLimitOnLogin(t *testing.T) {
	e := setup(t)
	// Per-account limit: attempts against one e-mail from many IPs.
	accountLimited := false
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"target@lueur.test","password":"Guess12345"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:5555", i+1)
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			accountLimited = true
			break
		}
	}
	if !accountLimited {
		t.Fatal("password guessing against one account must be rate limited")
	}

	// Per-IP limit: many e-mails from one IP.
	limited := false
	for i := 0; i < 40; i++ {
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(fmt.Sprintf(`{"email":"x%d@lueur.test","password":"Password123"}`, i)))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.77:5555"
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			limited = true
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("429 must carry Retry-After")
			}
			break
		}
	}
	if !limited {
		t.Fatal("login brute force must be rate limited")
	}
}
