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
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-0123456789abcdef0123456789"

var emailSeq atomic.Int64

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

func (m *captureMailer) lastCode(t *testing.T, to string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.sent) - 1; i >= 0; i-- {
		if m.sent[i].To == to {
			code := regexp.MustCompile(`\b\d{6}\b`).FindString(m.sent[i].Body)
			if code != "" {
				return code
			}
		}
	}
	t.Fatalf("no reset code sent to %s", to)
	return ""
}

type env struct {
	t         *testing.T
	srv       *httptest.Server
	pool      *pgxpool.Pool
	mail      *captureMailer
	uploadDir string
}

func setup(t *testing.T) *env {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("integration tests need DATABASE_URL_TEST or DATABASE_URL")
	}
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrations: %v", err)
	}
	uploadDir := t.TempDir()
	store, err := storage.NewLocal(uploadDir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &captureMailer{}
	router := newRouter(deps{
		pool:      pool,
		jwtSecret: []byte(testSecret),
		store:     store,
		mailer:    mail,
		hub:       realtime.NewHub(),
	}, generousLimits())
	srv := httptest.NewServer(router)
	e := &env{t: t, srv: srv, pool: pool, mail: mail, uploadDir: uploadDir}
	t.Cleanup(func() {
		srv.Close()
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email LIKE '%@lueur.test'`)
		pool.Close()
	})
	return e
}

// generousLimits keeps abuse protection wired but out of the way of tests.
func generousLimits() limits {
	n := func() *middleware.RateLimiter { return middleware.NewRateLimiter(10_000, time.Minute) }
	return limits{credentials: n(), refresh: n(), uploads: n(), swipes: n(), messages: n(), reports: n()}
}

type resp struct {
	status int
	body   []byte
}

func (r resp) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.body, dst); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
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
	b, _ := io.ReadAll(res.Body)
	return resp{status: res.StatusCode, body: b}
}

func (e *env) expect(r resp, status int, context string) {
	e.t.Helper()
	if r.status != status {
		e.t.Fatalf("%s: expected %d, got %d: %s", context, status, r.status, r.body)
	}
}

func (e *env) upload(method, path, token string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", "photo.png")
	_, _ = part.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(method, e.srv.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{status: res.StatusCode, body: b}
}

type session struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type member struct {
	session
	email    string
	password string
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s.%d.%d@lueur.test", prefix, time.Now().UnixNano(), emailSeq.Add(1))
}

func (e *env) register(prefix string) member {
	e.t.Helper()
	m := member{email: uniqueEmail(prefix), password: "Motdepasse42"}
	r := e.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(m.email) + " ", "password": m.password})
	e.expect(r, http.StatusCreated, "register")
	r.decode(e.t, &m.session)
	if m.User.Email != m.email {
		e.t.Fatalf("email not normalised: %q", m.User.Email)
	}
	return m
}

func testPNG(seed uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 320, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{seed, uint8(x), uint8(y), 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// onboard creates a complete profile (profile + preferences + location + photo).
func (e *env) onboard(m member, name, gender string, interestedIn []string, lat, lng float64, interests []string) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", m.AccessToken, map[string]any{
		"firstName": name, "birthdate": "1994-05-12", "gender": gender, "bio": "Bonjour, je suis " + name,
		"interests": interests, "lookingFor": "relationship",
	}), http.StatusOK, "profile "+name)
	e.expect(e.do("PUT", "/me/preferences", m.AccessToken, map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": 100,
	}), http.StatusOK, "preferences "+name)
	e.expect(e.do("PUT", "/me/location", m.AccessToken, map[string]any{"latitude": lat, "longitude": lng, "city": "Paris"}), http.StatusNoContent, "location "+name)
	e.expect(e.upload("POST", "/me/photos", m.AccessToken, testPNG(uint8(len(name)*20))), http.StatusCreated, "photo "+name)
}

type publicProfile struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	Interests []struct {
		Slug string `json:"slug"`
	} `json:"interests"`
}

func (e *env) discover(m member) []publicProfile {
	e.t.Helper()
	r := e.do("GET", "/discovery?limit=30", m.AccessToken, nil)
	e.expect(r, http.StatusOK, "discovery")
	var out struct {
		Profiles []publicProfile `json:"profiles"`
	}
	r.decode(e.t, &out)
	return out.Profiles
}

func contains(list []publicProfile, userID string) bool {
	for _, p := range list {
		if p.UserID == userID {
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

func (e *env) swipe(m member, target, action string) (swipeResult, resp) {
	e.t.Helper()
	r := e.do("POST", "/swipes", m.AccessToken, map[string]string{"targetUserId": target, "action": action})
	var out swipeResult
	if r.status < 300 {
		r.decode(e.t, &out)
	}
	return out, r
}

// wsClient connects to /ws and authenticates.
type wsClient struct {
	conn   *websocket.Conn
	events chan map[string]any
}

func (e *env) connect(token string) *wsClient {
	e.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		e.t.Fatalf("ws dial: %v", err)
	}
	auth, _ := json.Marshal(map[string]string{"type": "auth", "token": token})
	if err := conn.Write(ctx, websocket.MessageText, auth); err != nil {
		e.t.Fatalf("ws auth: %v", err)
	}
	c := &wsClient{conn: conn, events: make(chan map[string]any, 32)}
	go func() {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				close(c.events)
				return
			}
			var ev map[string]any
			if json.Unmarshal(data, &ev) == nil {
				c.events <- ev
			}
		}
	}()
	c.waitFor(e.t, "ready")
	e.t.Cleanup(func() { conn.Close(websocket.StatusNormalClosure, "") })
	return c
}

func (c *wsClient) waitFor(t *testing.T, eventType string) map[string]any {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				t.Fatalf("socket closed while waiting for %s", eventType)
			}
			if ev["type"] == eventType {
				return ev
			}
		case <-timeout:
			t.Fatalf("timeout waiting for %s", eventType)
		}
	}
}

func TestHealth(t *testing.T) {
	e := setup(t)
	e.expect(e.do("GET", "/health", "", nil), http.StatusOK, "health")
}

func TestAuthLifecycle(t *testing.T) {
	e := setup(t)

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Motdepasse42"}), http.StatusBadRequest, "invalid email")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": uniqueEmail("weak"), "password": "short"}), http.StatusBadRequest, "weak password")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": uniqueEmail("weak"), "password": "onlyletters"}), http.StatusBadRequest, "password without digit")

	m := e.register("auth")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": m.email, "password": m.password}), http.StatusConflict, "duplicate email")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": m.email, "password": "Wrong12345"}), http.StatusUnauthorized, "wrong password")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": uniqueEmail("ghost"), "password": "Wrong12345"}), http.StatusUnauthorized, "unknown email")

	r := e.do("POST", "/auth/login", "", map[string]string{"email": strings.ToUpper(m.email), "password": m.password})
	e.expect(r, http.StatusOK, "login")
	var login session
	r.decode(t, &login)

	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized, "me without token")
	e.expect(e.do("GET", "/me", "garbage", nil), http.StatusUnauthorized, "me with garbage token")
	e.expect(e.do("GET", "/me", login.AccessToken, nil), http.StatusOK, "me")

	r = e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken})
	e.expect(r, http.StatusOK, "refresh")
	var refreshed session
	r.decode(t, &refreshed)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), http.StatusUnauthorized, "refresh token reuse")

	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": refreshed.RefreshToken}), http.StatusNoContent, "logout")
	e.expect(e.do("GET", "/me", refreshed.AccessToken, nil), http.StatusUnauthorized, "access token revoked by logout")
	e.expect(e.do("GET", "/me", m.AccessToken, nil), http.StatusOK, "other sessions stay valid")
}

func TestPasswordReset(t *testing.T) {
	e := setup(t)
	m := e.register("reset")

	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": uniqueEmail("nobody")}), http.StatusAccepted, "unknown email does not leak")
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": m.email}), http.StatusAccepted, "forgot")
	code := e.mail.lastCode(t, m.email)

	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": m.email, "code": wrong, "password": "Nouveau12345"}), http.StatusBadRequest, "wrong code")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": m.email, "code": code, "password": "Nouveau12345"}), http.StatusNoContent, "reset")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": m.email, "code": code, "password": "Encore12345"}), http.StatusBadRequest, "code is single use")
	e.expect(e.do("GET", "/me", m.AccessToken, nil), http.StatusUnauthorized, "sessions revoked after reset")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": m.email, "password": m.password}), http.StatusUnauthorized, "old password rejected")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": m.email, "password": "Nouveau12345"}), http.StatusOK, "new password accepted")

	// Brute force protection: after 5 wrong attempts even the right code fails.
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": m.email}), http.StatusAccepted, "forgot again")
	code = e.mail.lastCode(t, m.email)
	for i := 0; i < 5; i++ {
		bad := fmt.Sprintf("%06d", i)
		if bad == code {
			bad = "999999"
		}
		e.do("POST", "/auth/password/reset", "", map[string]string{"email": m.email, "code": bad, "password": "Nouveau12345"})
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": m.email, "code": code, "password": "Nouveau12345"}), http.StatusBadRequest, "locked after attempts")
}

func TestProfileAndPhotos(t *testing.T) {
	e := setup(t)
	m := e.register("profile")

	r := e.do("GET", "/me/profile", m.AccessToken, nil)
	e.expect(r, http.StatusOK, "empty profile")
	var mine struct {
		Profile  *map[string]any `json:"profile"`
		Complete bool            `json:"complete"`
		Photos   []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"photos"`
	}
	r.decode(t, &mine)
	if mine.Profile != nil || mine.Complete {
		t.Fatal("expected empty, incomplete profile")
	}
	e.expect(e.do("GET", "/discovery", m.AccessToken, nil), http.StatusConflict, "discovery requires profile")
	e.expect(e.do("PUT", "/me/preferences", m.AccessToken, map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 10}), http.StatusConflict, "preferences need a profile")

	minorBirth := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", m.AccessToken, map[string]any{"firstName": "Léa", "birthdate": minorBirth, "gender": "woman"}), http.StatusBadRequest, "minor rejected")
	e.expect(e.do("PUT", "/me/profile", m.AccessToken, map[string]any{"firstName": "Léa", "birthdate": "1995-01-01", "gender": "woman", "interests": []string{"not-a-real-interest"}}), http.StatusBadRequest, "unknown interest")
	e.expect(e.do("PUT", "/me/profile", m.AccessToken, map[string]any{"firstName": "Léa", "birthdate": "1995-01-01", "gender": "woman", "interests": []string{"hiking", "cinema"}}), http.StatusOK, "valid profile")
	e.expect(e.do("PUT", "/me/location", m.AccessToken, map[string]any{"latitude": 123.0, "longitude": 2.0}), http.StatusBadRequest, "invalid latitude")
	e.expect(e.do("PUT", "/me/location", m.AccessToken, map[string]any{"latitude": 48.856613, "longitude": 2.352222, "city": "Paris"}), http.StatusNoContent, "location")

	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, m.User.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("location must be coarsened before storage, got %v,%v", lat, lng)
	}

	e.expect(e.upload("POST", "/me/photos", m.AccessToken, []byte("<html>not an image</html>")), http.StatusUnsupportedMediaType, "html upload rejected")
	for i := 0; i < 6; i++ {
		e.expect(e.upload("POST", "/me/photos", m.AccessToken, testPNG(uint8(i*30))), http.StatusCreated, "photo upload")
	}
	e.expect(e.upload("POST", "/me/photos", m.AccessToken, testPNG(200)), http.StatusConflict, "7th photo rejected")

	r = e.do("GET", "/me/profile", m.AccessToken, nil)
	r.decode(t, &mine)
	if !mine.Complete || len(mine.Photos) != 6 {
		t.Fatalf("expected complete profile with 6 photos, got %+v", mine)
	}

	// Signed media URL: valid → JPEG, tampered → 404.
	res, err := http.Get(e.srv.URL + mine.Photos[0].URL)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/jpeg" || !bytes.HasPrefix(body, []byte{0xFF, 0xD8}) {
		t.Fatalf("expected JPEG photo, got %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	tampered := strings.Replace(mine.Photos[0].URL, mine.Photos[0].ID, mine.Photos[1].ID, 1)
	res, _ = http.Get(e.srv.URL + tampered)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("tampered URL must be rejected, got %d", res.StatusCode)
	}

	// Reorder: reverse; then main photo is the former last one.
	ids := make([]string, 0, 6)
	for i := len(mine.Photos) - 1; i >= 0; i-- {
		ids = append(ids, mine.Photos[i].ID)
	}
	e.expect(e.do("PUT", "/me/photos/order", m.AccessToken, map[string]any{"photoIds": ids[:5]}), http.StatusBadRequest, "partial order rejected")
	r = e.do("PUT", "/me/photos/order", m.AccessToken, map[string]any{"photoIds": ids})
	e.expect(r, http.StatusOK, "reorder")
	var ordered struct {
		Photos []struct {
			ID string `json:"id"`
		} `json:"photos"`
	}
	r.decode(t, &ordered)
	if ordered.Photos[0].ID != ids[0] {
		t.Fatal("reorder not applied")
	}

	e.expect(e.upload("PUT", "/me/photos/"+ids[0], m.AccessToken, testPNG(77)), http.StatusOK, "replace photo")
	for i := 0; i < 5; i++ {
		e.expect(e.do("DELETE", "/me/photos/"+ids[i], m.AccessToken, nil), http.StatusOK, "delete photo")
	}
	e.expect(e.do("DELETE", "/me/photos/"+ids[5], m.AccessToken, nil), http.StatusConflict, "last photo kept")

	other := e.register("intruder")
	e.expect(e.do("DELETE", "/me/photos/"+ids[5], other.AccessToken, nil), http.StatusNotFound, "cannot delete someone else's photo")
}

