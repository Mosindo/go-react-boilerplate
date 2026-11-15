package app_test

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
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-0123456789abcdef0123456789"

// captureMailer records outgoing e-mails so the reset flow can be tested.
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

func (m *captureMailer) code(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return regexp.MustCompile(`\b\d{6}\b`).FindString(m.last[to])
}

type env struct {
	t      *testing.T
	server *httptest.Server
	pool   *pgxpool.Pool
	mail   *captureMailer
	app    *app.App
	suffix string
	dir    string
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
	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	dir := t.TempDir()
	store, err := storage.NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	mail := &captureMailer{}
	a, err := app.New(app.Options{
		DB: pool, JWTSecret: []byte(testSecret), Store: store, Mailer: mail,
		BcryptCost: 4, DisableRateLimit: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, server: httptest.NewServer(a.Router), pool: pool, mail: mail, app: a, dir: dir,
		suffix: fmt.Sprintf("%d@it.test", time.Now().UnixNano())}
	t.Cleanup(func() {
		e.server.Close()
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email LIKE '%' || $1`, e.suffix)
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
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *env) do(method, path, token string, body any) response {
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
	return e.send(req)
}

func (e *env) send(req *http.Request) response {
	e.t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return response{Status: resp.StatusCode, Body: data}
}

func (e *env) expect(r response, status int, context string) {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("%s: expected %d, got %d: %s", context, status, r.Status, r.Body)
	}
}

func (e *env) upload(method, path, token string, data []byte) response {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", "photo.png")
	_, _ = part.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(method, e.server.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.send(req)
}

type user struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

func (e *env) register(name string) user {
	e.t.Helper()
	email := name + "." + e.suffix
	r := e.do("POST", "/auth/register", "", map[string]any{"email": "  " + strings.ToUpper(email) + " ", "password": "Secret123", "acceptTerms": true})
	e.expect(r, 201, "register "+name)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	r.decode(e.t, &out)
	if out.User.Email != email {
		e.t.Fatalf("email not normalised: %q", out.User.Email)
	}
	return user{ID: out.User.ID, Email: email, Token: out.AccessToken, Refresh: out.RefreshToken}
}

func (e *env) completeProfile(u user, name, gender string, interestedIn []string, interests []int, lat, lng float64) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": name, "birthdate": "1995-04-12", "gender": gender,
		"bio": "Bonjour, je suis " + name, "interestIds": interests,
	}), 200, "profile "+name)
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": 50,
	}), 200, "preferences "+name)
	e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": lat, "longitude": lng, "city": "Paris"}), 200, "location "+name)
	e.expect(e.upload("POST", "/me/photos", u.Token, testPNG(e.t, 400, 500)), 201, "photo "+name)
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
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
	Interests []struct {
		ID int `json:"id"`
	} `json:"interests"`
}

func (e *env) discover(u user) []card {
	e.t.Helper()
	r := e.do("GET", "/discovery?limit=30", u.Token, nil)
	e.expect(r, 200, "discovery")
	var out struct {
		Profiles []card `json:"profiles"`
	}
	r.decode(e.t, &out)
	return out.Profiles
}

func containsCard(cards []card, userID string) bool {
	for _, c := range cards {
		if c.UserID == userID {
			return true
		}
	}
	return false
}

type wsClient struct {
	conn   *websocket.Conn
	events chan map[string]any
}

func (e *env) connectWS(token string) *wsClient {
	e.t.Helper()
	url := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		e.t.Fatalf("dial ws: %v", err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		e.t.Fatal(err)
	}
	c := &wsClient{conn: conn, events: make(chan map[string]any, 32)}
	go func() {
		for {
			var ev map[string]any
			if err := conn.ReadJSON(&ev); err != nil {
				close(c.events)
				return
			}
			c.events <- ev
		}
	}()
	c.waitFor(e.t, "ready")
	e.t.Cleanup(func() { _ = conn.Close() })
	return c
}

func (c *wsClient) waitFor(t *testing.T, eventType string) map[string]any {
	t.Helper()
	timeout := time.After(3 * time.Second)
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
			t.Fatalf("timed out waiting for %s", eventType)
		}
	}
}

func TestDatingCriticalFlow(t *testing.T) {
	e := setup(t)

	e.expect(e.do("GET", "/health", "", nil), 200, "health")

	// --- Registration rules -------------------------------------------------
	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": "x." + e.suffix, "password": "Secret123"}), 400, "terms required")
	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": "y." + e.suffix, "password": "allletters", "acceptTerms": true}), 400, "weak password")

	alice := e.register("alice")
	bob := e.register("bob")
	carol := e.register("carol")
	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": alice.Email, "password": "Secret123", "acceptTerms": true}), 409, "duplicate email")

	// --- Login / session ------------------------------------------------------
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": alice.Email, "password": "WrongPass1"}), 401, "bad password")
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": "nobody." + e.suffix, "password": "WrongPass1"}), 401, "unknown user")
	e.expect(e.do("GET", "/me", alice.Token, nil), 200, "me")
	e.expect(e.do("GET", "/me", "", nil), 401, "me without token")
	e.expect(e.do("GET", "/me", "garbage", nil), 401, "me with garbage token")

	// --- Profile validation -------------------------------------------------
	e.expect(e.do("GET", "/discovery", alice.Token, nil), 409, "discovery with incomplete profile")
	young := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", alice.Token, map[string]any{"firstName": "Alice", "birthdate": young, "gender": "woman"}), 400, "underage")
	e.expect(e.do("PUT", "/me/profile", alice.Token, map[string]any{"firstName": "Alice", "gender": "robot"}), 400, "bad gender")
	e.expect(e.do("PUT", "/me/profile", alice.Token, map[string]any{"firstName": "<script>"}), 400, "bad name")
	e.expect(e.do("PUT", "/me/profile", alice.Token, map[string]any{"firstName": "Alice", "interestIds": []int{9999}}), 400, "unknown interest")
	e.expect(e.do("PUT", "/me/preferences", alice.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10}), 400, "bad age range")
	e.expect(e.do("PUT", "/me/location", alice.Token, map[string]any{"latitude": 123.0, "longitude": 2.0}), 400, "bad latitude")

	// --- Photos --------------------------------------------------------------
	e.expect(e.upload("POST", "/me/photos", alice.Token, []byte("<?php echo 'pwned'; ?>")), 400, "upload non-image")
	e.expect(e.upload("POST", "/me/photos", alice.Token, testPNG(t, 50, 50)), 400, "upload tiny image")

	e.completeProfile(alice, "Alice", "woman", []string{"man"}, []int{1, 2, 3}, 48.8566, 2.3522)
	e.completeProfile(bob, "Bob", "man", []string{"woman"}, []int{2, 3, 4}, 48.8600, 2.3400)
	e.completeProfile(carol, "Carol", "man", []string{"woman"}, []int{5}, 48.8700, 2.3600)

	// Exact coordinates are never stored.
	var lat float64
	if err := e.pool.QueryRow(context.Background(), `SELECT approx_latitude FROM profiles WHERE user_id = $1`, alice.ID).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 {
		t.Fatalf("expected rounded latitude 48.86, got %v", lat)
	}

	var own struct {
		Profile struct {
			Age    *int `json:"age"`
			Photos []struct {
				ID  string `json:"id"`
				URL string `json:"url"`
			} `json:"photos"`
		} `json:"profile"`
		Completeness struct {
			IsComplete bool `json:"isComplete"`
		} `json:"completeness"`
	}
	r := e.do("GET", "/me/profile", alice.Token, nil)
	e.expect(r, 200, "own profile")
	r.decode(t, &own)
	if !own.Completeness.IsComplete || own.Profile.Age == nil || len(own.Profile.Photos) != 1 {
		t.Fatalf("unexpected own profile: %s", r.Body)
	}
	e.expect(e.do("PUT", "/me/profile", alice.Token, map[string]any{"firstName": "Alice", "birthdate": "1990-01-01", "gender": "woman"}), 400, "birthdate is locked")

	// Media: signed URL works, tampered signature does not, and the stored
	// file is a re-encoded JPEG.
	mediaURL := own.Profile.Photos[0].URL
	mediaResp := e.do("GET", mediaURL, "", nil)
	e.expect(mediaResp, 200, "media")
	if !bytes.HasPrefix(mediaResp.Body, []byte{0xFF, 0xD8, 0xFF}) {
		t.Fatal("stored photo should be a JPEG")
	}
	e.expect(e.do("GET", mediaURL[:len(mediaURL)-4]+"0000", "", nil), 404, "tampered media signature")
	e.expect(e.do("GET", "/media/../../etc/passwd", "", nil), 404, "path traversal")

	// --- Discovery --------------------------------------------------------------
	aliceFeed := e.discover(alice)
	if !containsCard(aliceFeed, bob.ID) || !containsCard(aliceFeed, carol.ID) || containsCard(aliceFeed, alice.ID) {
		t.Fatalf("unexpected discovery feed for alice: %+v", aliceFeed)
	}
	for _, c := range aliceFeed {
		if c.UserID == bob.ID && (c.DistanceKm == nil || *c.DistanceKm < 1 || c.Age < 18 || len(c.Photos) == 0) {
			t.Fatalf("bob card is missing data: %+v", c)
		}
	}
	if containsCard(e.discover(bob), carol.ID) {
		t.Fatal("bob (man seeking women) must not see carol (man)")
	}

	// --- Like / match -----------------------------------------------------------
	aliceWS := e.connectWS(alice.Token)
	bobWS := e.connectWS(bob.Token)

	e.expect(e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}), 400, "self like")
	r = e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"})
	e.expect(r, 200, "alice likes bob")
	var swipe struct {
		Matched bool `json:"matched"`
		Match   *struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversationId"`
		} `json:"match"`
	}
	r.decode(t, &swipe)
	if swipe.Matched {
		t.Fatal("one-sided like must not match")
	}
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"}), 409, "repeated like")
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "pass"}), 409, "pass after like")
	if containsCard(e.discover(alice), bob.ID) {
		t.Fatal("already-swiped profile must not reappear")
	}
	bobWS.waitFor(t, "notification.new")

	var likes struct {
		Likes []struct {
			User card `json:"user"`
		} `json:"likes"`
	}
	r = e.do("GET", "/likes/received", bob.Token, nil)
	e.expect(r, 200, "likes received")
	r.decode(t, &likes)
	if len(likes.Likes) != 1 || likes.Likes[0].User.UserID != alice.ID {
		t.Fatalf("bob should see alice's like: %s", r.Body)
	}

	r = e.do("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"})
	e.expect(r, 200, "bob likes alice")
	r.decode(t, &swipe)
	if !swipe.Matched || swipe.Match == nil || swipe.Match.ConversationID == "" {
		t.Fatalf("mutual like must match: %s", r.Body)
	}
	conversationID := swipe.Match.ConversationID
	matchID := swipe.Match.ID
	aliceWS.waitFor(t, "match.new")
	bobWS.waitFor(t, "match.new")

	var matchCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE $1 IN (user_a_id, user_b_id) AND $2 IN (user_a_id, user_b_id)`, alice.ID, bob.ID).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchCount)
	}

	// --- Chat --------------------------------------------------------------------
	e.expect(e.do("POST", "/conversations/"+conversationID+"/messages", alice.Token, map[string]any{"body": "   "}), 400, "empty message")
	r = e.do("POST", "/conversations/"+conversationID+"/messages", alice.Token, map[string]any{"body": "Salut Bob ! 👋"})
	e.expect(r, 201, "alice sends message")
	ev := bobWS.waitFor(t, "message.new")
	if data, _ := ev["data"].(map[string]any); data["body"] != "Salut Bob ! 👋" {
		t.Fatalf("unexpected realtime payload: %v", ev)
	}

	// Carol is not a participant: she can neither read nor write.
	e.expect(e.do("GET", "/conversations/"+conversationID+"/messages", carol.Token, nil), 404, "outsider reads")
	e.expect(e.do("POST", "/conversations/"+conversationID+"/messages", carol.Token, map[string]any{"body": "intrusion"}), 404, "outsider writes")
	e.expect(e.do("GET", "/conversations/"+conversationID, carol.Token, nil), 404, "outsider opens")
	e.expect(e.do("POST", "/conversations/"+conversationID+"/read", carol.Token, nil), 404, "outsider marks read")

	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			User        card   `json:"user"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	r = e.do("GET", "/conversations", bob.Token, nil)
	e.expect(r, 200, "bob conversations")
	r.decode(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].User.UserID != alice.ID {
		t.Fatalf("unexpected conversation list: %s", r.Body)
	}

	var badges struct {
		UnreadConversations int `json:"unreadConversations"`
		UnreadNotifications int `json:"unreadNotifications"`
	}
	r = e.do("GET", "/me/badges", bob.Token, nil)
	e.expect(r, 200, "badges")
	r.decode(t, &badges)
	if badges.UnreadConversations != 1 || badges.UnreadNotifications < 2 {
		t.Fatalf("unexpected badges: %s", r.Body)
	}

	e.expect(e.do("POST", "/conversations/"+conversationID+"/read", bob.Token, nil), 200, "bob reads")
	aliceWS.waitFor(t, "conversation.read")
	r = e.do("GET", "/conversations", bob.Token, nil)
	r.decode(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("conversation should be read")
	}

	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/conversations/"+conversationID+"/messages", bob.Token, map[string]any{"body": fmt.Sprintf("message %d", i)}), 201, "bob sends")
	}
	var page struct {
		Messages []struct {
			Body string `json:"body"`
		} `json:"messages"`
		NextCursor *string `json:"nextCursor"`
	}
	r = e.do("GET", "/conversations/"+conversationID+"/messages?limit=4", alice.Token, nil)
	e.expect(r, 200, "messages page 1")
	r.decode(t, &page)
	if len(page.Messages) != 4 || page.NextCursor == nil || page.Messages[0].Body != "message 4" {
		t.Fatalf("unexpected first page: %s", r.Body)
	}
	r = e.do("GET", "/conversations/"+conversationID+"/messages?limit=4&cursor="+*page.NextCursor, alice.Token, nil)
	r.decode(t, &page)
	if len(page.Messages) != 2 || page.NextCursor != nil || page.Messages[1].Body != "Salut Bob ! 👋" {
		t.Fatalf("unexpected second page: %s", r.Body)
	}

	// Local deletion hides history for Alice only.
	e.expect(e.do("DELETE", "/conversations/"+conversationID, alice.Token, nil), 204, "alice clears")
	r = e.do("GET", "/conversations/"+conversationID+"/messages", alice.Token, nil)
	r.decode(t, &page)
	if len(page.Messages) != 0 {
		t.Fatal("cleared conversation should be empty for alice")
	}
	r = e.do("GET", "/conversations/"+conversationID+"/messages", bob.Token, nil)
	r.decode(t, &page)
	if len(page.Messages) != 6 {
		t.Fatalf("bob keeps his history, got %d", len(page.Messages))
	}

	// --- Notifications -----------------------------------------------------------
	var notifs struct {
		Notifications []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"notifications"`
	}
	r = e.do("GET", "/notifications", alice.Token, nil)
	e.expect(r, 200, "notifications")
	r.decode(t, &notifs)
	types := map[string]bool{}
	for _, n := range notifs.Notifications {
		types[n.Type] = true
	}
	if !types["match"] || !types["message"] {
		t.Fatalf("expected match and message notifications: %s", r.Body)
	}
	e.expect(e.do("POST", "/notifications/"+notifs.Notifications[0].ID+"/read", bob.Token, nil), 404, "read someone else's notification")
	e.expect(e.do("POST", "/notifications/"+notifs.Notifications[0].ID+"/read", alice.Token, nil), 204, "read own notification")
	e.expect(e.do("POST", "/notifications/read-all", alice.Token, nil), 204, "read all")
	e.expect(e.do("POST", "/notifications", alice.Token, map[string]any{"type": "match"}), 404, "clients cannot forge notifications")

	// --- Block ---------------------------------------------------------------
	e.expect(e.do("GET", "/profiles/"+carol.ID, alice.Token, nil), 200, "view carol")
	e.expect(e.do("POST", "/blocks", alice.Token, map[string]any{"userId": carol.ID}), 204, "alice blocks carol")
	e.expect(e.do("GET", "/profiles/"+alice.ID, carol.Token, nil), 404, "blocked user cannot view")
	e.expect(e.do("GET", "/profiles/"+carol.ID, alice.Token, nil), 404, "blocker no longer sees")
	if containsCard(e.discover(carol), alice.ID) {
		t.Fatal("blocked user must not discover the blocker")
	}
	e.expect(e.do("POST", "/swipes", carol.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}), 404, "blocked user cannot like")
	var blocks struct {
		Blocks []struct {
			UserID string `json:"userId"`
		} `json:"blocks"`
	}
	r = e.do("GET", "/blocks", alice.Token, nil)
	r.decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != carol.ID {
		t.Fatalf("unexpected blocks: %s", r.Body)
	}
	e.expect(e.do("DELETE", "/blocks/"+carol.ID, alice.Token, nil), 204, "unblock")

	// --- Report (also blocks and ends the match) ---------------------------------
	e.expect(e.do("POST", "/reports", alice.Token, map[string]any{"userId": bob.ID, "reason": "nonsense"}), 400, "bad reason")
	e.expect(e.do("POST", "/reports", alice.Token, map[string]any{"userId": bob.ID, "reason": "harassment", "details": "Messages insistants"}), 201, "report bob")
	bobWS.waitFor(t, "match.removed")
	e.expect(e.do("GET", "/conversations/"+conversationID+"/messages", bob.Token, nil), 404, "conversation gone after report")
	e.expect(e.do("DELETE", "/matches/"+matchID, bob.Token, nil), 404, "match gone after report")

	// --- Photos management -------------------------------------------------------
	e.expect(e.upload("POST", "/me/photos", alice.Token, testPNG(t, 600, 400)), 201, "second photo")
	var photos struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	r = e.do("GET", "/me/photos", alice.Token, nil)
	r.decode(t, &photos)
	if len(photos.Photos) != 2 {
		t.Fatalf("expected 2 photos: %s", r.Body)
	}
	first, second := photos.Photos[0].ID, photos.Photos[1].ID
	e.expect(e.do("PUT", "/me/photos/order", alice.Token, map[string]any{"photoIds": []string{first}}), 400, "incomplete order")
	r = e.do("PUT", "/me/photos/order", alice.Token, map[string]any{"photoIds": []string{second, first}})
	e.expect(r, 200, "reorder")
	r.decode(t, &photos)
	if photos.Photos[0].ID != second {
		t.Fatal("reorder should make the second photo primary")
	}
	e.expect(e.upload("PUT", "/me/photos/"+first, alice.Token, testPNG(t, 300, 300)), 200, "replace photo")
	e.expect(e.do("DELETE", "/me/photos/"+second, bob.Token, nil), 404, "delete someone else's photo")
	r = e.do("DELETE", "/me/photos/"+second, alice.Token, nil)
	e.expect(r, 200, "delete photo")
	r.decode(t, &photos)
	if len(photos.Photos) != 1 || photos.Photos[0].Position != 0 {
		t.Fatalf("positions should be compacted: %s", r.Body)
	}

	// --- Refresh & logout ---------------------------------------------------
	r = e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": carol.Refresh})
	e.expect(r, 200, "refresh")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": carol.Refresh}), 401, "refresh token reuse")
	var refreshed struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	r.decode(t, &refreshed)
	e.expect(e.do("POST", "/auth/logout", "", map[string]any{"refreshToken": refreshed.RefreshToken}), 204, "logout")
	e.expect(e.do("GET", "/me", refreshed.AccessToken, nil), 401, "access token revoked by logout")

	// --- Password reset -----------------------------------------------------
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]any{"email": "ghost." + e.suffix}), 202, "forgot unknown")
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]any{"email": bob.Email}), 202, "forgot bob")
	code := e.mail.code(bob.Email)
	if code == "" {
		t.Fatal("reset code not sent")
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]any{"email": bob.Email, "code": wrong, "newPassword": "NewSecret456"}), 400, "wrong code")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]any{"email": bob.Email, "code": code, "newPassword": "NewSecret456"}), 204, "reset")
	e.expect(e.do("GET", "/me", bob.Token, nil), 401, "sessions revoked after reset")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]any{"email": bob.Email, "code": code, "newPassword": "Other789xx"}), 400, "code is single use")
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": bob.Email, "password": "NewSecret456"}), 200, "login with new password")

	// --- Account deletion --------------------------------------------------------
	e.expect(e.do("DELETE", "/me", alice.Token, map[string]any{"password": "nope-nope1"}), 403, "delete with wrong password")
	var keys []string
	rows, _ := e.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, alice.ID)
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	e.expect(e.do("DELETE", "/me", alice.Token, map[string]any{"password": "Secret123"}), 204, "delete account")
	e.expect(e.do("GET", "/me", alice.Token, nil), 401, "token dead after deletion")
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": alice.Email, "password": "Secret123"}), 401, "login after deletion")
	for _, k := range keys {
		if _, err := os.Stat(e.dir + "/" + k[:2] + "/" + k); !os.IsNotExist(err) {
			t.Fatalf("photo file %s should be deleted", k)
		}
	}
	var remaining int
	_ = e.pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1)
		     + (SELECT COUNT(*) FROM notifications WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM photos WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM messages WHERE sender_id = $1)
	`, alice.ID).Scan(&remaining)
	if remaining != 0 {
		t.Fatalf("personal data should be erased, %d rows remain", remaining)
	}
	var reportKept int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reported_id = $1 AND reporter_id IS NULL`, bob.ID).Scan(&reportKept)
	if reportKept != 1 {
		t.Fatal("report should survive reporter deletion (anonymised)")
	}
}

func TestConcurrentMutualLikesCreateSingleMatch(t *testing.T) {
	e := setup(t)
	dana := e.register("dana")
	evan := e.register("evan")
	e.completeProfile(dana, "Dana", "woman", []string{"man"}, []int{1}, 45.76, 4.84)
	e.completeProfile(evan, "Evan", "man", []string{"woman"}, []int{1}, 45.75, 4.85)

	var wg sync.WaitGroup
	results := make([]bool, 2)
	for i, pair := range [][2]user{{dana, evan}, {evan, dana}} {
		wg.Add(1)
		go func(i int, from, to user) {
			defer wg.Done()
			r := e.do("POST", "/swipes", from.Token, map[string]any{"targetUserId": to.ID, "action": "like"})
			var out struct {
				Matched bool `json:"matched"`
			}
			_ = json.Unmarshal(r.Body, &out)
			results[i] = out.Matched
		}(i, pair[0], pair[1])
	}
	wg.Wait()

	if results[0] == results[1] {
		t.Fatalf("exactly one of the two concurrent likes should report the match, got %v", results)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE $1 IN (user_a_id, user_b_id) AND $2 IN (user_a_id, user_b_id)`, dana.ID, evan.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected exactly one match, got %d", n)
	}
}

func TestWebSocketRejectsInvalidToken(t *testing.T) {
	e := setup(t)
	url := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": "invalid"})
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if ce, ok := err.(*websocket.CloseError); !ok || ce.Code != 4001 {
		t.Fatalf("expected close 4001, got %v", err)
	}
}
