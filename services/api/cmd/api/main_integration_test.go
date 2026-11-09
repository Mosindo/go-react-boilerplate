package main

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
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	photosfeature "example.com/api/internal/features/photos"
	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "0123456789abcdef0123456789abcdef"

type captureMailer struct {
	mu   sync.Mutex
	last string
}

func (m *captureMailer) Send(_ context.Context, _, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last = body
	return nil
}

type env struct {
	t       *testing.T
	pool    *pgxpool.Pool
	router  *gin.Engine
	mailer  *captureMailer
	storage *photosfeature.DiskStorage
	dir     string
}

// newEnv needs DATABASE_URL_TEST (a throw-away database: all tables are truncated).
func newEnv(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST is not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("skip integration tests: postgres unavailable (%v)", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, matches CASCADE`); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	storage, err := photosfeature.NewDiskStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := &captureMailer{}
	r := setupRouter(&app{dbPool: pool, jwtSecret: []byte(testSecret), storage: storage, mailer: m})
	return &env{t: t, pool: pool, router: r, mailer: m, storage: storage, dir: dir}
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

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{w.Code, w.Body.Bytes()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func jpegBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return b.Bytes()
}

func (e *env) upload(method, path, token string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("photo", "x.jpg")
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{w.Code, w.Body.Bytes()}
}

type person struct {
	ID, Token, Refresh string
}

func (e *env) register(email string) person {
	e.t.Helper()
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "password123"}), 201)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.json(e.t, &out)
	return person{ID: out.User.ID, Token: out.AccessToken, Refresh: out.RefreshToken}
}

// onboard registers a fully set-up user (profile, preferences, one photo).
func (e *env) onboard(email, name, gender, birth string, seeking []string, lat, lon float64) person {
	e.t.Helper()
	p := e.register(email)
	e.expect(e.do("PUT", "/me/profile", p.Token, map[string]any{
		"firstName": name, "birthDate": birth, "gender": gender, "bio": "bio " + name, "city": "Lyon",
		"latitude": lat, "longitude": lon, "interestIds": []int{1, 2},
	}), 200)
	e.expect(e.do("PUT", "/me/preferences", p.Token, map[string]any{
		"interestedIn": seeking, "minAge": 18, "maxAge": 45, "maxDistanceKm": 100,
	}), 200)
	e.expect(e.upload("POST", "/me/photos", p.Token, jpegBytes(300, 400)), 201)
	return p
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	DistanceKm *int   `json:"distanceKm"`
}

func (e *env) discover(p person) []card {
	e.t.Helper()
	var out struct {
		Profiles []card `json:"profiles"`
	}
	e.expect(e.do("GET", "/discover?limit=20", p.Token, nil), 200).json(e.t, &out)
	return out.Profiles
}

func hasCard(cards []card, id string) bool {
	for _, c := range cards {
		if c.UserID == id {
			return true
		}
	}
	return false
}

type swipeOut struct {
	Matched        bool   `json:"matched"`
	MatchID        string `json:"matchId"`
	ConversationID string `json:"conversationId"`
	AlreadySwiped  bool   `json:"alreadySwiped"`
}

func (e *env) swipe(p person, target, action string) swipeOut {
	e.t.Helper()
	var out swipeOut
	e.expect(e.do("POST", "/swipes", p.Token, map[string]string{"targetUserId": target, "action": action}), 200).json(e.t, &out)
	return out
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)
	p := e.register("  Alice@Example.COM ")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "alice@example.com", "password": "password123"}), 409)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "x@example.com", "password": "short"}), 400)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.com", "password": "wrong-pass"}), 401)
	r := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "ALICE@example.com", "password": "password123"}), 200)
	var login struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	r.json(t, &login)
	var me struct {
		Email string `json:"email"`
	}
	e.expect(e.do("GET", "/me", login.AccessToken, nil), 200).json(t, &me)
	if me.Email != "alice@example.com" {
		t.Fatalf("email not normalized: %s", me.Email)
	}
	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": p.Refresh}), 200)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": p.Refresh}), 401) // rotated

	// password recovery
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": "nobody@example.com"}), 202)
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": "alice@example.com"}), 202)
	var token string
	for _, line := range strings.Split(e.mailer.last, "\n") {
		if len(line) == 43 {
			token = line
		}
	}
	if token == "" {
		t.Fatalf("no reset token in email: %q", e.mailer.last)
	}
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"token": "bad", "password": "newpassword1"}), 400)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "newpassword1"}), 204)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "newpassword2"}), 400) // single use
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.com", "password": "password123"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.com", "password": "newpassword1"}), 200)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), 401) // sessions revoked
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t)
	p := e.register("a@example.com")
	e.expect(e.do("GET", "/me/profile", p.Token, nil), 404)
	bad := map[string]any{"firstName": "Kid", "birthDate": time.Now().AddDate(-16, 0, 0).Format("2006-01-02"), "gender": "man"}
	e.expect(e.do("PUT", "/me/profile", p.Token, bad), 400)
	e.expect(e.do("PUT", "/me/profile", p.Token, map[string]any{"firstName": "Ok", "birthDate": "1990-01-01", "gender": "man", "interestIds": []int{9999}}), 400)

	a := e.onboard("alice@example.com", "Alice", "woman", "1995-05-05", []string{"man"}, 45.7640, 4.8357)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.7578, 4.8320)

	r := e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 200)
	body := string(r.Body)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "45.75", "4.83", "email"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public profile leaks %q: %s", forbidden, body)
		}
	}
	var pub struct {
		DistanceKm *int `json:"distanceKm"`
		Age        int  `json:"age"`
	}
	r.json(t, &pub)
	if pub.DistanceKm == nil || *pub.DistanceKm < 1 {
		t.Fatalf("expected approximate distance, got %+v", pub)
	}

	// hide distance
	e.expect(e.do("PUT", "/me/profile", b.Token, map[string]any{"firstName": "Bob", "birthDate": "1993-03-03", "gender": "man", "showDistance": false}), 200)
	r = e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 200)
	if strings.Contains(string(r.Body), "distanceKm") {
		t.Fatalf("distance shown although disabled")
	}
	e.expect(e.do("GET", "/profiles/not-a-uuid", a.Token, nil), 404)
}

func TestMatchingAndChatFlow(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("alice@example.com", "Alice", "woman", "1995-05-05", []string{"man"}, 45.76, 4.83)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.75, 4.84)
	c := e.onboard("carl@example.com", "Carl", "man", "1992-02-02", []string{"woman"}, 45.77, 4.85)
	far := e.onboard("far@example.com", "Far", "man", "1992-02-02", []string{"woman"}, 43.30, 5.37)      // Marseille: out of range
	old := e.onboard("old@example.com", "Old", "man", "1960-02-02", []string{"woman"}, 45.76, 4.83)      // age out of range
	other := e.onboard("same@example.com", "Sam", "woman", "1995-02-02", []string{"woman"}, 45.76, 4.83) // not seeking women... A seeks men

	cards := e.discover(a)
	if hasCard(cards, a.ID) || hasCard(cards, far.ID) || hasCard(cards, old.ID) || hasCard(cards, other.ID) {
		t.Fatalf("discover returned an ineligible profile: %+v", cards)
	}
	if !hasCard(cards, b.ID) || !hasCard(cards, c.ID) {
		t.Fatalf("discover is missing eligible profiles: %+v", cards)
	}
	if hasCard(e.discover(b), c.ID) {
		t.Fatal("men seeking women must not see other men")
	}

	// A likes B: no match yet, B vanishes from A's feed.
	if r := e.swipe(a, b.ID, "like"); r.Matched {
		t.Fatal("one-sided like must not match")
	}
	if hasCard(e.discover(a), b.ID) {
		t.Fatal("liked profile reappeared")
	}
	// A passes C: C vanishes too.
	e.swipe(a, c.ID, "pass")
	if hasCard(e.discover(a), c.ID) {
		t.Fatal("passed profile reappeared")
	}
	// repeating a like is idempotent
	if r := e.swipe(a, b.ID, "like"); !r.AlreadySwiped || r.Matched {
		t.Fatalf("repeat like: %+v", r)
	}
	// a changed mind is ignored: the first decision stands
	if r := e.swipe(a, b.ID, "pass"); !r.AlreadySwiped {
		t.Fatalf("repeat with other action: %+v", r)
	}
	if e.count(`SELECT count(*) FROM swipes WHERE from_user_id = $1 AND action = 'like'`, a.ID) != 1 {
		t.Fatal("duplicate like stored")
	}

	// B likes A: match.
	if !hasCard(e.discover(b), a.ID) {
		t.Fatal("B should see A (who liked B)")
	}
	m := e.swipe(b, a.ID, "like")
	if !m.Matched || m.MatchID == "" || m.ConversationID == "" {
		t.Fatalf("expected match: %+v", m)
	}
	again := e.swipe(b, a.ID, "like")
	if !again.Matched || again.MatchID != m.MatchID {
		t.Fatalf("replay must return same match: %+v", again)
	}
	if e.count(`SELECT count(*) FROM matches`) != 1 || e.count(`SELECT count(*) FROM conversations`) != 1 {
		t.Fatal("duplicate match/conversation")
	}
	if hasCard(e.discover(a), b.ID) || hasCard(e.discover(b), a.ID) {
		t.Fatal("matched users must leave discovery")
	}
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"targetUserId": a.ID, "action": "like"}), 400)
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"targetUserId": b.ID, "action": "super"}), 400)
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"targetUserId": "00000000-0000-0000-0000-000000000000", "action": "like"}), 404)

	// chat
	conv := m.ConversationID
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "   "}), 400)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": strings.Repeat("x", 2001)}), 400)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "Salut Bob"}), 201)

	// outsider cannot read, write, read-mark or hide
	for _, call := range []struct{ method, path string }{
		{"GET", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/read"},
		{"DELETE", "/conversations/" + conv},
	} {
		e.expect(e.do(call.method, call.path, c.Token, map[string]string{"body": "intruder"}), 404)
	}
	if e.count(`SELECT count(*) FROM messages`) != 1 {
		t.Fatal("outsider message stored")
	}

	var list struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body   string     `json:"body"`
				ReadAt *time.Time `json:"readAt"`
			} `json:"lastMessage"`
			Peer struct {
				FirstName string `json:"firstName"`
			} `json:"peer"`
		} `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 1 || list.Conversations[0].Peer.FirstName != "Alice" ||
		list.Conversations[0].LastMessage == nil || list.Conversations[0].LastMessage.Body != "Salut Bob" {
		t.Fatalf("unexpected conversations for B: %+v", list)
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 204)
	e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t, &list)
	if list.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread not cleared")
	}
	var msgs struct {
		Messages []struct {
			Body   string     `json:"body"`
			ReadAt *time.Time `json:"readAt"`
		} `json:"messages"`
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", a.Token, nil), 200).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.Messages[0].ReadAt == nil {
		t.Fatalf("A should see message as read: %+v", msgs)
	}

	// notifications
	var notifs struct {
		Notifications []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do("GET", "/notifications", a.Token, nil), 200).json(t, &notifs)
	if len(notifs.Notifications) != 1 || notifs.Notifications[0].Type != "match" || notifs.UnreadCount != 1 {
		t.Fatalf("A notifications: %+v", notifs)
	}
	e.expect(e.do("POST", "/notifications/"+notifs.Notifications[0].ID+"/read", c.Token, nil), 404) // not yours
	e.expect(e.do("POST", "/notifications/read", a.Token, nil), 204)
	e.expect(e.do("GET", "/notifications", a.Token, nil), 200).json(t, &notifs)
	if notifs.UnreadCount != 0 {
		t.Fatal("notifications not marked read")
	}

	// local deletion hides the conversation only for the deleting user, until a new message
	e.expect(e.do("DELETE", "/conversations/"+conv, b.Token, nil), 204)
	e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("hidden conversation still listed")
	}
	e.expect(e.do("GET", "/conversations", a.Token, nil), 200).json(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("hiding must be local")
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "Tu es là ?"}), 201)
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", b.Token, nil), 200).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.Messages[0].Body != "Tu es là ?" {
		t.Fatalf("B should only see messages after deletion: %+v", msgs)
	}

	// unmatch closes the conversation
	e.expect(e.do("DELETE", "/matches/"+m.MatchID, c.Token, nil), 404)
	e.expect(e.do("DELETE", "/matches/"+m.MatchID, b.Token, nil), 204)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "?"}), 404)
}