func TestMatchingChatAndRealtime(t *testing.T) {
	e := setup(t)
	alice := e.register("alice")
	bruno := e.register("bruno")
	chris := e.register("chris")
	dana := e.register("dana")
	e.onboard(alice, "Alice", "woman", []string{"man"}, 48.85, 2.35, []string{"hiking", "music"})
	e.onboard(bruno, "Bruno", "man", []string{"woman"}, 48.87, 2.33, []string{"hiking"})
	e.onboard(chris, "Chris", "man", []string{"man"}, 48.86, 2.34, nil) // not mutual with Alice
	e.onboard(dana, "Dana", "woman", []string{"man"}, 43.30, 5.37, nil) // Marseille: too far for Bruno

	deck := e.discover(alice)
	if !contains(deck, bruno.User.ID) || contains(deck, chris.User.ID) || contains(deck, alice.User.ID) {
		t.Fatalf("unexpected deck for Alice: %+v", deck)
	}
	if contains(e.discover(bruno), dana.User.ID) {
		t.Fatal("distance filter must exclude Dana from Bruno's deck")
	}
	for _, p := range deck {
		if p.UserID == bruno.User.ID {
			if p.DistanceKm == nil || *p.DistanceKm < 1 || *p.DistanceKm > 5 {
				t.Fatalf("expected approximate distance, got %v", p.DistanceKm)
			}
			if strings.Contains(string(mustJSON(p)), "latitude") {
				t.Fatal("coordinates must never be exposed")
			}
		}
	}

	aliceWS := e.connect(alice.AccessToken)
	brunoWS := e.connect(bruno.AccessToken)

	result, r := e.swipe(alice, bruno.User.ID, "like")
	e.expect(r, http.StatusOK, "alice likes bruno")
	if result.Matched {
		t.Fatal("no match before reciprocity")
	}
	_, r = e.swipe(alice, bruno.User.ID, "like")
	e.expect(r, http.StatusConflict, "duplicate like")
	_, r = e.swipe(alice, alice.User.ID, "like")
	e.expect(r, http.StatusBadRequest, "self like")
	_, r = e.swipe(alice, bruno.User.ID, "superlike")
	e.expect(r, http.StatusBadRequest, "unknown action")
	if contains(e.discover(alice), bruno.User.ID) {
		t.Fatal("processed profile must not reappear")
	}

	bdeck := e.discover(bruno)
	if len(bdeck) == 0 || bdeck[0].UserID != alice.User.ID {
		t.Fatalf("someone who liked you should rank first, got %+v", bdeck)
	}
	result, r = e.swipe(bruno, alice.User.ID, "like")
	e.expect(r, http.StatusCreated, "bruno likes alice back")
	if !result.Matched || result.Match == nil || result.Match.User.UserID != alice.User.ID {
		t.Fatalf("expected match with Alice, got %+v", result)
	}
	matchID := result.Match.ID
	ev := aliceWS.waitFor(t, "match.created")
	if ev["data"].(map[string]any)["id"] != matchID {
		t.Fatal("alice should receive the match in real time")
	}

	var matches int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE $1 IN (user_a_id, user_b_id) AND $2 IN (user_a_id, user_b_id)`, alice.User.ID, bruno.User.ID).Scan(&matches)
	if matches != 1 {
		t.Fatalf("expected exactly one match, got %d", matches)
	}

	// Pass keeps profiles out of the deck too.
	_, r = e.swipe(chris, bruno.User.ID, "pass")
	e.expect(r, http.StatusOK, "pass")

	// Notifications.
	var notifs struct {
		Notifications []struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			IsRead bool   `json:"isRead"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.do("GET", "/notifications", alice.AccessToken, nil).decode(t, &notifs)
	if notifs.UnreadCount != 1 || notifs.Notifications[0].Type != "match" {
		t.Fatalf("expected one match notification, got %+v", notifs)
	}

	// Chat permissions.
	path := "/conversations/" + matchID + "/messages"
	e.expect(e.do("GET", path, chris.AccessToken, nil), http.StatusNotFound, "outsider cannot read")
	e.expect(e.do("POST", path, chris.AccessToken, map[string]string{"body": "intrusion"}), http.StatusNotFound, "outsider cannot write")
	e.expect(e.do("POST", path, alice.AccessToken, map[string]string{"body": "   "}), http.StatusBadRequest, "empty message")
	e.expect(e.do("POST", path, alice.AccessToken, map[string]string{"body": strings.Repeat("a", 2001)}), http.StatusBadRequest, "too long message")

	for i := 1; i <= 35; i++ {
		e.expect(e.do("POST", path, alice.AccessToken, map[string]string{"body": fmt.Sprintf("message %d", i)}), http.StatusCreated, "send")
	}
	created := brunoWS.waitFor(t, "message.created")
	if created["data"].(map[string]any)["body"] != "message 1" {
		t.Fatalf("bruno should receive messages in real time, got %v", created)
	}
	brunoWS.waitFor(t, "notification.created")

	var convs struct {
		Conversations []struct {
			MatchID     string `json:"matchId"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.do("GET", "/conversations", bruno.AccessToken, nil).decode(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 35 || convs.Conversations[0].LastMessage.Body != "message 35" {
		t.Fatalf("unexpected conversations %+v", convs)
	}

	e.do("GET", "/notifications", bruno.AccessToken, nil).decode(t, &notifs)
	messageNotifs := 0
	for _, n := range notifs.Notifications {
		if n.Type == "message" {
			messageNotifs++
		}
	}
	if messageNotifs != 1 {
		t.Fatalf("message notifications must be collapsed per conversation, got %d", messageNotifs)
	}

	// Pagination.
	var page struct {
		Messages []struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"messages"`
		HasMore bool `json:"hasMore"`
	}
	e.do("GET", path+"?limit=30", bruno.AccessToken, nil).decode(t, &page)
	if len(page.Messages) != 30 || !page.HasMore || page.Messages[0].Body != "message 35" {
		t.Fatalf("unexpected first page: %d %v", len(page.Messages), page.HasMore)
	}
	last := page.Messages[len(page.Messages)-1].ID
	e.do("GET", path+"?limit=30&before="+last, bruno.AccessToken, nil).decode(t, &page)
	if len(page.Messages) != 5 || page.HasMore || page.Messages[4].Body != "message 1" {
		t.Fatalf("unexpected second page: %+v", page)
	}

	// Read receipts.
	e.expect(e.do("POST", "/conversations/"+matchID+"/read", bruno.AccessToken, nil), http.StatusOK, "mark read")
	aliceWS.waitFor(t, "conversation.read")
	e.do("GET", "/conversations", bruno.AccessToken, nil).decode(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread count must reset after reading")
	}
	e.do("GET", "/notifications", bruno.AccessToken, nil).decode(t, &notifs)
	for _, n := range notifs.Notifications {
		if n.Type == "message" && !n.IsRead {
			t.Fatal("reading the conversation must mark its notification read")
		}
	}

	// Local clear: Bruno no longer sees history, Alice still does.
	e.expect(e.do("DELETE", path, bruno.AccessToken, nil), http.StatusNoContent, "clear for me")
	e.do("GET", path, bruno.AccessToken, nil).decode(t, &page)
	if len(page.Messages) != 0 {
		t.Fatal("cleared history must be hidden for Bruno")
	}
	e.do("GET", path, alice.AccessToken, nil).decode(t, &page)
	if len(page.Messages) == 0 {
		t.Fatal("clearing is local: Alice keeps her history")
	}

	// Public profile visibility.
	e.expect(e.do("GET", "/profiles/"+alice.User.ID, bruno.AccessToken, nil), http.StatusOK, "view match profile")

	// Unmatch removes the conversation for both.
	e.expect(e.do("DELETE", "/matches/"+matchID, chris.AccessToken, nil), http.StatusNotFound, "outsider cannot unmatch")
	e.expect(e.do("DELETE", "/matches/"+matchID, alice.AccessToken, nil), http.StatusNoContent, "unmatch")
	brunoWS.waitFor(t, "match.removed")
	e.expect(e.do("GET", path, bruno.AccessToken, nil), http.StatusNotFound, "conversation gone after unmatch")
	if contains(e.discover(alice), bruno.User.ID) || contains(e.discover(bruno), alice.User.ID) {
		t.Fatal("unmatched profiles must not reappear")
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := setup(t)
	for round := 0; round < 5; round++ {
		a := e.register("race_a")
		b := e.register("race_b")
		e.onboard(a, "Ana", "woman", []string{"man"}, 45.76, 4.83, nil)
		e.onboard(b, "Ben", "man", []string{"woman"}, 45.76, 4.84, nil)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); e.swipe(a, b.User.ID, "like") }()
		go func() { defer wg.Done(); e.swipe(b, a.User.ID, "like") }()
		wg.Wait()
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE $1 IN (user_a_id, user_b_id)`, a.User.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, n)
		}
	}
}

func TestBlockReportAndAccountDeletion(t *testing.T) {
	e := setup(t)
	eve := e.register("eve")
	finn := e.register("finn")
	e.onboard(eve, "Eve", "woman", []string{"man"}, 44.84, -0.58, nil)
	e.onboard(finn, "Finn", "man", []string{"woman"}, 44.84, -0.57, nil)
	e.swipe(eve, finn.User.ID, "like")
	res, _ := e.swipe(finn, eve.User.ID, "like")
	if !res.Matched {
		t.Fatal("expected match")
	}
	matchID := res.Match.ID

	e.expect(e.do("POST", "/blocks", eve.AccessToken, map[string]string{"userId": eve.User.ID}), http.StatusBadRequest, "self block")
	e.expect(e.do("POST", "/blocks", eve.AccessToken, map[string]string{"userId": "00000000-0000-4000-8000-000000000000"}), http.StatusNotFound, "block unknown user")
	e.expect(e.do("POST", "/blocks", eve.AccessToken, map[string]string{"userId": finn.User.ID}), http.StatusNoContent, "block")
	e.expect(e.do("POST", "/blocks", eve.AccessToken, map[string]string{"userId": finn.User.ID}), http.StatusNoContent, "block is idempotent")
	e.expect(e.do("GET", "/conversations/"+matchID+"/messages", finn.AccessToken, nil), http.StatusNotFound, "blocked user loses the conversation")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", finn.AccessToken, map[string]string{"body": "hello?"}), http.StatusNotFound, "blocked user cannot write")
	e.expect(e.do("GET", "/profiles/"+eve.User.ID, finn.AccessToken, nil), http.StatusNotFound, "blocked user cannot view profile")
	var blocks struct {
		Blocks []struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"blocks"`
	}
	e.do("GET", "/blocks", eve.AccessToken, nil).decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].FirstName != "Finn" {
		t.Fatalf("unexpected blocks %+v", blocks)
	}
	e.expect(e.do("DELETE", "/blocks/"+finn.User.ID, eve.AccessToken, nil), http.StatusNoContent, "unblock")

	// Reports: invalid reason, then three distinct reporters suspend Finn.
	e.expect(e.do("POST", "/reports", eve.AccessToken, map[string]string{"userId": finn.User.ID, "reason": "dislike"}), http.StatusBadRequest, "invalid reason")
	viewer := e.register("viewer")
	e.onboard(viewer, "Gaia", "woman", []string{"man"}, 44.84, -0.58, nil)
	if !contains(e.discover(viewer), finn.User.ID) {
		t.Fatal("Finn should be discoverable before reports")
	}
	reporters := []member{eve, e.register("rep2"), e.register("rep3")}
	for _, rep := range reporters {
		e.expect(e.do("POST", "/reports", rep.AccessToken, map[string]string{"userId": finn.User.ID, "reason": "harassment", "details": "messages insistants"}), http.StatusCreated, "report")
	}
	if contains(e.discover(viewer), finn.User.ID) {
		t.Fatal("suspended account must be hidden from discovery")
	}

	// Account deletion.
	e.expect(e.do("DELETE", "/me", finn.AccessToken, map[string]string{"password": "wrong-password1"}), http.StatusForbidden, "deletion requires password")
	var key string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1 LIMIT 1`, finn.User.ID).Scan(&key)
	if _, err := os.Stat(e.uploadDir + "/" + key); err != nil {
		t.Fatalf("photo file should exist before deletion: %v", err)
	}
	finnWS := e.connect(finn.AccessToken)
	e.expect(e.do("DELETE", "/me", finn.AccessToken, map[string]string{"password": finn.password}), http.StatusNoContent, "delete account")
	e.expect(e.do("GET", "/me", finn.AccessToken, nil), http.StatusUnauthorized, "token invalid after deletion")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": finn.email, "password": finn.password}), http.StatusUnauthorized, "login after deletion")
	if _, err := os.Stat(e.uploadDir + "/" + key); !os.IsNotExist(err) {
		t.Fatal("photo files must be removed with the account")
	}
	select {
	case _, ok := <-finnWS.events:
		for ok {
			_, ok = <-finnWS.events
		}
	case <-time.After(3 * time.Second):
		t.Fatal("sockets must be closed after account deletion")
	}
	var leftovers int
	_ = e.pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM profiles WHERE user_id = $1) + (SELECT COUNT(*) FROM photos WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM swipes WHERE actor_id = $1 OR target_id = $1)
		     + (SELECT COUNT(*) FROM sessions WHERE user_id = $1)
	`, finn.User.ID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("personal data must be deleted, %d rows left", leftovers)
	}
	var keptReports int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1`, eve.User.ID).Scan(&keptReports)
	if keptReports != 1 {
		t.Fatal("reports are kept for moderation after the reported account is deleted")
	}
}

func TestRateLimitingCredentials(t *testing.T) {
	e := setup(t)
	// Dedicated router with production limits.
	router := newRouter(deps{pool: e.pool, jwtSecret: []byte(testSecret), store: mustStore(t), mailer: e.mail, hub: realtime.NewHub()}, defaultLimits())
	srv := httptest.NewServer(router)
	defer srv.Close()
	limited := false
	for i := 0; i < 15; i++ {
		res, err := http.Post(srv.URL+"/auth/login", "application/json", strings.NewReader(`{"email":"x@lueur.test","password":"Wrong12345"}`))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests {
			limited = true
			if res.Header.Get("Retry-After") == "" {
				t.Fatal("expected Retry-After header")
			}
			break
		}
	}
	if !limited {
		t.Fatal("login brute force must be rate limited")
	}
}

func TestWebSocketRejectsMissingAuth(t *testing.T) {
	e := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(e.srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"auth","token":"forged"}`))
	_, _, err = conn.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("expected policy violation close, got %v", err)
	}
}

func mustStore(t *testing.T) storage.Storage {
	s, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
