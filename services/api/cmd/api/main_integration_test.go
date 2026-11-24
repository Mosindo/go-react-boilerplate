package main

import (
	"bytes"
	"context"
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

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

type captureMailer struct {
	mu   sync.Mutex
	sent map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent[to] = body
	return nil
}

func (m *captureMailer) last(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sent[to]
}

type harness struct {
	t         *testing.T
	router    *gin.Engine
	pool      *pgxpool.Pool
	mailer    *captureMailer
	uploadDir string
}

var emailCounter atomic.Int64

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	if os.Getenv("TEST_VERBOSE_LOGS") == "" {
		log.SetOutput(io.Discard)
	}
	os.Exit(m.Run())
}

func setup(t *testing.T) *harness {
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
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	uploadDir := t.TempDir()
	store, err := storage.NewLocal(uploadDir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{sent: map[string]string{}}
	a := &app{
		cfg:    config.Config{Env: "test", JWTSecret: "integration-secret-0123456789abcdefghijkl", UploadDir: uploadDir},
		dbPool: pool,
		store:  store,
		mailer: mail,
		hub:    realtime.NewHub(),
	}
	router, err := setupRouter(a)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, router: router, pool: pool, mailer: mail, uploadDir: uploadDir}
}

type response struct {
	Status int
	Body   []byte
}

func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (h *harness) do(method, path string, payload any, token string) response {
	h.t.Helper()
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			h.t.Fatal(err)
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
	h.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (h *harness) expect(resp response, status int, context string) {
	h.t.Helper()
	if resp.Status != status {
		h.t.Fatalf("%s: expected %d, got %d body=%s", context, status, resp.Status, resp.Body)
	}
}

func (h *harness) upload(method, path, token string, data []byte) response {
	h.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		h.t.Fatal(err)
	}
	part.Write(data)
	w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes()}
}

func pngBytes(t *testing.T, shade uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 400, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{R: shade, G: uint8(x % 255), B: uint8(y % 255), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type user struct {
	ID           string
	Email        string
	Password     string
	AccessToken  string
	RefreshToken string
}

type authPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (h *harness) register(prefix string) user {
	h.t.Helper()
	email := fmt.Sprintf("%s_%d_%d@lueur.test", prefix, time.Now().UnixNano(), emailCounter.Add(1))
	u := user{Email: email, Password: "Password123"}
	resp := h.do(http.MethodPost, "/auth/register", map[string]string{"email": strings.ToUpper(email), "password": u.Password}, "")
	h.expect(resp, http.StatusCreated, "register")
	var payload authPayload
	resp.decode(h.t, &payload)
	u.ID, u.AccessToken, u.RefreshToken = payload.User.ID, payload.AccessToken, payload.RefreshToken
	if payload.User.Email != email {
		h.t.Fatalf("email must be normalised, got %s", payload.User.Email)
	}
	h.t.Cleanup(func() {
		_, _ = h.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID)
	})
	return u
}

type profileSpec struct {
	Name, Gender string
	InterestedIn []string
	Lat, Lng     float64
	Interests    []int
}

// onboard creates a complete, discoverable member.
func (h *harness) onboard(prefix string, spec profileSpec) user {
	h.t.Helper()
	u := h.register(prefix)
	h.expect(h.do(http.MethodPut, "/me/profile", map[string]any{
		"firstName": spec.Name, "birthdate": "1994-05-12", "gender": spec.Gender,
		"bio": "Bonjour", "relationshipGoal": "long_term", "city": "Lyon",
	}, u.AccessToken), http.StatusOK, "profile upsert")
	h.expect(h.do(http.MethodPut, "/me/location", map[string]any{"latitude": spec.Lat, "longitude": spec.Lng}, u.AccessToken), http.StatusOK, "location")
	h.expect(h.do(http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": spec.InterestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50,
	}, u.AccessToken), http.StatusOK, "preferences")
	if len(spec.Interests) > 0 {
		h.expect(h.do(http.MethodPut, "/me/interests", map[string]any{"interestIds": spec.Interests}, u.AccessToken), http.StatusOK, "interests")
	}
	h.expect(h.upload(http.MethodPost, "/me/photos", u.AccessToken, pngBytes(h.t, 120)), http.StatusCreated, "photo upload")
	return u
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Interests  []struct {
		ID     int  `json:"id"`
		Shared bool `json:"shared"`
	} `json:"interests"`
	Photos []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
}

