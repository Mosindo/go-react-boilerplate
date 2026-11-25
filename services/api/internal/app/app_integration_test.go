package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
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
	"example.com/api/internal/platform/middleware"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests exercise the real HTTP API against PostgreSQL. They run only
// when DATABASE_URL_TEST points to a disposable database: every table is
// truncated at start.

func init() {
	gin.SetMode(gin.TestMode)
}

type captureMailer struct {
	mu   sync.Mutex
	last map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last[to] = body
	return nil
}

func (m *captureMailer) code(t *testing.T, to string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	match := regexp.MustCompile(`[A-Z0-9]{5}-[A-Z0-9]{5}`).FindString(m.last[to])
	if match == "" {
		t.Fatalf("no reset code sent to %s", to)
	}
	return match
}

type env struct {
	t       *testing.T
	server  *httptest.Server
	pool    *pgxpool.Pool
	mail    *captureMailer
	uploads string
}

func setup(t *testing.T) *env {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		t.Skip("DATABASE_URL_TEST not set: skipping API integration tests")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, reports CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	uploads := t.TempDir()
	cfg := config.Config{Env: config.EnvDevelopment, JWTSecret: "integration-secret-0123456789abcdefghij", UploadDir: uploads}
	limits := Limits{
		Auth:     middleware.NewRateLimiter(10_000, time.Minute),
		Recovery: middleware.NewRateLimiter(10_000, time.Minute),
		Upload:   middleware.NewRateLimiter(10_000, time.Minute),
		Swipe:    middleware.NewRateLimiter(10_000, time.Minute),
		Message:  middleware.NewRateLimiter(10_000, time.Minute),
		Report:   middleware.NewRateLimiter(10_000, time.Minute),
	}
	mail := &captureMailer{last: map[string]string{}}
	a, err := New(cfg, pool, Options{Limits: limits, Mailer: mail})
	if err != nil {
		t.Fatalf("app: %v", err)
	}
	server := httptest.NewServer(a.Engine)
	t.Cleanup(func() {
		server.Close()
		pool.Close()
	})
	return &env{t: t, server: server, pool: pool, mail: mail, uploads: uploads}
}

type resp struct {
	status int
	body   []byte
}

func (r resp) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.body, dst); err != nil {
		t.Fatalf("decode %s: %v", string(r.body), err)
	}
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, e.server.URL+path, reader)
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
	data, _ := io.ReadAll(res.Body)
	return resp{status: res.StatusCode, body: data}
}

func (e *env) expect(r resp, status int, context string) {
	e.t.Helper()
	if r.status != status {
		e.t.Fatalf("%s: expected %d, got %d: %s", context, status, r.status, string(r.body))
	}
}

func (e *env) upload(method, path, token string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", "photo.png")
	part.Write(data)
	w.Close()
	req, _ := http.NewRequest(method, e.server.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("upload: %v", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return resp{status: res.StatusCode, body: body}
}

func testImage(t *testing.T, shade uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 400, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{shade, uint8(x / 2), uint8(y / 2), 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

type user struct {
	id      string
	email   string
	access  string
	refresh string
}

type authBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

var emailSeq int
var emailMu sync.Mutex

func (e *env) register(prefix string) user {
	e.t.Helper()
	emailMu.Lock()
	emailSeq++
	email := fmt.Sprintf("%s%d@example.test", prefix, emailSeq)
	emailMu.Unlock()
	r := e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"})
	e.expect(r, http.StatusCreated, "register")
	var b authBody
	r.decode(e.t, &b)
	return user{id: b.User.ID, email: email, access: b.AccessToken, refresh: b.RefreshToken}
}

type profileSpec struct {
	name, gender, birthdate string
	interestedIn            []string
	lat, lon                float64
	interests               []int
}

// onboard creates a complete profile (profile, preferences, photo, location).
func (e *env) onboard(u user, p profileSpec) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", u.access, map[string]any{
		"firstName": p.name, "birthdate": p.birthdate, "gender": p.gender, "bio": "Bonjour", "interestIds": p.interests,
	}), http.StatusOK, "profile")
	e.expect(e.do("PUT", "/me/preferences", u.access, map[string]any{
		"interestedIn": p.interestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": 100,
	}), http.StatusOK, "preferences")
	e.expect(e.upload("POST", "/me/photos", u.access, testImage(e.t, 120)), http.StatusCreated, "photo")
	e.expect(e.do("PUT", "/me/location", u.access, map[string]any{"latitude": p.lat, "longitude": p.lon}), http.StatusOK, "location")
}