func TestSwipeRequiresCompleteProfile(t *testing.T) {
	e := newEnv(t)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.75, 4.84)
	raw := e.register("raw@example.com")
	e.expect(e.do("POST", "/swipes", raw.Token, map[string]string{"targetUserId": b.ID, "action": "like"}), 409)
	e.expect(e.do("GET", "/discover", raw.Token, nil), 200) // empty, not an error
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("alice@example.com", "Alice", "woman", "1995-05-05", []string{"man"}, 45.76, 4.83)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.75, 4.84)
	var wg sync.WaitGroup
	for _, pair := range [][2]person{{a, b}, {b, a}, {a, b}, {b, a}} {
		wg.Add(1)
		go func(from, to person) {
			defer wg.Done()
			e.do("POST", "/swipes", from.Token, map[string]string{"targetUserId": to.ID, "action": "like"})
		}(pair[0], pair[1])
	}
	wg.Wait()
	if e.count(`SELECT count(*) FROM matches`) != 1 || e.count(`SELECT count(*) FROM conversations`) != 1 {
		t.Fatal("race produced duplicate or missing match")
	}
	if e.count(`SELECT count(*) FROM notifications WHERE type = 'match'`) != 2 {
		t.Fatal("expected exactly one match notification per user")
	}
}

func TestBlockAndReport(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("alice@example.com", "Alice", "woman", "1995-05-05", []string{"man"}, 45.76, 4.83)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.75, 4.84)
	e.swipe(a, b.ID, "like")
	m := e.swipe(b, a.ID, "like")
	if !m.Matched {
		t.Fatal("setup: no match")
	}
	var bobPhoto string
	_ = e.pool.QueryRow(context.Background(), `SELECT id::text FROM photos WHERE user_id = $1`, b.ID).Scan(&bobPhoto)
	e.expect(e.do("GET", "/photos/"+bobPhoto, a.Token, nil), 200)

	e.expect(e.do("PUT", "/users/"+a.ID+"/block", a.Token, nil), 400) // self
	e.expect(e.do("PUT", "/users/"+b.ID+"/block", a.Token, nil), 204)
	e.expect(e.do("PUT", "/users/"+b.ID+"/block", a.Token, nil), 204) // idempotent

	// everything between the two disappears, in both directions
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 404)
	e.expect(e.do("GET", "/photos/"+bobPhoto, a.Token, nil), 404)
	e.expect(e.do("POST", "/conversations/"+m.ConversationID+"/messages", b.Token, map[string]string{"body": "hi"}), 404)
	e.expect(e.do("POST", "/conversations/"+m.ConversationID+"/messages", a.Token, map[string]string{"body": "hi"}), 404)
	e.expect(e.do("POST", "/swipes", b.Token, map[string]string{"targetUserId": a.ID, "action": "like"}), 404)
	if hasCard(e.discover(b), a.ID) || hasCard(e.discover(a), b.ID) {
		t.Fatal("blocked users still in discovery")
	}
	var blocks struct {
		Blocks []struct {
			UserID string `json:"userId"`
		} `json:"blocks"`
	}
	e.expect(e.do("GET", "/me/blocks", a.Token, nil), 200).json(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != b.ID {
		t.Fatalf("blocks: %+v", blocks)
	}

	// report
	e.expect(e.do("POST", "/users/"+b.ID+"/report", a.Token, map[string]any{"reason": "nonsense"}), 400)
	e.expect(e.do("POST", "/users/"+a.ID+"/report", a.Token, map[string]any{"reason": "spam"}), 400)
	e.expect(e.do("POST", "/users/00000000-0000-0000-0000-000000000000/report", a.Token, map[string]any{"reason": "spam"}), 404)
	e.expect(e.do("POST", "/users/"+b.ID+"/report", a.Token, map[string]any{"reason": "harassment", "details": "rude"}), 201)
	if e.count(`SELECT count(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID) != 1 {
		t.Fatal("report not stored")
	}

	// unblock restores visibility but not the ended match
	e.expect(e.do("DELETE", "/users/"+b.ID+"/block", a.Token, nil), 204)
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 200)
	e.expect(e.do("POST", "/conversations/"+m.ConversationID+"/messages", a.Token, map[string]string{"body": "hi"}), 404)

	// report + block in one call
	c := e.onboard("carl@example.com", "Carl", "man", "1992-02-02", []string{"woman"}, 45.77, 4.85)
	e.expect(e.do("POST", "/users/"+c.ID+"/report", a.Token, map[string]any{"reason": "fake", "block": true}), 201)
	e.expect(e.do("GET", "/profiles/"+c.ID, a.Token, nil), 404)
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	a := e.register("a@example.com")
	e.expect(e.do("PUT", "/me/profile", a.Token, map[string]any{"firstName": "A", "birthDate": "1990-01-01", "gender": "woman"}), 200)

	e.expect(e.upload("POST", "/me/photos", a.Token, []byte("<?php evil() ?>")), 400)
	e.expect(e.upload("POST", "/me/photos", a.Token, bytes.Repeat([]byte{0xff}, 9<<20)), 400)

	var ids []string
	for i := 0; i < 6; i++ {
		var ph struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		}
		e.expect(e.upload("POST", "/me/photos", a.Token, jpegBytes(200+i, 300)), 201).json(t, &ph)
		if ph.Position != i {
			t.Fatalf("position %d != %d", ph.Position, i)
		}
		ids = append(ids, ph.ID)
	}
	e.expect(e.upload("POST", "/me/photos", a.Token, jpegBytes(200, 300)), 409)

	type photoList struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	var pl photoList
	e.expect(e.do("PUT", "/me/photos/"+ids[3]+"/primary", a.Token, nil), 200).json(t, &pl)
	if pl.Photos[0].ID != ids[3] || len(pl.Photos) != 6 {
		t.Fatalf("primary not first: %+v", pl)
	}
	e.expect(e.do("PUT", "/me/photos/order", a.Token, map[string]any{"ids": ids[:5]}), 400) // incomplete
	rev := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	e.expect(e.do("PUT", "/me/photos/order", a.Token, map[string]any{"ids": rev}), 200).json(t, &pl)
	if pl.Photos[0].ID != ids[5] || pl.Photos[5].ID != ids[0] {
		t.Fatalf("reorder failed: %+v", pl)
	}

	// replace keeps id and position, swaps the file
	var oldKey, newKey string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[0]).Scan(&oldKey)
	e.expect(e.upload("PUT", "/me/photos/"+ids[0], a.Token, jpegBytes(250, 250)), 200)
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[0]).Scan(&newKey)
	if oldKey == newKey {
		t.Fatal("file not replaced")
	}
	if _, err := e.storage.Read(oldKey); err == nil {
		t.Fatal("old file still on disk")
	}

	// delete compacts positions and removes the file
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[5]).Scan(&oldKey)
	e.expect(e.do("DELETE", "/me/photos/"+ids[5], a.Token, nil), 204)
	e.expect(e.do("GET", "/me/photos", a.Token, nil), 200).json(t, &pl)
	if len(pl.Photos) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(pl.Photos))
	}
	for i, p := range pl.Photos {
		if p.Position != i {
			t.Fatalf("positions not compact: %+v", pl)
		}
	}
	if _, err := e.storage.Read(oldKey); err == nil {
		t.Fatal("deleted file still on disk")
	}

	// stranger without a profile/visibility cannot fetch; owner can; others cannot delete
	other := e.onboard("o@example.com", "O", "man", "1990-01-01", []string{"woman"}, 45.7, 4.8)
	e.expect(e.do("DELETE", "/me/photos/"+ids[0], other.Token, nil), 404)
	r := e.expect(e.do("GET", "/photos/"+ids[0], a.Token, nil), 200)
	if _, _, err := image.Decode(bytes.NewReader(r.Body)); err != nil {
		t.Fatal("served photo is not an image")
	}
	e.expect(e.do("GET", "/photos/"+ids[0], "", nil), 401)
	e.expect(e.do("GET", "/photos/../../etc/passwd", a.Token, nil), 404)

	// hidden profile: photos not visible to strangers
	e.expect(e.do("PUT", "/me/profile", a.Token, map[string]any{"firstName": "A", "birthDate": "1990-01-01", "gender": "woman", "isVisible": false}), 200)
	e.expect(e.do("GET", "/photos/"+ids[0], other.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+a.ID, other.Token, nil), 404)
}

func TestDeleteAccount(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("alice@example.com", "Alice", "woman", "1995-05-05", []string{"man"}, 45.76, 4.83)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.75, 4.84)
	e.swipe(a, b.ID, "like")
	m := e.swipe(b, a.ID, "like")
	e.expect(e.do("POST", "/conversations/"+m.ConversationID+"/messages", a.Token, map[string]string{"body": "hello"}), 201)
	var key string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID).Scan(&key)

	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong-password"}), 403)
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "password123"}), 204)

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.com", "password": "password123"}), 401)
	for _, q := range []string{
		`SELECT count(*) FROM users WHERE id = $1`, `SELECT count(*) FROM profiles WHERE user_id = $1`,
		`SELECT count(*) FROM photos WHERE user_id = $1`, `SELECT count(*) FROM swipes WHERE from_user_id = $1 OR to_user_id = $1`,
		`SELECT count(*) FROM matches WHERE user_a_id = $1 OR user_b_id = $1`, `SELECT count(*) FROM messages WHERE sender_id = $1`,
		`SELECT count(*) FROM notifications WHERE user_id = $1 OR actor_id = $1`, `SELECT count(*) FROM sessions WHERE user_id = $1`,
	} {
		if n := e.count(q, a.ID); n != 0 {
			t.Fatalf("%s left %d rows", q, n)
		}
	}
	if e.count(`SELECT count(*) FROM conversations`) != 0 {
		t.Fatal("conversation survived")
	}
	if _, err := e.storage.Read(key); err == nil {
		t.Fatal("photo file survived account deletion")
	}
	var list struct {
		Conversations []any `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("B still sees deleted user's conversation")
	}
	e.expect(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 404)
}