func (h *harness) discover(u user) []card {
	h.t.Helper()
	resp := h.do(http.MethodGet, "/discovery?limit=30", nil, u.AccessToken)
	h.expect(resp, http.StatusOK, "discovery")
	var payload struct {
		Profiles []card `json:"profiles"`
	}
	resp.decode(h.t, &payload)
	return payload.Profiles
}

func containsCard(cards []card, id string) bool {
	for _, c := range cards {
		if c.UserID == id {
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
		User           card   `json:"user"`
	} `json:"match"`
}

func (h *harness) swipe(u user, target, decision string) (response, swipeResult) {
	h.t.Helper()
	resp := h.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "decision": decision}, u.AccessToken)
	var result swipeResult
	if resp.Status == http.StatusCreated {
		resp.decode(h.t, &result)
	}
	return resp, result
}

// Lyon area, a few kilometres apart. Each test uses its own longitude band
// so that members created by other tests do not interfere.
func spot(band int) (float64, float64) {
	return 10 + float64(band)*1.5, 100 + float64(time.Now().UnixNano()%1000)/100
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHealthEndpoint(t *testing.T) {
	h := setup(t)
	resp := h.do(http.MethodGet, "/health", nil, "")
	h.expect(resp, http.StatusOK, "health")
	if !strings.Contains(string(resp.Body), `"status":"ok"`) {
		t.Fatalf("unexpected body %s", resp.Body)
	}
}

func TestUnauthenticatedRequestsAreRejected(t *testing.T) {
	h := setup(t)
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/me"},
		{http.MethodGet, "/me/profile"},
		{http.MethodGet, "/discovery"},
		{http.MethodPost, "/swipes"},
		{http.MethodGet, "/conversations"},
		{http.MethodGet, "/notifications"},
		{http.MethodPost, "/reports"},
		{http.MethodDelete, "/me"},
	} {
		resp := h.do(route.method, route.path, nil, "")
		h.expect(resp, http.StatusUnauthorized, route.method+" "+route.path)
		resp = h.do(route.method, route.path, nil, "not-a-jwt")
		h.expect(resp, http.StatusUnauthorized, route.method+" "+route.path+" (garbage token)")
	}
}

func TestAuthLifecycle(t *testing.T) {
	h := setup(t)
	h.expect(h.do(http.MethodPost, "/auth/register", map[string]string{"email": "weak@lueur.test", "password": "password"}, ""), http.StatusBadRequest, "weak password")
	h.expect(h.do(http.MethodPost, "/auth/register", map[string]string{"email": "not-an-email", "password": "Password123"}, ""), http.StatusBadRequest, "bad email")

	u := h.register("auth")
	h.expect(h.do(http.MethodPost, "/auth/register", map[string]string{"email": u.Email, "password": u.Password}, ""), http.StatusConflict, "duplicate email")
	h.expect(h.do(http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "Wrong12345"}, ""), http.StatusUnauthorized, "wrong password")
	h.expect(h.do(http.MethodPost, "/auth/login", map[string]string{"email": "nobody@lueur.test", "password": "Wrong12345"}, ""), http.StatusUnauthorized, "unknown email")

	resp := h.do(http.MethodPost, "/auth/login", map[string]string{"email": "  " + strings.ToUpper(u.Email), "password": u.Password}, "")
	h.expect(resp, http.StatusOK, "login")
	var login authPayload
	resp.decode(t, &login)

	resp = h.do(http.MethodGet, "/me", nil, login.AccessToken)
	h.expect(resp, http.StatusOK, "me")
	if !strings.Contains(string(resp.Body), u.ID) || strings.Contains(string(resp.Body), "password") {
		t.Fatalf("unexpected /me body %s", resp.Body)
	}

	// Refresh tokens rotate and are single use.
	resp = h.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": login.RefreshToken}, "")
	h.expect(resp, http.StatusOK, "refresh")
	var refreshed authPayload
	resp.decode(t, &refreshed)
	h.expect(h.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": login.RefreshToken}, ""), http.StatusUnauthorized, "refresh reuse")

	// Logout revokes the session immediately, access token included.
	h.expect(h.do(http.MethodPost, "/auth/logout", map[string]string{"refreshToken": refreshed.RefreshToken}, ""), http.StatusNoContent, "logout")
	h.expect(h.do(http.MethodGet, "/me", nil, refreshed.AccessToken), http.StatusUnauthorized, "access after logout")
	h.expect(h.do(http.MethodGet, "/me", nil, u.AccessToken), http.StatusOK, "other session still valid")
}

