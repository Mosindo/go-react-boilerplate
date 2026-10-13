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
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-secret-0123456789abcdef-xyz"

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

func (m *captureMailer) codeFor(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	match := regexp.MustCompile(`code is ([A-Z0-9]+)`).FindStringSubmatch(m.last[email])
	if match == nil {
		return ""
	}
	return match[1]
}

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mail   *captureMailer
	hub    *realtime.Hub
	emails []string
	// freshLimiter rebuilds the router (and its rate limiter) before every
	// password-recovery call so brute-force logic can be exercised past the IP limit.
	freshLimiter bool
}

func setup(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	e := &env{t: t, pool: pool, mail: &captureMailer{}, hub: realtime.NewHub()}
	e.router = setupRouter(&app{
		dbPool: pool, jwtSecret: []byte(testSecret), mailer: e.mail, hub: e.hub,
		limiter: middleware.NewRateLimiter(),
	})
	t.Cleanup(func() {
		for _, email := range e.emails {
			_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
		}
		pool.Close()
	})
	return e
}

type resp struct {
	Status int
	Body   map[string]any
	Raw    []byte
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	if e.freshLimiter && strings.HasPrefix(path, "/auth/password") {
		e.router = rebuildWithFreshLimiter(e)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	out := resp{Status: w.Code, Raw: w.Body.Bytes()}
	_ = json.Unmarshal(w.Body.Bytes(), &out.Body)
	return out
}

func (e *env) upload(method, path, token string, contentType string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="x"`)
	h.Set("Content-Type", contentType)
	part, _ := mw.CreatePart(h)
	_, _ = part.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	out := resp{Status: w.Code, Raw: w.Body.Bytes()}
	_ = json.Unmarshal(w.Body.Bytes(), &out.Body)
	return out
}

func testPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type user struct {
	ID, Token, Refresh, Email string
}

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s_%d@test.invalid", prefix, time.Now().UnixNano())
	e.emails = append(e.emails, email)
	r := e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"})
	if r.Status != http.StatusCreated {
		e.t.Fatalf("register %s: %d %s", email, r.Status, r.Raw)
	}
	u := r.Body["user"].(map[string]any)
	return user{ID: u["id"].(string), Token: r.Body["accessToken"].(string), Refresh: r.Body["refreshToken"].(string), Email: email}
}

type spec struct {
	Name, Gender string
	Birth        string
	Interested   []string
	Lat, Lon     float64
	MaxDist      *int
	MinAge       int
	MaxAge       int
}

// onboard completes a profile: profile, preferences, location and one photo.
func (e *env) onboard(u user, s spec) {
	e.t.Helper()
	if s.MinAge == 0 {
		s.MinAge, s.MaxAge = 18, 99
	}
	r := e.do("PUT", "/profile", u.Token, map[string]any{
		"firstName": s.Name, "birthDate": s.Birth, "gender": s.Gender, "bio": "Bonjour", "interestIds": []int{1, 2},
	})
	if r.Status != 200 {
		e.t.Fatalf("profile: %d %s", r.Status, r.Raw)
	}
	r = e.do("PUT", "/profile/preferences", u.Token, map[string]any{
		"interestedIn": s.Interested, "minAge": s.MinAge, "maxAge": s.MaxAge, "maxDistanceKm": s.MaxDist,
	})
	if r.Status != 200 {
		e.t.Fatalf("prefs: %d %s", r.Status, r.Raw)
	}
	r = e.do("PUT", "/profile/location", u.Token, map[string]any{"latitude": s.Lat, "longitude": s.Lon})
	if r.Status != 200 {
		e.t.Fatalf("location: %d %s", r.Status, r.Raw)
	}
	r = e.upload("POST", "/photos", u.Token, "image/png", testPNG(400, 300))
	if r.Status != 201 {
		e.t.Fatalf("photo: %d %s", r.Status, r.Raw)
	}
}

func ids(r resp, key string) []string {
	var out []string
	list, _ := r.Body[key].([]any)
	for _, item := range list {
		out = append(out, item.(map[string]any)["id"].(string))
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func birthYearsAgo(years int) string {
	return time.Now().AddDate(-years, 0, -3).Format("2006-01-02")
}

func TestHealth(t *testing.T) {
	e := setup(t)
	if r := e.do("GET", "/health", "", nil); r.Status != 200 {
		t.Fatalf("health: %d", r.Status)
	}
}

// Registration -> profile -> discovery -> like -> match -> conversation.
func TestCriticalDatingFlow(t *testing.T) {
	e := setup(t)
	alice, bob := e.register("alice"), e.register("bob")
	// Paris and a point ~20 km away.
	e.onboard(alice, spec{Name: "Alice", Gender: "woman", Birth: birthYearsAgo(28), Interested: []string{"man"}, Lat: 48.8566, Lon: 2.3522})
	e.onboard(bob, spec{Name: "Bob", Gender: "man", Birth: birthYearsAgo(31), Interested: []string{"woman"}, Lat: 48.9566, Lon: 2.4522})

	// Profile completeness gates discovery.
	carol := e.register("carol")
	if r := e.do("GET", "/discover", carol.Token, nil); r.Status != http.StatusConflict {
		t.Fatalf("incomplete profile must not discover: %d", r.Status)
	}

	// Discovery returns a coarse distance and never coordinates / birth date.
	r := e.do("GET", "/discover", alice.Token, nil)
	if r.Status != 200 || !contains(ids(r, "profiles"), bob.ID) {
		t.Fatalf("alice should discover bob: %d %s", r.Status, r.Raw)
	}
	raw := string(r.Raw)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("discover leaked %q: %s", forbidden, raw)
		}
	}
	bobCard := r.Body["profiles"].([]any)[0].(map[string]any)
	if d, ok := bobCard["distanceKm"].(float64); !ok || int(d)%5 != 0 || d < 10 {
		t.Fatalf("expected a 5km-bucketed distance, got %v", bobCard["distanceKm"])
	}

	// Alice likes Bob: no match yet. Repeating is rejected.
	r = e.do("POST", "/likes", alice.Token, map[string]string{"userId": bob.ID, "action": "like"})
	if r.Status != 200 || r.Body["matched"] != false {
		t.Fatalf("first like: %d %s", r.Status, r.Raw)
	}
	if r = e.do("POST", "/likes", alice.Token, map[string]string{"userId": bob.ID, "action": "like"}); r.Status != http.StatusConflict {
		t.Fatalf("duplicate like must conflict: %d", r.Status)
	}
	if r = e.do("POST", "/likes", alice.Token, map[string]string{"userId": bob.ID, "action": "pass"}); r.Status != http.StatusConflict {
		t.Fatalf("changing a decision must conflict: %d", r.Status)
	}
	// Bob is not shown to Alice again; Alice is first in Bob's feed.
	if r = e.do("GET", "/discover", alice.Token, nil); contains(ids(r, "profiles"), bob.ID) {
		t.Fatal("already-processed profile reappeared")
	}
	if r = e.do("GET", "/matches", bob.Token, nil); len(ids(r, "matches")) != 0 {
		t.Fatal("no match should exist yet")
	}

	// Bob opens a socket before answering, to verify real-time delivery.
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	ws := dialWS(t, srv.URL, bob.Token)
	defer ws.Close()

	r = e.do("POST", "/likes", bob.Token, map[string]string{"userId": alice.ID, "action": "like"})
	if r.Status != 200 || r.Body["matched"] != true {
		t.Fatalf("reciprocal like must match: %d %s", r.Status, r.Raw)
	}
	conv := r.Body["match"].(map[string]any)["conversationId"].(string)
	if ev := readEvent(t, ws, "match"); ev == nil {
		t.Fatal("bob should receive a match event")
	}

	for _, u := range []user{alice, bob} {
		r = e.do("GET", "/matches", u.Token, nil)
		if len(ids(r, "matches")) != 1 {
			t.Fatalf("expected one match for %s: %s", u.Email, r.Raw)
		}
		r = e.do("GET", "/notifications", u.Token, nil)
		if r.Body["unreadCount"].(float64) != 1 {
			t.Fatalf("expected a match notification: %s", r.Raw)
		}
	}

	// Conversation: send, realtime push, unread, read receipts.
	r = e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]string{"body": "  Salut Bob !  "})
	if r.Status != 201 || r.Body["body"] != "Salut Bob !" {
		t.Fatalf("send: %d %s", r.Status, r.Raw)
	}
	if ev := readEvent(t, ws, "message"); ev == nil {
		t.Fatal("bob should receive the message in real time")
	}
	r = e.do("GET", "/conversations", bob.Token, nil)
	list := r.Body["conversations"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["unreadCount"].(float64) != 1 {
		t.Fatalf("bob should have 1 unread: %s", r.Raw)
	}
	r = e.do("GET", "/conversations/"+conv+"/messages", alice.Token, nil)
	if msgs := r.Body["messages"].([]any); len(msgs) != 1 || msgs[0].(map[string]any)["read"] != false {
		t.Fatalf("message should be unread by bob: %s", r.Raw)
	}
	if r = e.do("POST", "/conversations/"+conv+"/read", bob.Token, nil); r.Status != 200 {
		t.Fatalf("mark read: %d", r.Status)
	}
	r = e.do("GET", "/conversations/"+conv+"/messages", alice.Token, nil)
	if msgs := r.Body["messages"].([]any); msgs[0].(map[string]any)["read"] != true {
		t.Fatalf("message should now be read: %s", r.Raw)
	}
	r = e.do("GET", "/conversations", bob.Token, nil)
	if r.Body["conversations"].([]any)[0].(map[string]any)["unreadCount"].(float64) != 0 {
		t.Fatal("unread should be cleared")
	}

	// Validation.
	if r = e.do("POST", "/conversations/"+conv+"/messages", bob.Token, map[string]string{"body": "   "}); r.Status != 400 {
		t.Fatalf("blank message must be rejected: %d", r.Status)
	}
	if r = e.do("POST", "/conversations/"+conv+"/messages", bob.Token, map[string]string{"body": strings.Repeat("x", 2001)}); r.Status != 400 {
		t.Fatalf("overlong message must be rejected: %d", r.Status)
	}

	// Local deletion hides the conversation until a new message arrives.
	if r = e.do("DELETE", "/conversations/"+conv, bob.Token, nil); r.Status != 204 {
		t.Fatalf("hide: %d", r.Status)
	}
	if r = e.do("GET", "/conversations", bob.Token, nil); len(r.Body["conversations"].([]any)) != 0 {
		t.Fatal("hidden conversation still listed")
	}
	if r = e.do("GET", "/conversations", alice.Token, nil); len(r.Body["conversations"].([]any)) != 1 {
		t.Fatal("alice's copy must be unaffected")
	}
	e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]string{"body": "Tu es là ?"})
	if r = e.do("GET", "/conversations", bob.Token, nil); len(r.Body["conversations"].([]any)) != 1 {
		t.Fatal("a new message must resurface the conversation")
	}

	// Unmatch ends the conversation for both.
	matchID := ids(e.do("GET", "/matches", alice.Token, nil), "matches")[0]
	if r = e.do("DELETE", "/matches/"+matchID, alice.Token, nil); r.Status != 204 {
		t.Fatalf("unmatch: %d", r.Status)
	}
	if r = e.do("POST", "/conversations/"+conv+"/messages", bob.Token, map[string]string{"body": "hello?"}); r.Status != 404 {
		t.Fatalf("messaging after unmatch must fail: %d", r.Status)
	}
}

func dialWS(t *testing.T, base, token string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(base, "http") + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := c.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		t.Fatal(err)
	}
	readEvent(t, c, "ready")
	return c
}

func readEvent(t *testing.T, c *websocket.Conn, want string) map[string]any {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		var ev map[string]any
		if err := c.ReadJSON(&ev); err != nil {
			t.Fatalf("waiting for %q: %v", want, err)
		}
		if ev["type"] == want {
			return ev
		}
	}
}

func TestWebSocketRejectsBadToken(t *testing.T) {
	e := setup(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.WriteJSON(map[string]string{"type": "auth", "token": "not-a-jwt"})
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("connection with an invalid token must be closed")
	}
}

func TestMatchingRespectsPreferences(t *testing.T) {
	e := setup(t)
	seeker := e.register("seeker")
	young := e.register("young")
	oldMan := e.register("old")
	ok := e.register("ok")
	farAway := e.register("far")
	wrongGender := e.register("wg")
	notMutual := e.register("nm")
	near := 50
	e.onboard(seeker, spec{Name: "Sam", Gender: "woman", Birth: birthYearsAgo(30), Interested: []string{"man"}, MinAge: 25, MaxAge: 40, MaxDist: &near, Lat: 48.85, Lon: 2.35})
	e.onboard(young, spec{Name: "Yan", Gender: "man", Birth: birthYearsAgo(22), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
	e.onboard(oldMan, spec{Name: "Olaf", Gender: "man", Birth: birthYearsAgo(55), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
	e.onboard(ok, spec{Name: "Omar", Gender: "man", Birth: birthYearsAgo(33), Interested: []string{"woman"}, Lat: 48.86, Lon: 2.36})
	e.onboard(farAway, spec{Name: "Felix", Gender: "man", Birth: birthYearsAgo(33), Interested: []string{"woman"}, Lat: 43.30, Lon: 5.37}) // Marseille
	e.onboard(wrongGender, spec{Name: "Wendy", Gender: "woman", Birth: birthYearsAgo(33), Interested: []string{"man"}, Lat: 48.85, Lon: 2.35})
	e.onboard(notMutual, spec{Name: "Nico", Gender: "man", Birth: birthYearsAgo(33), Interested: []string{"man"}, Lat: 48.85, Lon: 2.35})

	got := ids(e.do("GET", "/discover?limit=20", seeker.Token, nil), "profiles")
	if !contains(got, ok.ID) {
		t.Fatal("compatible profile missing")
	}
	for name, id := range map[string]string{"too young": young.ID, "too old": oldMan.ID, "too far": farAway.ID, "wrong gender": wrongGender.ID, "not mutual": notMutual.ID} {
		if contains(got, id) {
			t.Errorf("%s profile must not be shown", name)
		}
	}
	// The server also refuses direct likes on ineligible profiles.
	if r := e.do("POST", "/likes", seeker.Token, map[string]string{"userId": young.ID, "action": "like"}); r.Status != 404 {
		t.Fatalf("liking an ineligible profile must fail: %d", r.Status)
	}
	if r := e.do("POST", "/likes", seeker.Token, map[string]string{"userId": seeker.ID, "action": "like"}); r.Status != 400 {
		t.Fatalf("self like must fail: %d", r.Status)
	}
	if r := e.do("POST", "/likes", seeker.Token, map[string]string{"userId": ok.ID, "action": "super"}); r.Status != 400 {
		t.Fatalf("unknown action must fail: %d", r.Status)
	}
	if r := e.do("GET", "/discover?limit=500", seeker.Token, nil); r.Status != 400 {
		t.Fatalf("limit must be validated: %d", r.Status)
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := setup(t)
	for i := 0; i < 5; i++ {
		a, b := e.register("racea"), e.register("raceb")
		e.onboard(a, spec{Name: "Ana", Gender: "woman", Birth: birthYearsAgo(27), Interested: []string{"man"}, Lat: 48.85, Lon: 2.35})
		e.onboard(b, spec{Name: "Ben", Gender: "man", Birth: birthYearsAgo(27), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
		var wg sync.WaitGroup
		results := make([]resp, 2)
		for idx, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func(idx int, from, to user) {
				defer wg.Done()
				results[idx] = e.do("POST", "/likes", from.Token, map[string]string{"userId": to.ID, "action": "like"})
			}(idx, pair[0], pair[1])
		}
		wg.Wait()
		matched := 0
		for _, r := range results {
			if r.Status != 200 {
				t.Fatalf("like failed: %d %s", r.Status, r.Raw)
			}
			if r.Body["matched"] == true {
				matched++
			}
		}
		if matched < 1 {
			t.Fatal("simultaneous reciprocal likes must produce a match")
		}
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("expected exactly one match row, got %d", n)
		}
	}
}

func TestConversationPermissions(t *testing.T) {
	e := setup(t)
	a, b, mallory := e.register("pa"), e.register("pb"), e.register("mallory")
	e.onboard(a, spec{Name: "Ada", Gender: "woman", Birth: birthYearsAgo(27), Interested: []string{"man"}, Lat: 48.85, Lon: 2.35})
	e.onboard(b, spec{Name: "Bao", Gender: "man", Birth: birthYearsAgo(27), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
	e.onboard(mallory, spec{Name: "Mal", Gender: "man", Birth: birthYearsAgo(27), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
	e.do("POST", "/likes", a.Token, map[string]string{"userId": b.ID, "action": "like"})
	conv := e.do("POST", "/likes", b.Token, map[string]string{"userId": a.ID, "action": "like"}).Body["match"].(map[string]any)["conversationId"].(string)
	e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "secret"})

	for _, tc := range []struct{ method, path string }{
		{"GET", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/read"},
		{"DELETE", "/conversations/" + conv},
	} {
		r := e.do(tc.method, tc.path, mallory.Token, map[string]string{"body": "intruder"})
		if r.Status != 404 {
			t.Errorf("%s %s by outsider: want 404, got %d", tc.method, tc.path, r.Status)
		}
		if strings.Contains(string(r.Raw), "secret") {
			t.Errorf("%s leaked message content", tc.path)
		}
	}
	if r := e.do("GET", "/conversations", mallory.Token, nil); len(r.Body["conversations"].([]any)) != 0 {
		t.Fatal("outsider sees conversations")
	}
	if r := e.do("GET", "/conversations/not-a-uuid/messages", a.Token, nil); r.Status != 404 {
		t.Fatalf("malformed id: %d", r.Status)
	}
	// Mallory may not unmatch someone else's match either.
	matchID := ids(e.do("GET", "/matches", a.Token, nil), "matches")[0]
	if r := e.do("DELETE", "/matches/"+matchID, mallory.Token, nil); r.Status != 404 {
		t.Fatalf("outsider unmatch: %d", r.Status)
	}
	// Endpoints require authentication.
	for _, p := range []string{"/discover", "/matches", "/conversations", "/notifications", "/profile", "/photos", "/blocks"} {
		if r := e.do("GET", p, "", nil); r.Status != 401 {
			t.Errorf("%s without token: %d", p, r.Status)
		}
	}
}

func TestBlockAndReport(t *testing.T) {
	e := setup(t)
	a, b := e.register("ba"), e.register("bb")
	e.onboard(a, spec{Name: "Ana", Gender: "woman", Birth: birthYearsAgo(27), Interested: []string{"man"}, Lat: 48.85, Lon: 2.35})
	e.onboard(b, spec{Name: "Ben", Gender: "man", Birth: birthYearsAgo(27), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
	e.do("POST", "/likes", a.Token, map[string]string{"userId": b.ID, "action": "like"})
	m := e.do("POST", "/likes", b.Token, map[string]string{"userId": a.ID, "action": "like"})
	conv := m.Body["match"].(map[string]any)["conversationId"].(string)
	e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "hi"})
	photoID := ids(e.do("GET", "/photos", b.Token, nil), "photos")[0]
	if r := e.do("GET", "/photos/"+photoID+"/file", a.Token, nil); r.Status != 200 || r.Body != nil {
		t.Fatalf("match can view photo: %d", r.Status)
	}

	// Reports validate their input and are idempotent per day.
	if r := e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "boredom"}); r.Status != 400 {
		t.Fatalf("invalid reason: %d", r.Status)
	}
	if r := e.do("POST", "/reports", a.Token, map[string]any{"userId": a.ID, "reason": "spam"}); r.Status != 400 {
		t.Fatalf("self report: %d", r.Status)
	}
	for i := 0; i < 2; i++ {
		if r := e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "rude"}); r.Status != 201 {
			t.Fatalf("report: %d %s", r.Status, r.Raw)
		}
	}
	var reports int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reports WHERE reporter_id = $1`, a.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("duplicate report within 24h must be ignored, got %d rows", reports)
	}

	// Blocking removes every trace in both directions.
	if r := e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}); r.Status != 204 {
		t.Fatalf("block: %d", r.Status)
	}
	for name, tok := range map[string]string{"blocker": a.Token, "blocked": b.Token} {
		if r := e.do("GET", "/matches", tok, nil); len(ids(r, "matches")) != 0 {
			t.Errorf("%s still sees the match", name)
		}
		if r := e.do("GET", "/conversations", tok, nil); len(r.Body["conversations"].([]any)) != 0 {
			t.Errorf("%s still sees the conversation", name)
		}
		if r := e.do("POST", "/conversations/"+conv+"/messages", tok, map[string]string{"body": "x"}); r.Status != 404 {
			t.Errorf("%s can still send: %d", name, r.Status)
		}
	}
	if r := e.do("GET", "/profiles/"+b.ID, a.Token, nil); r.Status != 404 {
		t.Fatalf("blocker views blocked profile: %d", r.Status)
	}
	if r := e.do("GET", "/profiles/"+a.ID, b.Token, nil); r.Status != 404 {
		t.Fatalf("blocked views blocker profile: %d", r.Status)
	}
	if r := e.do("GET", "/photos/"+photoID+"/file", a.Token, nil); r.Status != 404 {
		t.Fatalf("blocker can still fetch photo: %d", r.Status)
	}
	if r := e.do("POST", "/likes", a.Token, map[string]string{"userId": b.ID, "action": "like"}); r.Status == 200 {
		t.Fatal("cannot like a blocked user")
	}
	if r := e.do("GET", "/blocks", a.Token, nil); len(r.Body["blocks"].([]any)) != 1 {
		t.Fatalf("block list: %s", r.Raw)
	}
	if r := e.do("DELETE", "/blocks/"+b.ID, a.Token, nil); r.Status != 204 {
		t.Fatalf("unblock: %d", r.Status)
	}
	// Reporting with alsoBlock blocks in one call.
	c := e.register("bc")
	e.onboard(c, spec{Name: "Cal", Gender: "man", Birth: birthYearsAgo(27), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
	if r := e.do("POST", "/reports", a.Token, map[string]any{"userId": c.ID, "reason": "fake", "alsoBlock": true}); r.Status != 201 {
		t.Fatalf("report+block: %d", r.Status)
	}
	if r := e.do("GET", "/profiles/"+c.ID, a.Token, nil); r.Status != 404 {
		t.Fatalf("report+block must hide the profile: %d", r.Status)
	}
	if r := e.do("POST", "/blocks", a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}); r.Status != 404 {
		t.Fatalf("block unknown user: %d", r.Status)
	}
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e := setup(t)
	a, b := e.register("da"), e.register("db")
	e.onboard(a, spec{Name: "Ana", Gender: "woman", Birth: birthYearsAgo(27), Interested: []string{"man"}, Lat: 48.85, Lon: 2.35})
	e.onboard(b, spec{Name: "Ben", Gender: "man", Birth: birthYearsAgo(27), Interested: []string{"woman"}, Lat: 48.85, Lon: 2.35})
	e.do("POST", "/likes", a.Token, map[string]string{"userId": b.ID, "action": "like"})
	conv := e.do("POST", "/likes", b.Token, map[string]string{"userId": a.ID, "action": "like"}).Body["match"].(map[string]any)["conversationId"].(string)
	e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "bye"})
	e.do("POST", "/reports", b.Token, map[string]any{"userId": a.ID, "reason": "spam"})

	if r := e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong-password1"}); r.Status != 403 {
		t.Fatalf("wrong password: %d", r.Status)
	}
	if r := e.do("DELETE", "/me", a.Token, map[string]string{"password": "Password123"}); r.Status != 204 {
		t.Fatalf("delete: %d %s", r.Status, r.Raw)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Password123"}); r.Status != 401 {
		t.Fatalf("login after deletion: %d", r.Status)
	}
	for table, col := range map[string]string{"profiles": "user_id", "preferences": "user_id", "photos": "user_id", "likes": "from_user_id", "sessions": "user_id", "notifications": "user_id", "user_interests": "user_id", "reports": "reported_id"} {
		var n int
		_ = e.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, table, col), a.ID).Scan(&n)
		if n != 0 {
			t.Errorf("%s still holds %d rows of the deleted user", table, n)
		}
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM messages WHERE conversation_id = $1`, conv).Scan(&n)
	if n != 0 {
		t.Error("messages of the deleted user remain")
	}
	if r := e.do("GET", "/matches", b.Token, nil); len(ids(r, "matches")) != 0 {
		t.Error("the other user still sees the deleted match")
	}
	if r := e.do("GET", "/profiles/"+a.ID, b.Token, nil); r.Status != 404 {
		t.Errorf("deleted profile is still viewable: %d", r.Status)
	}
}

func TestProfileValidationAndAgeGate(t *testing.T) {
	e := setup(t)
	u := e.register("val")
	base := func() map[string]any {
		return map[string]any{"firstName": "Zoé", "birthDate": birthYearsAgo(25), "gender": "woman", "bio": "", "interestIds": []int{1}}
	}
	check := func(name string, mutate func(m map[string]any), want int) {
		m := base()
		mutate(m)
		if r := e.do("PUT", "/profile", u.Token, m); r.Status != want {
			t.Errorf("%s: want %d got %d %s", name, want, r.Status, r.Raw)
		}
	}
	check("minor", func(m map[string]any) { m["birthDate"] = birthYearsAgo(17) }, 400)
	check("bad date", func(m map[string]any) { m["birthDate"] = "yesterday" }, 400)
	check("empty name", func(m map[string]any) { m["firstName"] = " " }, 400)
	check("html name", func(m map[string]any) { m["firstName"] = "<b>x</b>" }, 400)
	check("bad gender", func(m map[string]any) { m["gender"] = "robot" }, 400)
	check("long bio", func(m map[string]any) { m["bio"] = strings.Repeat("a", 501) }, 400)
	check("unknown interest", func(m map[string]any) { m["interestIds"] = []int{9999} }, 400)
	check("too many interests", func(m map[string]any) { m["interestIds"] = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} }, 400)
	check("valid", func(m map[string]any) {}, 200)
	// The birth date is set once: it cannot be edited to dodge the age gate.
	check("birth date change", func(m map[string]any) { m["birthDate"] = birthYearsAgo(40) }, 400)

	if r := e.do("PUT", "/profile/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30}); r.Status != 400 {
		t.Fatalf("preferences under 18: %d", r.Status)
	}
	if r := e.do("PUT", "/profile/location", u.Token, map[string]any{"latitude": 123.0, "longitude": 2.0}); r.Status != 400 {
		t.Fatalf("bad coordinates: %d", r.Status)
	}
	r := e.do("PUT", "/profile/location", u.Token, map[string]any{"latitude": 48.856614, "longitude": 2.352222})
	if r.Body["latitude"].(float64) != 48.86 || r.Body["longitude"].(float64) != 2.35 {
		t.Fatalf("coordinates must be stored rounded: %s", r.Raw)
	}
	if r := e.do("GET", "/interests", u.Token, nil); len(r.Body["interests"].([]any)) < 10 {
		t.Fatal("interests missing")
	}
}

func TestPhotoRulesAndProtection(t *testing.T) {
	e := setup(t)
	u, other := e.register("ph"), e.register("pho")
	e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Pia", "birthDate": birthYearsAgo(25), "gender": "woman"})

	if r := e.upload("POST", "/photos", u.Token, "text/plain", []byte("<?php system($_GET[0]); ?>")); r.Status != 422 {
		t.Fatalf("non-image: %d", r.Status)
	}
	// A real PNG header with a lying content type is still validated by content.
	if r := e.upload("POST", "/photos", u.Token, "image/jpeg", []byte("GIF89a....")); r.Status != 422 {
		t.Fatalf("gif: %d", r.Status)
	}
	if r := e.upload("POST", "/photos", u.Token, "image/png", testPNG(50, 50)); r.Status != 422 {
		t.Fatalf("tiny image: %d", r.Status)
	}
	if r := e.upload("POST", "/photos", u.Token, "image/png", bytes.Repeat([]byte{0}, 7<<20)); r.Status != 413 && r.Status != 422 {
		t.Fatalf("oversize: %d", r.Status)
	}
	var first, second string
	for i := 0; i < 6; i++ {
		r := e.upload("POST", "/photos", u.Token, "image/png", testPNG(2400, 1800))
		if r.Status != 201 {
			t.Fatalf("upload %d: %d %s", i, r.Status, r.Raw)
		}
		if i == 0 {
			first = r.Body["id"].(string)
			if r.Body["width"].(float64) != 1280 {
				t.Fatalf("expected downscale to 1280, got %v", r.Body["width"])
			}
		}
		if i == 1 {
			second = r.Body["id"].(string)
		}
	}
	if r := e.upload("POST", "/photos", u.Token, "image/png", testPNG(400, 400)); r.Status != 409 {
		t.Fatalf("seventh photo: %d", r.Status)
	}

	// Served as a re-encoded JPEG, private, and only to authenticated viewers.
	req := httptest.NewRequest("GET", "/photos/"+first+"/file?size=thumb", nil)
	req.Header.Set("Authorization", "Bearer "+u.Token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/jpeg" || !strings.Contains(w.Header().Get("Cache-Control"), "private") {
		t.Fatalf("serve photo: %d %v", w.Code, w.Header())
	}
	if !bytes.HasPrefix(w.Body.Bytes(), []byte{0xFF, 0xD8}) {
		t.Fatal("served photo is not a JPEG")
	}
	if r := e.do("GET", "/photos/"+first+"/file", "", nil); r.Status != 401 {
		t.Fatalf("anonymous photo access: %d", r.Status)
	}
	// Not viewable yet: the owner's profile has no preferences but has photos & is visible -> viewable.
	// Hidden profiles are not viewable by non-matches.
	e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Pia", "birthDate": birthYearsAgo(25), "gender": "woman", "isVisible": false})
	if r := e.do("GET", "/photos/"+first+"/file", other.Token, nil); r.Status != 404 {
		t.Fatalf("paused profile photo must be private: %d", r.Status)
	}
	// Others cannot delete/replace/reorder them.
	if r := e.do("DELETE", "/photos/"+first, other.Token, nil); r.Status != 404 {
		t.Fatalf("foreign delete: %d", r.Status)
	}
	if r := e.upload("PUT", "/photos/"+first, other.Token, "image/png", testPNG(300, 300)); r.Status != 404 {
		t.Fatalf("foreign replace: %d", r.Status)
	}

	// Ordering: the first id becomes the main photo; the set must match exactly.
	list := ids(e.do("GET", "/photos", u.Token, nil), "photos")
	reordered := append([]string{list[1], list[0]}, list[2:]...)
	if r := e.do("PUT", "/photos/order", u.Token, map[string]any{"ids": reordered}); r.Status != 200 || ids(r, "photos")[0] != second {
		t.Fatalf("reorder: %d %s", r.Status, r.Raw)
	}
	if r := e.do("PUT", "/photos/order", u.Token, map[string]any{"ids": list[:3]}); r.Status != 422 {
		t.Fatalf("partial reorder: %d", r.Status)
	}
	// Replace keeps the slot but changes the id; delete compacts positions.
	r := e.upload("PUT", "/photos/"+second, u.Token, "image/png", testPNG(500, 500))
	if r.Status != 200 || r.Body["id"] == second || r.Body["position"].(float64) != 0 {
		t.Fatalf("replace: %d %s", r.Status, r.Raw)
	}
	if r := e.do("DELETE", "/photos/"+r.Body["id"].(string), u.Token, nil); r.Status != 204 {
		t.Fatalf("delete: %d", r.Status)
	}
	after := e.do("GET", "/photos", u.Token, nil)
	for i, p := range after.Body["photos"].([]any) {
		if int(p.(map[string]any)["position"].(float64)) != i {
			t.Fatalf("positions must stay contiguous: %s", after.Raw)
		}
	}
}

func TestAuthLifecycle(t *testing.T) {
	e := setup(t)
	u := e.register("auth")
	// Email is normalized and case-insensitive; passwords must be reasonably strong.
	if r := e.do("POST", "/auth/register", "", map[string]string{"email": strings.ToUpper(u.Email), "password": "Password123"}); r.Status != 409 {
		t.Fatalf("duplicate email (case-insensitive): %d", r.Status)
	}
	for _, pw := range []string{"short1", "onlyletters", "12345678"} {
		if r := e.do("POST", "/auth/register", "", map[string]string{"email": "weak" + pw + "@test.invalid", "password": pw}); r.Status != 400 {
			t.Errorf("weak password %q accepted: %d", pw, r.Status)
		}
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Wrong12345"}); r.Status != 401 {
		t.Fatalf("bad login: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@test.invalid", "password": "Wrong12345"}); r.Status != 401 || r.Body["error"] != "invalid credentials" {
		t.Fatalf("unknown user must look identical: %d", r.Status)
	}
	if r := e.do("GET", "/me", u.Token, nil); r.Status != 200 || r.Body["email"] != u.Email {
		t.Fatalf("me: %d", r.Status)
	}

	// Refresh rotates; replaying an old refresh token fails.
	r := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh})
	if r.Status != 200 {
		t.Fatalf("refresh: %d", r.Status)
	}
	if r2 := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}); r2.Status != 401 {
		t.Fatalf("replayed refresh token: %d", r2.Status)
	}
	next := r.Body["refreshToken"].(string)
	if r := e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": next}); r.Status != 204 {
		t.Fatalf("logout: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": next}); r.Status != 401 {
		t.Fatalf("refresh after logout: %d", r.Status)
	}

	// Token validation: wrong algorithm, expired, missing expiry, wrong issuer, tampering.
	mk := func(method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	good := jwt.MapClaims{"uid": u.ID, "sid": "s", "iss": middleware.TokenIssuer, "exp": time.Now().Add(time.Hour).Unix()}
	cases := map[string]string{
		"none alg":     mk(jwt.SigningMethodNone, good, jwt.UnsafeAllowNoneSignatureType),
		"hs512":        mk(jwt.SigningMethodHS512, good, []byte(testSecret)),
		"expired":      mk(jwt.SigningMethodHS256, jwt.MapClaims{"uid": u.ID, "sid": "s", "iss": middleware.TokenIssuer, "exp": time.Now().Add(-time.Minute).Unix()}, []byte(testSecret)),
		"no exp":       mk(jwt.SigningMethodHS256, jwt.MapClaims{"uid": u.ID, "sid": "s", "iss": middleware.TokenIssuer}, []byte(testSecret)),
		"wrong issuer": mk(jwt.SigningMethodHS256, jwt.MapClaims{"uid": u.ID, "sid": "s", "iss": "evil", "exp": time.Now().Add(time.Hour).Unix()}, []byte(testSecret)),
		"wrong secret": mk(jwt.SigningMethodHS256, good, []byte("another-secret-another-secret-another")),
		"no uid":       mk(jwt.SigningMethodHS256, jwt.MapClaims{"sid": "s", "iss": middleware.TokenIssuer, "exp": time.Now().Add(time.Hour).Unix()}, []byte(testSecret)),
	}
	for name, tok := range cases {
		if r := e.do("GET", "/me", tok, nil); r.Status != 401 {
			t.Errorf("%s token accepted: %d", name, r.Status)
		}
	}
	if r := e.do("GET", "/me", mk(jwt.SigningMethodHS256, good, []byte(testSecret)), nil); r.Status != 200 {
		t.Errorf("control token rejected: %d", r.Status)
	}
}

func TestPasswordRecovery(t *testing.T) {
	e := setup(t)
	u := e.register("rec")
	e.freshLimiter = true
	if r := e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "ghost@test.invalid"}); r.Status != 202 {
		t.Fatalf("unknown email must look identical: %d", r.Status)
	}
	if e.mail.codeFor("ghost@test.invalid") != "" {
		t.Fatal("no mail for unknown accounts")
	}
	if r := e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}); r.Status != 202 {
		t.Fatalf("forgot: %d", r.Status)
	}
	code := e.mail.codeFor(u.Email)
	if code == "" {
		t.Fatal("recovery code was not sent")
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": "WRONGCODE", "newPassword": "NewPassword9"}); r.Status != 400 {
		t.Fatalf("wrong code: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "short"}); r.Status != 400 {
		t.Fatalf("weak new password: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": strings.ToLower(code), "newPassword": "NewPassword9"}); r.Status != 204 {
		t.Fatalf("reset: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another12345"}); r.Status != 400 {
		t.Fatalf("code must be single use: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}); r.Status != 401 {
		t.Fatalf("old sessions must be revoked: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123"}); r.Status != 401 {
		t.Fatalf("old password still works: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword9"}); r.Status != 200 {
		t.Fatalf("new password: %d", r.Status)
	}

	// Brute force: after 5 wrong guesses even the right code is refused.
	e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email})
	code = e.mail.codeFor(u.Email)
	for i := 0; i < 5; i++ {
		e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": "AAAAAAAA", "newPassword": "NewPassword9"})
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another12345"}); r.Status != 400 {
		t.Fatalf("code must lock after repeated failures: %d", r.Status)
	}
}

func rebuildWithFreshLimiter(e *env) *gin.Engine {
	return setupRouter(&app{dbPool: e.pool, jwtSecret: []byte(testSecret), mailer: e.mail, hub: e.hub, limiter: middleware.NewRateLimiter()})
}

func TestRateLimitingOnAuth(t *testing.T) {
	e := setup(t)
	limited := false
	for i := 0; i < 40; i++ {
		r := e.do("POST", "/auth/login", "", map[string]string{"email": "x@test.invalid", "password": "Nope123456"})
		if r.Status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("login must be rate limited per IP")
	}
}