func TestResetPasses(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("alice@example.com", "Alice", "woman", "1995-05-05", []string{"man"}, 45.76, 4.83)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.75, 4.84)
	e.swipe(a, b.ID, "pass")
	if hasCard(e.discover(a), b.ID) {
		t.Fatal("passed profile visible")
	}
	e.expect(e.do("POST", "/discover/reset", a.Token, nil), 200)
	if !hasCard(e.discover(a), b.ID) {
		t.Fatal("reset must bring passed profiles back")
	}
}

func TestRealtimeMessageDelivery(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("alice@example.com", "Alice", "woman", "1995-05-05", []string{"man"}, 45.76, 4.83)
	b := e.onboard("bob@example.com", "Bob", "man", "1993-03-03", []string{"woman"}, 45.75, 4.84)
	e.swipe(a, b.ID, "like")

	srv := httptest.NewServer(e.router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	dial := func(token string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.WriteJSON(map[string]string{"type": "auth", "token": token})
		var ev struct{ Type string }
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := conn.ReadJSON(&ev); err != nil || ev.Type != "ready" {
			t.Fatalf("ws auth failed: %v %+v", err, ev)
		}
		return conn
	}
	wsB := dial(b.Token)
	defer wsB.Close()

	// invalid token is rejected
	bad, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "nope"})
	var ev struct{ Type string }
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	_ = bad.ReadJSON(&ev)
	if ev.Type != "error" {
		t.Fatalf("expected error event, got %+v", ev)
	}
	bad.Close()

	m := e.swipe(b, a.ID, "like")
	read := func() map[string]any {
		var msg map[string]any
		_ = wsB.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := wsB.ReadJSON(&msg); err != nil {
			t.Fatalf("no ws event: %v", err)
		}
		return msg
	}
	if got := read(); got["type"] != "match" {
		t.Fatalf("expected match event, got %v", got)
	}
	e.expect(e.do("POST", "/conversations/"+m.ConversationID+"/messages", a.Token, map[string]string{"body": "live!"}), 201)
	got := read()
	data, _ := got["data"].(map[string]any)
	if got["type"] != "message" || fmt.Sprint(data["body"]) != "live!" {
		t.Fatalf("expected message event, got %v", got)
	}
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	os.Exit(m.Run())
}