type publicProfile struct {
	UserID            string  `json:"userId"`
	FirstName         string  `json:"firstName"`
	Age               int     `json:"age"`
	DistanceKm        *int    `json:"distanceKm"`
	CommonInterestIDs []int   `json:"commonInterestIds"`
	Photos            []photo `json:"photos"`
}

type photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

func (e *env) discover(u user) []publicProfile {
	e.t.Helper()
	r := e.do("GET", "/discover?limit=30", u.access, nil)
	e.expect(r, http.StatusOK, "discover")
	var body struct {
		Profiles []publicProfile `json:"profiles"`
	}
	r.decode(e.t, &body)
	return body.Profiles
}

func contains(list []publicProfile, id string) bool {
	for _, p := range list {
		if p.UserID == id {
			return true
		}
	}
	return false
}

type swipeBody struct {
	Action        string `json:"action"`
	AlreadySwiped bool   `json:"alreadySwiped"`
	Match         *struct {
		MatchID        string `json:"matchId"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (e *env) swipe(u user, target, action string, status int) swipeBody {
	e.t.Helper()
	r := e.do("POST", "/swipes", u.access, map[string]string{"targetUserId": target, "action": action})
	e.expect(r, status, "swipe "+action)
	var b swipeBody
	if status == http.StatusOK {
		r.decode(e.t, &b)
	}
	return b
}

func TestAuthLifecycle(t *testing.T) {
	e := setup(t)

	e.expect(e.do("GET", "/health", "", nil), http.StatusOK, "health")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "bad", "password": "Password123"}), http.StatusBadRequest, "invalid email")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "weak@example.test", "password": "short"}), http.StatusBadRequest, "weak password")

	u := e.register("auth")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": strings.ToUpper(u.email), "password": "Password123"}), http.StatusConflict, "duplicate email (case-insensitive)")

	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized, "me without token")
	e.expect(e.do("GET", "/me", "garbage", nil), http.StatusUnauthorized, "me with garbage token")
	e.expect(e.do("GET", "/me", u.access, nil), http.StatusOK, "me")

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.email, "password": "WrongPass1"}), http.StatusUnauthorized, "wrong password")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@example.test", "password": "Password123"}), http.StatusUnauthorized, "unknown email")
	login := e.do("POST", "/auth/login", "", map[string]string{"email": "  " + strings.ToUpper(u.email) + " ", "password": "Password123"})
	e.expect(login, http.StatusOK, "login with normalised email")

	// Refresh rotation: the old refresh token cannot be reused.
	refreshed := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.refresh})
	e.expect(refreshed, http.StatusOK, "refresh")
	var rb authBody
	refreshed.decode(t, &rb)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.refresh}), http.StatusUnauthorized, "refresh token reuse")

	// Logout revokes the session: its access token stops working at once.
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": rb.RefreshToken}), http.StatusNoContent, "logout")
	e.expect(e.do("GET", "/me", rb.AccessToken, nil), http.StatusUnauthorized, "access token after logout")

	// Password recovery never reveals whether an email exists.
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "ghost@example.test"}), http.StatusAccepted, "forgot unknown")
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.email}), http.StatusAccepted, "forgot")
	code := e.mail.code(t, u.email)
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"code": "AAAAA-BBBBB", "password": "NewPassword9"}), http.StatusBadRequest, "reset wrong code")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"code": strings.ToLower(code), "password": "NewPassword9"}), http.StatusNoContent, "reset")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"code": code, "password": "OtherPass99"}), http.StatusBadRequest, "reset code reuse")
	var loginBody authBody
	login.decode(t, &loginBody)
	e.expect(e.do("GET", "/me", loginBody.AccessToken, nil), http.StatusUnauthorized, "sessions revoked after reset")
	newLogin := e.do("POST", "/auth/login", "", map[string]string{"email": u.email, "password": "NewPassword9"})
	e.expect(newLogin, http.StatusOK, "login with new password")
	var nb authBody
	newLogin.decode(t, &nb)

	// Change password keeps the current session only.
	e.expect(e.do("PUT", "/me/password", nb.AccessToken, map[string]string{"currentPassword": "nope", "newPassword": "Another123"}), http.StatusForbidden, "change password wrong current")
	e.expect(e.do("PUT", "/me/password", nb.AccessToken, map[string]string{"currentPassword": "NewPassword9", "newPassword": "Another123"}), http.StatusNoContent, "change password")
	e.expect(e.do("GET", "/me", nb.AccessToken, nil), http.StatusOK, "current session kept")
}

func TestProfileValidationAndPhotos(t *testing.T) {
	e := setup(t)
	u := e.register("profile")

	r := e.do("GET", "/me/profile", u.access, nil)
	e.expect(r, http.StatusOK, "empty profile")
	var me struct {
		Profile      *struct{ FirstName string } `json:"profile"`
		Completeness struct {
			IsComplete bool     `json:"isComplete"`
			Missing    []string `json:"missing"`
		} `json:"completeness"`
	}
	r.decode(t, &me)
	if me.Profile != nil || me.Completeness.IsComplete || len(me.Completeness.Missing) != 4 {
		t.Fatalf("unexpected empty profile state: %s", string(r.body))
	}

	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", u.access, map[string]any{"firstName": "Léa", "birthdate": minor, "gender": "woman"}), http.StatusBadRequest, "underage")
	e.expect(e.do("PUT", "/me/profile", u.access, map[string]any{"firstName": "Léa", "birthdate": "1995-02-30", "gender": "woman"}), http.StatusBadRequest, "invalid date")
	e.expect(e.do("PUT", "/me/profile", u.access, map[string]any{"firstName": "Léa", "birthdate": "1995-02-03", "gender": "woman", "interestIds": []int{9999}}), http.StatusBadRequest, "unknown interest")
	e.expect(e.do("PUT", "/me/location", u.access, map[string]any{"latitude": 48.85, "longitude": 2.35}), http.StatusConflict, "location before profile")
	e.expect(e.do("PUT", "/me/profile", u.access, map[string]any{"firstName": "Léa", "birthdate": "1995-02-03", "gender": "woman", "interestIds": []int{1, 2, 3}}), http.StatusOK, "profile")
	e.expect(e.do("PUT", "/me/preferences", u.access, map[string]any{"interestedIn": []string{}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50}), http.StatusBadRequest, "empty preferences")

	// Exact coordinates are coarsened before storage.
	e.expect(e.do("PUT", "/me/location", u.access, map[string]any{"latitude": 48.856614, "longitude": 2.3522219}), http.StatusOK, "location")
	var lat, lon float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.id).Scan(&lat, &lon); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lon != 2.35 {
		t.Fatalf("location was not coarsened: %v %v", lat, lon)
	}

	// Uploads: type validation, limit of 6, reorder, replace, delete.
	e.expect(e.upload("POST", "/me/photos", u.access, []byte("GIF89a not really")), http.StatusBadRequest, "invalid upload")
	var ids []string
	for i := 0; i < 6; i++ {
		r := e.upload("POST", "/me/photos", u.access, testImage(t, uint8(i*30)))
		e.expect(r, http.StatusCreated, "upload")
		var p photo
		r.decode(t, &p)
		if p.Position != i {
			t.Fatalf("expected position %d, got %d", i, p.Position)
		}
		ids = append(ids, p.ID)
	}
	e.expect(e.upload("POST", "/me/photos", u.access, testImage(t, 1)), http.StatusConflict, "seventh photo")

	reversed := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	r = e.do("PUT", "/me/photos/order", u.access, map[string]any{"photoIds": reversed})
	e.expect(r, http.StatusOK, "reorder")
	var list struct {
		Photos []photo `json:"photos"`
	}
	r.decode(t, &list)
	if list.Photos[0].ID != ids[5] {
		t.Fatalf("reorder not applied")
	}
	e.expect(e.do("PUT", "/me/photos/order", u.access, map[string]any{"photoIds": ids[:3]}), http.StatusBadRequest, "partial reorder")

	e.expect(e.upload("PUT", "/me/photos/"+ids[0], u.access, testImage(t, 200)), http.StatusOK, "replace")
	e.expect(e.do("DELETE", "/me/photos/"+ids[5], u.access, nil), http.StatusNoContent, "delete primary")

	r = e.do("GET", "/me/photos", u.access, nil)
	r.decode(t, &list)
	if len(list.Photos) != 5 || list.Photos[0].ID != ids[4] || list.Photos[4].Position != 4 {
		t.Fatalf("positions not compacted: %s", string(r.body))
	}
	files, _ := filepath.Glob(filepath.Join(e.uploads, "*.jpg"))
	if len(files) != 5 {
		t.Fatalf("expected 5 stored files, got %d", len(files))
	}

	// Signed photo URLs: valid signature serves a JPEG, tampering is refused.
	res, err := http.Get(e.server.URL + list.Photos[0].URL)
	if err != nil || res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("signed photo url failed: %v %v", err, res)
	}
	res.Body.Close()
	res, _ = http.Get(e.server.URL + list.Photos[0].URL + "x")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("tampered signature accepted: %d", res.StatusCode)
	}
	res.Body.Close()

	// Another user cannot delete my photos.
	other := e.register("intruder")
	e.expect(e.do("DELETE", "/me/photos/"+ids[4], other.access, nil), http.StatusNotFound, "foreign photo delete")
}

func TestDiscoveryMatchingChatAndSafety(t *testing.T) {
	e := setup(t)

	alice := e.register("alice")
	bob := e.register("bob")
	carol := e.register("carol")
	dan := e.register("dan")
	far := e.register("far")

	e.onboard(alice, profileSpec{"Alice", "woman", "1994-05-01", []string{"man"}, 48.8566, 2.3522, []int{1, 2, 3}})
	e.onboard(bob, profileSpec{"Bob", "man", "1992-03-10", []string{"woman"}, 48.86, 2.36, []int{2, 3, 4}})
	e.onboard(carol, profileSpec{"Carol", "woman", "1990-01-01", []string{"woman"}, 48.85, 2.35, nil})
	e.onboard(dan, profileSpec{"Dan", "man", "1991-07-07", []string{"woman"}, 48.87, 2.34, nil})
	e.onboard(far, profileSpec{"Far", "man", "1991-07-07", []string{"woman"}, 43.30, 5.37, nil}) // Marseille, > 100 km

	incomplete := e.register("incomplete")
	e.expect(e.do("GET", "/discover", incomplete.access, nil), http.StatusConflict, "discover with incomplete profile")

	// Mutual filters: Alice (woman→man) sees Bob and Dan, not Carol (woman→woman) nor Far (too far).
	deck := e.discover(alice)
	if !contains(deck, bob.id) || !contains(deck, dan.id) || contains(deck, carol.id) || contains(deck, far.id) || contains(deck, alice.id) {
		t.Fatalf("unexpected deck for alice: %+v", deck)
	}
	for _, p := range deck {
		if p.UserID == bob.id {
			if p.DistanceKm == nil || *p.DistanceKm < 1 || len(p.Photos) != 1 || p.Age < 18 {
				t.Fatalf("unexpected bob card: %+v", p)
			}
			if len(p.CommonInterestIDs) != 2 {
				t.Fatalf("expected 2 common interests, got %v", p.CommonInterestIDs)
			}
		}
	}
	if deck[0].UserID != bob.id {
		t.Fatalf("bob shares more interests and should rank first, got %s", deck[0].FirstName)
	}
	bobDeck := e.discover(bob)
	if !contains(bobDeck, alice.id) || contains(bobDeck, carol.id) {
		t.Fatalf("unexpected deck for bob")
	}

	// Validation of swipes.
	e.swipe(alice, alice.id, "like", http.StatusBadRequest)
	e.swipe(alice, bob.id, "superlike", http.StatusBadRequest)
	e.swipe(alice, "00000000-0000-0000-0000-000000000000", "like", http.StatusNotFound)

	// A like without reciprocity: no match; processed profiles leave the deck.
	first := e.swipe(alice, bob.id, "like", http.StatusOK)
	if first.Match != nil {
		t.Fatal("unexpected match on one-sided like")
	}
	again := e.swipe(alice, bob.id, "like", http.StatusOK)
	if !again.AlreadySwiped {
		t.Fatal("repeated like must be idempotent")
	}
	e.swipe(alice, bob.id, "pass", http.StatusConflict)
	if contains(e.discover(alice), bob.id) {
		t.Fatal("liked profile reappeared in the deck")
	}

	// Pass: Dan disappears for Alice, and Alice no longer sees Dan... and Dan no longer sees Alice.
	e.swipe(alice, dan.id, "pass", http.StatusOK)
	if contains(e.discover(alice), dan.id) || contains(e.discover(dan), alice.id) {
		t.Fatal("passed profile reappeared")
	}

	// Mutual like → exactly one match and one conversation.
	match := e.swipe(bob, alice.id, "like", http.StatusOK)
	if match.Match == nil || match.Match.ConversationID == "" {
		t.Fatalf("expected a match: %+v", match)
	}
	var matches int
	e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches`).Scan(&matches)
	if matches != 1 {
		t.Fatalf("expected 1 match, got %d", matches)
	}
	convID := match.Match.ConversationID

	// Both users receive a match notification.
	for _, u := range []user{alice, bob} {
		r := e.do("GET", "/notifications", u.access, nil)
		e.expect(r, http.StatusOK, "notifications")
		var nb struct {
			Notifications []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		r.decode(t, &nb)
		if len(nb.Notifications) != 1 || nb.Notifications[0].Type != "match" || nb.UnreadCount != 1 {
			t.Fatalf("unexpected notifications: %s", string(r.body))
		}
		e.expect(e.do("POST", "/notifications/"+nb.Notifications[0].ID+"/read", u.access, nil), http.StatusOK, "mark read")
		e.expect(e.do("POST", "/notifications/"+nb.Notifications[0].ID+"/read", carol.access, nil), http.StatusNotFound, "mark someone else's notification")
	}
	e.expect(e.do("POST", "/notifications", alice.access, map[string]string{"title": "fake"}), http.StatusNotFound, "clients cannot create notifications")

	// Realtime: Alice listens, Bob sends a message.
	ticketResp := e.do("POST", "/realtime/ticket", alice.access, nil)
	e.expect(ticketResp, http.StatusOK, "ticket")
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	ticketResp.decode(t, &ticket)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/realtime?ticket=" + ticket.Ticket
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.CloseNow()
	if _, _, err := websocket.Dial(ctx, wsURL, nil); err == nil {
		t.Fatal("ticket must be single-use")
	}
	readEvent := func(want string) map[string]any {
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				t.Fatalf("ws read: %v", err)
			}
			var ev struct {
				Type string         `json:"type"`
				Data map[string]any `json:"data"`
			}
			json.Unmarshal(data, &ev)
			if ev.Type == want {
				return ev.Data
			}
		}
	}
	readEvent("ready")

	// Chat permissions: outsiders cannot read or write.
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", carol.access, nil), http.StatusNotFound, "outsider reads")
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", carol.access, map[string]string{"body": "hi"}), http.StatusNotFound, "outsider writes")
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", bob.access, map[string]string{"body": "   "}), http.StatusBadRequest, "empty message")

	e.expect(e.do("POST", "/conversations/"+convID+"/messages", bob.access, map[string]string{"body": "Salut Alice !"}), http.StatusCreated, "send")
	ev := readEvent("message.created")
	if ev["body"] != "Salut Alice !" || ev["conversationId"] != convID {
		t.Fatalf("unexpected realtime message: %v", ev)
	}

	unread := e.do("GET", "/conversations/unread-count", alice.access, nil)
	if !strings.Contains(string(unread.body), `"unreadCount":1`) {
		t.Fatalf("expected 1 unread: %s", string(unread.body))
	}
	r := e.do("GET", "/conversations", alice.access, nil)
	e.expect(r, http.StatusOK, "conversations")
	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			Peer        struct {
				FirstName string `json:"firstName"`
				Photo     *photo `json:"photo"`
			} `json:"peer"`
			LastMessage *struct{ Body string } `json:"lastMessage"`
		} `json:"conversations"`
	}
	r.decode(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].Peer.FirstName != "Bob" || convs.Conversations[0].UnreadCount != 1 ||
		convs.Conversations[0].LastMessage == nil || convs.Conversations[0].Peer.Photo == nil {
		t.Fatalf("unexpected conversations: %s", string(r.body))
	}
	if strings.Contains(string(r.body), "@example.test") {
		t.Fatal("conversation list leaks emails")
	}

	// Read receipts.
	e.expect(e.do("POST", "/conversations/"+convID+"/read", alice.access, nil), http.StatusOK, "read")
	r = e.do("GET", "/conversations/"+convID+"/messages", bob.access, nil)
	var msgs struct {
		Messages []struct {
			ID        string    `json:"id"`
			Body      string    `json:"body"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"messages"`
		PeerLastReadAt *time.Time `json:"peerLastReadAt"`
		NextCursor     *string    `json:"nextCursor"`
	}
	r.decode(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.PeerLastReadAt == nil || msgs.PeerLastReadAt.Before(msgs.Messages[0].CreatedAt) {
		t.Fatalf("read receipt missing: %s", string(r.body))
	}

	// Pagination of messages (newest first, keyset cursor).
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/conversations/"+convID+"/messages", alice.access, map[string]string{"body": fmt.Sprintf("m%d", i)}), http.StatusCreated, "send")
	}
	r = e.do("GET", "/conversations/"+convID+"/messages?limit=4", bob.access, nil)
	r.decode(t, &msgs)
	if len(msgs.Messages) != 4 || msgs.Messages[0].Body != "m4" || msgs.NextCursor == nil {
		t.Fatalf("unexpected first page: %s", string(r.body))
	}
	r = e.do("GET", "/conversations/"+convID+"/messages?limit=4&before="+*msgs.NextCursor, bob.access, nil)
	r.decode(t, &msgs)
	if len(msgs.Messages) != 2 || msgs.Messages[1].Body != "Salut Alice !" || msgs.NextCursor != nil {
		t.Fatalf("unexpected second page: %s", string(r.body))
	}

	// Local deletion hides history for Alice only.
	e.expect(e.do("DELETE", "/conversations/"+convID, alice.access, nil), http.StatusNoContent, "clear")
	r = e.do("GET", "/conversations", alice.access, nil)
	r.decode(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("cleared conversation still listed")
	}
	r = e.do("GET", "/conversations/"+convID+"/messages", bob.access, nil)
	r.decode(t, &msgs)
	if len(msgs.Messages) != 6 {
		t.Fatal("clearing must not affect the peer")
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", bob.access, map[string]string{"body": "Toujours là ?"}), http.StatusCreated, "send after clear")
	r = e.do("GET", "/conversations/"+convID+"/messages", alice.access, nil)
	r.decode(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.Messages[0].Body != "Toujours là ?" {
		t.Fatalf("expected only new messages after clear: %s", string(r.body))
	}

	// Profile visibility: matched profile visible even when hidden from discovery.
	e.expect(e.do("PUT", "/me/privacy", bob.access, map[string]bool{"discoverable": false, "showDistance": false}), http.StatusOK, "privacy")
	r = e.do("GET", "/profiles/"+bob.id, alice.access, nil)
	e.expect(r, http.StatusOK, "matched profile visible")
	var bobProfile publicProfile
	r.decode(t, &bobProfile)
	if bobProfile.DistanceKm != nil {
		t.Fatal("distance must be hidden when showDistance=false")
	}
	if strings.Contains(string(r.body), "latitude") || strings.Contains(string(r.body), "@") || strings.Contains(string(r.body), "birthdate") {
		t.Fatal("public profile leaks private data")
	}
	e.expect(e.do("GET", "/profiles/"+bob.id, carol.access, nil), http.StatusNotFound, "hidden profile not visible to others")

	// Block: match ends, chat closes, profiles disappear both ways.
	e.expect(e.do("POST", "/blocks", alice.access, map[string]string{"userId": bob.id}), http.StatusNoContent, "block")
	readEvent("match.ended")
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", bob.access, nil), http.StatusNotFound, "chat after block")
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", bob.access, map[string]string{"body": "?"}), http.StatusNotFound, "send after block")
	e.expect(e.do("GET", "/profiles/"+alice.id, bob.access, nil), http.StatusNotFound, "blocked profile")
	e.swipe(bob, alice.id, "like", http.StatusNotFound)
	r = e.do("GET", "/blocks", alice.access, nil)
	if !strings.Contains(string(r.body), bob.id) {
		t.Fatalf("block list missing bob: %s", string(r.body))
	}
	e.expect(e.do("DELETE", "/blocks/"+bob.id, alice.access, nil), http.StatusNoContent, "unblock")

	// Report auto-blocks by default.
	r = e.do("POST", "/reports", alice.access, map[string]string{"userId": dan.id, "reason": "harassment", "details": "messages insistants"})
	e.expect(r, http.StatusCreated, "report")
	if !strings.Contains(string(r.body), `"blocked":true`) {
		t.Fatalf("report should block: %s", string(r.body))
	}
	e.expect(e.do("POST", "/reports", alice.access, map[string]string{"userId": dan.id, "reason": "bad"}), http.StatusBadRequest, "invalid reason")
	e.expect(e.do("POST", "/reports", alice.access, map[string]string{"userId": alice.id, "reason": "spam"}), http.StatusBadRequest, "self report")
}

func TestConcurrentMutualLikeCreatesSingleMatch(t *testing.T) {
	e := setup(t)
	a := e.register("ca")
	b := e.register("cb")
	e.onboard(a, profileSpec{"Ana", "woman", "1993-01-01", []string{"man"}, 45.76, 4.83, nil})
	e.onboard(b, profileSpec{"Ben", "man", "1993-01-01", []string{"woman"}, 45.76, 4.84, nil})

	var wg sync.WaitGroup
	results := make([]swipeBody, 2)
	for i, pair := range [][2]user{{a, b}, {b, a}} {
		wg.Add(1)
		go func(i int, from, to user) {
			defer wg.Done()
			r := e.do("POST", "/swipes", from.access, map[string]string{"targetUserId": to.id, "action": "like"})
			json.Unmarshal(r.body, &results[i])
		}(i, pair[0], pair[1])
	}
	wg.Wait()

	matched := 0
	for _, r := range results {
		if r.Match != nil {
			matched++
		}
	}
	var count int
	e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches`).Scan(&count)
	if count != 1 || matched < 1 {
		t.Fatalf("expected exactly one match, got %d (responses with match: %d)", count, matched)
	}
}

func TestDeleteAccountRemovesData(t *testing.T) {
	e := setup(t)
	a := e.register("del")
	b := e.register("keep")
	e.onboard(a, profileSpec{"Zoé", "woman", "1993-01-01", []string{"man"}, 45.76, 4.83, nil})
	e.onboard(b, profileSpec{"Yann", "man", "1993-01-01", []string{"woman"}, 45.76, 4.84, nil})
	e.swipe(a, b.id, "like", http.StatusOK)
	m := e.swipe(b, a.id, "like", http.StatusOK)
	e.expect(e.do("POST", "/conversations/"+m.Match.ConversationID+"/messages", a.access, map[string]string{"body": "coucou"}), http.StatusCreated, "send")

	e.expect(e.do("DELETE", "/me", a.access, map[string]string{"password": "wrong"}), http.StatusForbidden, "delete with wrong password")
	e.expect(e.do("DELETE", "/me", a.access, map[string]string{"password": "Password123"}), http.StatusNoContent, "delete")
	e.expect(e.do("GET", "/me", a.access, nil), http.StatusUnauthorized, "token after deletion")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": a.email, "password": "Password123"}), http.StatusUnauthorized, "login after deletion")

	var leftovers int
	e.pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM profiles WHERE user_id = $1) + (SELECT COUNT(*) FROM photos WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM messages WHERE sender_id = $1) + (SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1)
		     + (SELECT COUNT(*) FROM matches) + (SELECT COUNT(*) FROM sessions WHERE user_id = $1)
	`, a.id).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("expected all data to be deleted, %d rows left", leftovers)
	}
	files, _ := filepath.Glob(filepath.Join(e.uploads, "*.jpg"))
	if len(files) != 1 {
		t.Fatalf("expected only the remaining user's photo on disk, got %d files", len(files))
	}
	r := e.do("GET", "/conversations", b.access, nil)
	if !strings.Contains(string(r.body), `"conversations":[]`) {
		t.Fatalf("conversation with deleted user still listed: %s", string(r.body))
	}
}