func TestPasswordRecovery(t *testing.T) {
	h := setup(t)
	u := h.register("recovery")
	h.expect(h.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": "ghost@lueur.test"}, ""), http.StatusAccepted, "forgot unknown email")
	h.expect(h.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": u.Email}, ""), http.StatusAccepted, "forgot")

	code := regexp.MustCompile(`\b\d{8}\b`).FindString(h.mailer.last(u.Email))
	if code == "" {
		t.Fatalf("no reset code in email: %q", h.mailer.last(u.Email))
	}
	wrong := "00000000"
	if wrong == code {
		wrong = "11111111"
	}
	h.expect(h.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": wrong, "password": "NewPassword9"}, ""), http.StatusBadRequest, "wrong code")
	h.expect(h.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "password": "NewPassword9"}, ""), http.StatusNoContent, "reset")
	h.expect(h.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "password": "OtherPass99"}, ""), http.StatusBadRequest, "code reuse")

	h.expect(h.do(http.MethodGet, "/me", nil, u.AccessToken), http.StatusUnauthorized, "sessions revoked after reset")
	h.expect(h.do(http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": u.Password}, ""), http.StatusUnauthorized, "old password")
	h.expect(h.do(http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "NewPassword9"}, ""), http.StatusOK, "new password")
}

func TestProfileValidationAndOnboardingState(t *testing.T) {
	h := setup(t)
	u := h.register("onboarding")

	resp := h.do(http.MethodGet, "/me/profile", nil, u.AccessToken)
	h.expect(resp, http.StatusOK, "empty profile")
	var mine struct {
		Profile    *map[string]any `json:"profile"`
		IsComplete bool            `json:"isComplete"`
		Missing    []string        `json:"missing"`
	}
	resp.decode(t, &mine)
	if mine.Profile != nil || mine.IsComplete || len(mine.Missing) != 3 {
		t.Fatalf("unexpected onboarding state %s", resp.Body)
	}

	minorBirthdate := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	resp = h.do(http.MethodPut, "/me/profile", map[string]any{"firstName": "Léa", "birthdate": minorBirthdate, "gender": "woman"}, u.AccessToken)
	h.expect(resp, http.StatusBadRequest, "underage")
	if !strings.Contains(string(resp.Body), "underage") {
		t.Fatalf("expected underage code, got %s", resp.Body)
	}
	h.expect(h.do(http.MethodPut, "/me/profile", map[string]any{"firstName": "Léa", "birthdate": "1990-01-01", "gender": "dragon"}, u.AccessToken), http.StatusBadRequest, "gender")
	h.expect(h.do(http.MethodPut, "/me/location", map[string]any{"latitude": 45.0, "longitude": 4.0}, u.AccessToken), http.StatusConflict, "location before profile")
	h.expect(h.do(http.MethodGet, "/discovery", nil, u.AccessToken), http.StatusConflict, "discovery before completion")

	h.expect(h.do(http.MethodPut, "/me/profile", map[string]any{"firstName": "Léa", "birthdate": "1990-01-01", "gender": "woman", "bio": "Café & randos"}, u.AccessToken), http.StatusOK, "profile")
	h.expect(h.do(http.MethodPut, "/me/location", map[string]any{"latitude": 45.764043, "longitude": 4.835659, "city": "Lyon"}, u.AccessToken), http.StatusOK, "location")
	h.expect(h.do(http.MethodPut, "/me/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 40, "maxDistanceKm": 20}, u.AccessToken), http.StatusBadRequest, "minor age range")
	h.expect(h.do(http.MethodPut, "/me/interests", map[string]any{"interestIds": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}}, u.AccessToken), http.StatusBadRequest, "too many interests")
	h.expect(h.do(http.MethodPut, "/me/interests", map[string]any{"interestIds": []int{32000}}, u.AccessToken), http.StatusBadRequest, "unknown interest")
	h.expect(h.do(http.MethodPut, "/me/interests", map[string]any{"interestIds": []int{1, 2}}, u.AccessToken), http.StatusOK, "interests")

	var lat, lng float64
	if err := h.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 45.76 || lng != 4.84 {
		t.Fatalf("exact coordinates must never be stored, got %v,%v", lat, lng)
	}

	h.expect(h.upload(http.MethodPost, "/me/photos", u.AccessToken, pngBytes(t, 10)), http.StatusCreated, "photo")
	resp = h.do(http.MethodGet, "/me/profile", nil, u.AccessToken)
	resp.decode(t, &mine)
	if !mine.IsComplete || len(mine.Missing) != 0 {
		t.Fatalf("profile should be complete: %s", resp.Body)
	}
}

func TestPhotoManagementAndProtectedMedia(t *testing.T) {
	h := setup(t)
	u := h.register("photos")
	other := h.register("photos_other")

	h.expect(h.upload(http.MethodPost, "/me/photos", u.AccessToken, []byte("<?php echo 'pwned'; ?>")), http.StatusBadRequest, "script upload")
	h.expect(h.upload(http.MethodPost, "/me/photos", u.AccessToken, []byte("GIF89a....")), http.StatusBadRequest, "gif upload")

	type photo struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Position int    `json:"position"`
	}
	var uploaded []photo
	for i := 0; i < 6; i++ {
		resp := h.upload(http.MethodPost, "/me/photos", u.AccessToken, pngBytes(t, uint8(i*30)))
		h.expect(resp, http.StatusCreated, "upload")
		var p photo
		resp.decode(t, &p)
		if p.Position != i {
			t.Fatalf("expected position %d, got %d", i, p.Position)
		}
		uploaded = append(uploaded, p)
	}
	h.expect(h.upload(http.MethodPost, "/me/photos", u.AccessToken, pngBytes(t, 1)), http.StatusConflict, "7th photo")

	// Signed media URL: works, rejects tampering, needs no bearer token.
	resp := h.do(http.MethodGet, uploaded[0].URL, nil, "")
	h.expect(resp, http.StatusOK, "media")
	if _, format, err := image.DecodeConfig(bytes.NewReader(resp.Body)); err != nil || format != "jpeg" {
		t.Fatalf("media must be a re-encoded jpeg, got %q %v", format, err)
	}
	h.expect(h.do(http.MethodGet, strings.Replace(uploaded[0].URL, "sig=", "sig=00", 1), nil, ""), http.StatusForbidden, "tampered media")
	h.expect(h.do(http.MethodGet, "/media/photos/"+uploaded[0].ID, nil, ""), http.StatusForbidden, "unsigned media")

	// Reorder: last becomes main photo.
	order := []string{uploaded[5].ID, uploaded[0].ID, uploaded[1].ID, uploaded[2].ID, uploaded[3].ID, uploaded[4].ID}
	resp = h.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order}, u.AccessToken)
	h.expect(resp, http.StatusOK, "reorder")
	var list struct {
		Photos []photo `json:"photos"`
	}
	resp.decode(t, &list)
	if list.Photos[0].ID != uploaded[5].ID {
		t.Fatalf("expected reordered main photo, got %+v", list.Photos[0])
	}
	h.expect(h.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order[:3]}, u.AccessToken), http.StatusBadRequest, "partial order")

	// Ownership: another member cannot touch these photos.
	h.expect(h.do(http.MethodDelete, "/me/photos/"+uploaded[1].ID, nil, other.AccessToken), http.StatusNotFound, "foreign delete")
	h.expect(h.upload(http.MethodPut, "/me/photos/"+uploaded[1].ID, other.AccessToken, pngBytes(t, 3)), http.StatusNotFound, "foreign replace")

	// Replace keeps the id and position but changes the file.
	var oldKey string
	h.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, uploaded[1].ID).Scan(&oldKey)
	h.expect(h.upload(http.MethodPut, "/me/photos/"+uploaded[1].ID, u.AccessToken, pngBytes(t, 200)), http.StatusOK, "replace")
	if _, err := os.Stat(filepath.Join(h.uploadDir, oldKey)); !os.IsNotExist(err) {
		t.Fatalf("old file must be deleted after replace (err=%v)", err)
	}

	// Delete compacts positions.
	h.expect(h.do(http.MethodDelete, "/me/photos/"+uploaded[5].ID, nil, u.AccessToken), http.StatusNoContent, "delete")
	resp = h.do(http.MethodGet, "/me/photos", nil, u.AccessToken)
	resp.decode(t, &list)
	if len(list.Photos) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(list.Photos))
	}
	for i, p := range list.Photos {
		if p.Position != i {
			t.Fatalf("positions must stay dense: %+v", list.Photos)
		}
	}
	if list.Photos[0].ID != uploaded[0].ID {
		t.Fatal("next photo must become the main photo")
	}
}

func TestDiscoveryLikeMatchAndChatJourney(t *testing.T) {
	h := setup(t)
	lat, lng := spot(1)
	alice := h.onboard("alice", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng, Interests: []int{1, 2, 3}})
	bob := h.onboard("bob", profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat + 0.03, Lng: lng, Interests: []int{2, 3, 4}})
	carl := h.onboard("carl", profileSpec{Name: "Carl", Gender: "man", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	far := h.onboard("far", profileSpec{Name: "Dan", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat + 3, Lng: lng})

	// Discovery honours both sides' preferences and distance.
	cards := h.discover(alice)
	if !containsCard(cards, bob.ID) {
		t.Fatalf("alice should discover bob: %+v", cards)
	}
	if containsCard(cards, carl.ID) || containsCard(cards, far.ID) || containsCard(cards, alice.ID) {
		t.Fatalf("incompatible, distant or self profiles leaked: %+v", cards)
	}
	for _, c := range cards {
		if c.UserID == bob.ID {
			if c.DistanceKm == nil || *c.DistanceKm < 1 || *c.DistanceKm > 6 {
				t.Fatalf("expected approximate distance, got %v", c.DistanceKm)
			}
			shared := 0
			for _, i := range c.Interests {
				if i.Shared {
					shared++
				}
			}
			if shared != 2 || len(c.Photos) != 1 {
				t.Fatalf("unexpected card details %+v", c)
			}
		}
	}

	// Public profile never exposes private fields.
	resp := h.do(http.MethodGet, "/profiles/"+bob.ID, nil, alice.AccessToken)
	h.expect(resp, http.StatusOK, "profile view")
	for _, forbidden := range []string{"email", "birthdate", "latitude", "longitude", bob.Email} {
		if strings.Contains(string(resp.Body), forbidden) {
			t.Fatalf("public profile leaks %q: %s", forbidden, resp.Body)
		}
	}

	// Like / pass rules.
	resp, result := h.swipe(alice, bob.ID, "like")
	h.expect(resp, http.StatusCreated, "alice likes bob")
	if result.Matched {
		t.Fatal("one-sided like must not match")
	}
	resp, _ = h.swipe(alice, bob.ID, "like")
	h.expect(resp, http.StatusConflict, "duplicate like")
	resp, _ = h.swipe(alice, bob.ID, "pass")
	h.expect(resp, http.StatusConflict, "changing decision")
	resp, _ = h.swipe(alice, alice.ID, "like")
	h.expect(resp, http.StatusBadRequest, "self like")
	if containsCard(h.discover(alice), bob.ID) {
		t.Fatal("swiped profiles must not reappear")
	}
	resp, _ = h.swipe(bob, far.ID, "pass")
	h.expect(resp, http.StatusCreated, "pass")

	resp, result = h.swipe(bob, alice.ID, "like")
	h.expect(resp, http.StatusCreated, "bob likes alice")
	if !result.Matched || result.Match == nil || result.Match.ConversationID == "" || result.Match.User.UserID != alice.ID {
		t.Fatalf("mutual like must match: %s", resp.Body)
	}
	conversationID := result.Match.ConversationID

	var matches int
	h.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low_id IN ($1::uuid, $2::uuid) AND user_high_id IN ($1::uuid, $2::uuid)`, alice.ID, bob.ID).Scan(&matches)
	if matches != 1 {
		t.Fatalf("expected exactly one match, got %d", matches)
	}

	// Both receive a match notification.
	for _, u := range []user{alice, bob} {
		resp := h.do(http.MethodGet, "/notifications", nil, u.AccessToken)
		h.expect(resp, http.StatusOK, "notifications")
		var payload struct {
			Notifications []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		resp.decode(t, &payload)
		if len(payload.Notifications) != 1 || payload.Notifications[0].Type != "match" || payload.UnreadCount != 1 {
			t.Fatalf("expected one unread match notification: %s", resp.Body)
		}
		h.expect(h.do(http.MethodPost, "/notifications/"+payload.Notifications[0].ID+"/read", nil, u.AccessToken), http.StatusNoContent, "mark read")
		h.expect(h.do(http.MethodPost, "/notifications/"+payload.Notifications[0].ID+"/read", nil, carl.AccessToken), http.StatusNotFound, "foreign notification")
	}

	// Chat.
	h.expect(h.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"body": "   "}, alice.AccessToken), http.StatusBadRequest, "empty message")
	resp = h.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"body": "Salut Bob !"}, alice.AccessToken)
	h.expect(resp, http.StatusCreated, "send")
	h.expect(h.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"body": "Ça va ?"}, alice.AccessToken), http.StatusCreated, "send 2")

	type convList struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body   string `json:"body"`
				IsRead bool   `json:"isRead"`
			} `json:"lastMessage"`
			OtherUser struct {
				UserID    string `json:"userId"`
				FirstName string `json:"firstName"`
				Photo     *struct {
					URL string `json:"url"`
				} `json:"photo"`
			} `json:"otherUser"`
		} `json:"conversations"`
	}
	resp = h.do(http.MethodGet, "/conversations", nil, bob.AccessToken)
	h.expect(resp, http.StatusOK, "bob conversations")
	var bobConvs convList
	resp.decode(t, &bobConvs)
	if len(bobConvs.Conversations) != 1 || bobConvs.Conversations[0].UnreadCount != 2 || bobConvs.Conversations[0].OtherUser.FirstName != "Alice" || bobConvs.Conversations[0].OtherUser.Photo == nil {
		t.Fatalf("unexpected conversation list: %s", resp.Body)
	}

	// Strangers can neither read nor write nor mark read.
	h.expect(h.do(http.MethodGet, "/conversations/"+conversationID+"/messages", nil, carl.AccessToken), http.StatusNotFound, "stranger read")
	h.expect(h.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"body": "intrusion"}, carl.AccessToken), http.StatusNotFound, "stranger write")
	h.expect(h.do(http.MethodPost, "/conversations/"+conversationID+"/read", nil, carl.AccessToken), http.StatusNotFound, "stranger mark read")

	resp = h.do(http.MethodGet, "/conversations/"+conversationID+"/messages?limit=1", nil, bob.AccessToken)
	h.expect(resp, http.StatusOK, "bob messages")
	var page struct {
		Messages []struct {
			ID     string `json:"id"`
			Body   string `json:"body"`
			IsRead bool   `json:"isRead"`
		} `json:"messages"`
		NextCursor *string `json:"nextCursor"`
	}
	resp.decode(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Ça va ?" || page.NextCursor == nil {
		t.Fatalf("expected newest message first with cursor: %s", resp.Body)
	}
	resp = h.do(http.MethodGet, "/conversations/"+conversationID+"/messages?limit=1&before="+*page.NextCursor, nil, bob.AccessToken)
	resp.decode(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Salut Bob !" {
		t.Fatalf("unexpected second page: %s", resp.Body)
	}

	h.expect(h.do(http.MethodPost, "/conversations/"+conversationID+"/read", nil, bob.AccessToken), http.StatusNoContent, "mark read")
	resp = h.do(http.MethodGet, "/conversations/"+conversationID+"/messages", nil, alice.AccessToken)
	resp.decode(t, &page)
	for _, m := range page.Messages {
		if !m.IsRead {
			t.Fatalf("alice's messages must be read by bob: %s", resp.Body)
		}
	}

	// Local deletion only hides for the caller, and reappears on new message.
	h.expect(h.do(http.MethodDelete, "/conversations/"+conversationID, nil, bob.AccessToken), http.StatusNoContent, "hide")
	resp = h.do(http.MethodGet, "/conversations", nil, bob.AccessToken)
	resp.decode(t, &bobConvs)
	if len(bobConvs.Conversations) != 0 {
		t.Fatalf("hidden conversation must disappear: %s", resp.Body)
	}
	var aliceConvs convList
	h.do(http.MethodGet, "/conversations", nil, alice.AccessToken).decode(t, &aliceConvs)
	if len(aliceConvs.Conversations) != 1 {
		t.Fatal("hiding must not affect the other member")
	}
	h.expect(h.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"body": "Tu es là ?"}, alice.AccessToken), http.StatusCreated, "send after hide")
	h.do(http.MethodGet, "/conversations", nil, bob.AccessToken).decode(t, &bobConvs)
	if len(bobConvs.Conversations) != 1 || bobConvs.Conversations[0].UnreadCount != 1 {
		t.Fatalf("conversation must reappear with only new messages: %+v", bobConvs)
	}

	// Unmatch removes the conversation for both.
	h.expect(h.do(http.MethodDelete, "/matches/"+result.Match.ID, nil, carl.AccessToken), http.StatusNotFound, "stranger unmatch")
	h.expect(h.do(http.MethodDelete, "/matches/"+result.Match.ID, nil, alice.AccessToken), http.StatusNoContent, "unmatch")
	h.expect(h.do(http.MethodGet, "/conversations/"+conversationID+"/messages", nil, bob.AccessToken), http.StatusNotFound, "conversation gone")
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	h := setup(t)
	lat, lng := spot(2)
	for round := 0; round < 3; round++ {
		a := h.onboard("race_a", profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
		b := h.onboard("race_b", profileSpec{Name: "Bea", Gender: "woman", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to user) {
				defer wg.Done()
				_, results[i] = h.swipe(from, to.ID, "like")
			}(i, pair[0], pair[1])
		}
		wg.Wait()
		if results[0].Matched == results[1].Matched {
			t.Fatalf("exactly one of the concurrent likes must report the match: %+v", results)
		}
		var count int
		h.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)`, a.ID, b.ID).Scan(&count)
		if count != 1 {
			t.Fatalf("expected one match, got %d", count)
		}
	}
}

func TestBlockAndReport(t *testing.T) {
	h := setup(t)
	lat, lng := spot(3)
	eve := h.onboard("eve", profileSpec{Name: "Eve", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	finn := h.onboard("finn", profileSpec{Name: "Finn", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	gus := h.onboard("gus", profileSpec{Name: "Gus", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})

	h.swipe(eve, finn.ID, "like")
	_, result := h.swipe(finn, eve.ID, "like")
	if !result.Matched {
		t.Fatal("expected match")
	}

	h.expect(h.do(http.MethodPost, "/blocks", map[string]string{"userId": eve.ID}, eve.AccessToken), http.StatusBadRequest, "self block")
	h.expect(h.do(http.MethodPost, "/blocks", map[string]string{"userId": finn.ID}, eve.AccessToken), http.StatusNoContent, "block")
	h.expect(h.do(http.MethodPost, "/blocks", map[string]string{"userId": finn.ID}, eve.AccessToken), http.StatusNoContent, "block is idempotent")
	h.expect(h.do(http.MethodGet, "/conversations/"+result.Match.ConversationID+"/messages", nil, finn.AccessToken), http.StatusNotFound, "blocked conversation")
	h.expect(h.do(http.MethodPost, "/conversations/"+result.Match.ConversationID+"/messages", map[string]string{"body": "hey"}, finn.AccessToken), http.StatusNotFound, "blocked send")
	h.expect(h.do(http.MethodGet, "/profiles/"+eve.ID, nil, finn.AccessToken), http.StatusNotFound, "blocked profile (blocked side)")
	h.expect(h.do(http.MethodGet, "/profiles/"+finn.ID, nil, eve.AccessToken), http.StatusNotFound, "blocked profile (blocker side)")

	resp := h.do(http.MethodGet, "/blocks", nil, eve.AccessToken)
	if !strings.Contains(string(resp.Body), finn.ID) {
		t.Fatalf("block list should contain finn: %s", resp.Body)
	}
	h.expect(h.do(http.MethodDelete, "/blocks/"+finn.ID, nil, eve.AccessToken), http.StatusNoContent, "unblock")
	h.expect(h.do(http.MethodDelete, "/blocks/"+finn.ID, nil, eve.AccessToken), http.StatusNotFound, "unblock twice")

	// Reporting validates input and blocks by default.
	h.expect(h.do(http.MethodPost, "/reports", map[string]any{"userId": gus.ID, "reason": "because"}, eve.AccessToken), http.StatusBadRequest, "bad reason")
	resp = h.do(http.MethodPost, "/reports", map[string]any{"userId": gus.ID, "reason": "harassment", "details": "messages insistants"}, eve.AccessToken)
	h.expect(resp, http.StatusCreated, "report")
	if !strings.Contains(string(resp.Body), `"blocked":true`) {
		t.Fatalf("report should block: %s", resp.Body)
	}
	h.expect(h.do(http.MethodPost, "/reports", map[string]any{"userId": gus.ID, "reason": "spam"}, eve.AccessToken), http.StatusCreated, "report again is idempotent")
	var reports int
	h.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_user_id = $2`, eve.ID, gus.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("expected one open report, got %d", reports)
	}
	if containsCard(h.discover(eve), gus.ID) {
		t.Fatal("reported member must disappear from discovery")
	}
	resp, _ = h.swipe(gus, eve.ID, "like")
	h.expect(resp, http.StatusNotFound, "blocked member cannot like")
}

func TestDeleteAccountErasesData(t *testing.T) {
	h := setup(t)
	lat, lng := spot(4)
	hana := h.onboard("hana", profileSpec{Name: "Hana", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	ivan := h.onboard("ivan", profileSpec{Name: "Ivan", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	h.swipe(hana, ivan.ID, "like")
	_, result := h.swipe(ivan, hana.ID, "like")
	h.do(http.MethodPost, "/conversations/"+result.Match.ConversationID+"/messages", map[string]string{"body": "Coucou"}, hana.AccessToken)

	var key string
	h.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, hana.ID).Scan(&key)

	h.expect(h.do(http.MethodDelete, "/me", map[string]string{"password": "Nope12345"}, hana.AccessToken), http.StatusUnauthorized, "delete with wrong password")
	h.expect(h.do(http.MethodDelete, "/me", map[string]string{"password": hana.Password}, hana.AccessToken), http.StatusNoContent, "delete")

	h.expect(h.do(http.MethodGet, "/me", nil, hana.AccessToken), http.StatusUnauthorized, "token after deletion")
	h.expect(h.do(http.MethodPost, "/auth/login", map[string]string{"email": hana.Email, "password": hana.Password}, ""), http.StatusUnauthorized, "login after deletion")
	if _, err := os.Stat(filepath.Join(h.uploadDir, key)); !os.IsNotExist(err) {
		t.Fatalf("photo file must be erased (err=%v)", err)
	}
	for _, table := range []string{"profiles", "photos", "swipes", "notifications", "sessions"} {
		var n int
		column := "user_id"
		if table == "swipes" {
			column = "swiper_id"
		}
		h.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM `+table+` WHERE `+column+` = $1`, hana.ID).Scan(&n)
		if n != 0 {
			t.Fatalf("%s still holds %d rows", table, n)
		}
	}
	var messages int
	h.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`, result.Match.ConversationID).Scan(&messages)
	if messages != 0 {
		t.Fatal("conversation messages must be erased with the match")
	}
	var convs struct {
		Conversations []any `json:"conversations"`
	}
	h.do(http.MethodGet, "/conversations", nil, ivan.AccessToken).decode(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("the other member must no longer see the conversation")
	}
}

func TestRealtimeDeliversMessages(t *testing.T) {
	h := setup(t)
	server := httptest.NewServer(h.router)
	defer server.Close()
	lat, lng := spot(5)
	jade := h.onboard("jade", profileSpec{Name: "Jade", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	kai := h.onboard("kai", profileSpec{Name: "Kai", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})

	resp := h.do(http.MethodPost, "/realtime/ticket", nil, kai.AccessToken)
	h.expect(resp, http.StatusOK, "ticket")
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	resp.decode(t, &ticket)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime?ticket="
	if _, _, err := websocket.DefaultDialer.Dial(wsURL+"forged", nil); err == nil {
		t.Fatal("forged ticket must be rejected")
	}
	if _, _, err := websocket.DefaultDialer.Dial(wsURL+kai.AccessToken, nil); err == nil {
		t.Fatal("access tokens must not be accepted as realtime tickets")
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+ticket.Ticket, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	readEvent := func(want string) map[string]any {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for {
			conn.SetReadDeadline(deadline)
			_, data, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("waiting for %s: %v", want, err)
			}
			var event map[string]any
			json.Unmarshal(data, &event)
			if event["type"] == want {
				return event
			}
		}
	}
	readEvent("ready")

	h.swipe(kai, jade.ID, "like")
	_, result := h.swipe(jade, kai.ID, "like")
	readEvent("match.created")
	readEvent("notification.created")

	h.do(http.MethodPost, "/conversations/"+result.Match.ConversationID+"/messages", map[string]string{"body": "En temps réel !"}, jade.AccessToken)
	event := readEvent("message.created")
	data, _ := event["data"].(map[string]any)
	if data["body"] != "En temps réel !" || data["conversationId"] != result.Match.ConversationID {
		t.Fatalf("unexpected realtime payload %v", event)
	}
}
