package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// recordingMailer captures outgoing recovery emails.
type recordingMailer struct {
	mu   sync.Mutex
	last string
}

func (m *recordingMailer) Send(_ context.Context, _, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last = body
	return nil
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	mail   *recordingMailer
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration tests")
	}
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Start every test from a clean slate (reference data in `interests` stays).
	if _, err := pool.Exec(ctx, `TRUNCATE users, reports CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	mail := &recordingMailer{}
	router, err := NewRouter(Deps{
		Pool: pool, Mailer: mail, LimitScale: 1000,
		Config: config.Config{JWTSecret: "integration-secret-0123456789abcdef-xyz", UploadDir: t.TempDir()},
	})
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	return &env{t: t, pool: pool, router: router, mail: mail}
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
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

type user struct {
	ID, Token, Refresh, Email string
}

var emailSeq int

func (e *env) register(prefix string) user {
	e.t.Helper()
	emailSeq++
	email := fmt.Sprintf("%s%d@test.local", prefix, emailSeq)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"}), 201).json(e.t, &out)
	return user{ID: out.User.ID, Token: out.AccessToken, Refresh: out.RefreshToken, Email: email}
}

func (e *env) upload(token, method, path string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "photo.bin")
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes()}
}

func jpegBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 255), uint8(y % 255), 120, 255})
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return b.Bytes()
}

func pngBytes(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

// onboard gives a user a profile, preferences and a photo.
func (e *env) onboard(u user, name, gender string, interestedIn []string, lat, lng float64) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": name, "birthDate": "1994-05-12", "gender": gender, "bio": "Bonjour",
		"city": "Paris", "latitude": lat, "longitude": lng, "interests": []string{"travel", "music"},
	}), 200)
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{
		"interestedIn": interestedIn, "minAge": 21, "maxAge": 50, "maxDistanceKm": 100,
	}), 200)
	e.expect(e.upload(u.Token, "POST", "/me/photos", jpegBytes(640, 800)), 201)
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "bad", "password": "Password123"}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "a@test.local", "password": "short"}), 400)

	u := e.register("auth")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": strings.ToUpper(u.Email), "password": "Password123"}), 409)

	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	var me struct {
		Email string `json:"email"`
	}
	e.expect(e.do("GET", "/me", u.Token, nil), 200).json(t, &me)
	if me.Email != u.Email {
		t.Fatalf("unexpected email %q", me.Email)
	}

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "wrong-password"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": "Password123"}), 200)

	// Refresh rotates the token: the old one cannot be used again.
	var tokens struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), 200).json(t, &tokens)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), 401)

	// Logout revokes the session immediately, even for a still-valid access token.
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": tokens.RefreshToken}), 204)
	e.expect(e.do("GET", "/me", tokens.AccessToken, nil), 401)
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t)
	u := e.register("recover")

	// Unknown addresses get the same answer and no email.
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "nobody@test.local"}), 202)
	if e.mail.last != "" {
		t.Fatal("no mail expected for unknown address")
	}
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202)
	code := ""
	for _, f := range strings.Fields(e.mail.last) {
		f = strings.TrimRight(f, ".")
		if len(f) == 6 && strings.Trim(f, "0123456789") == "" {
			code = f
		}
	}
	if code == "" {
		t.Fatalf("no code in mail: %q", e.mail.last)
	}

	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": "000000", "newPassword": "NewPassword456"}), 400)
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}), 204)
	// Code is single use, old password and old sessions are dead.
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another123456"}), 400)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword456"}), 200)
	e.expect(e.do("GET", "/me", u.Token, nil), 401)
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("prof")
	base := func() map[string]any {
		return map[string]any{"firstName": "Léa", "birthDate": "1995-01-01", "gender": "woman", "bio": "",
			"city": "Lyon", "latitude": 45.76, "longitude": 4.83, "interests": []string{}}
	}
	bad := func(mutate func(m map[string]any)) {
		m := base()
		mutate(m)
		e.expect(e.do("PUT", "/me/profile", u.Token, m), 400)
	}
	bad(func(m map[string]any) { m["birthDate"] = time.Now().AddDate(-17, 0, 0).Format("2006-01-02") }) // minor
	bad(func(m map[string]any) { m["birthDate"] = "not-a-date" })
	bad(func(m map[string]any) { m["gender"] = "robot" })
	bad(func(m map[string]any) { m["firstName"] = "<script>" })
	bad(func(m map[string]any) { m["latitude"] = 123.0 })
	bad(func(m map[string]any) { m["interests"] = []string{"does-not-exist"} })
	bad(func(m map[string]any) { m["bio"] = strings.Repeat("a", 501) })

	var mine struct {
		Profile *struct {
			FirstName string `json:"firstName"`
			Age       int    `json:"age"`
		} `json:"profile"`
		OnboardingComplete bool `json:"onboardingComplete"`
	}
	e.expect(e.do("PUT", "/me/profile", u.Token, base()), 200).json(t, &mine)
	if mine.Profile == nil || mine.Profile.FirstName != "Léa" || mine.Profile.Age < 18 || mine.OnboardingComplete {
		t.Fatalf("unexpected profile: %+v", mine)
	}
	// Coordinates are never echoed back.
	if bytes.Contains(e.do("GET", "/me/profile", u.Token, nil).Body, []byte("latitude")) {
		t.Fatal("profile response must not contain coordinates")
	}
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 10}), 400)
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 40, "maxDistanceKm": 10}), 400)
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 10}), 200)
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	a, b := e.register("photoa"), e.register("photob")
	e.onboard(b, "Bob", "man", []string{"woman"}, 48.85, 2.35)

	// Non-images and corrupt files are refused, whatever the declared name.
	e.expect(e.upload(a.Token, "POST", "/me/photos", []byte("<?php echo 1; ?>")), 415)
	e.expect(e.upload(a.Token, "POST", "/me/photos", append([]byte("\xff\xd8\xff"), []byte("garbage")...)), 400)
	e.expect(e.upload(a.Token, "POST", "/me/photos", jpegBytes(50, 50)), 400) // too small
	e.expect(e.upload(a.Token, "POST", "/me/photos", make([]byte, 9<<20)), 413)

	var p1, p2 struct {
		ID       string `json:"id"`
		Width    int    `json:"width"`
		Position int    `json:"position"`
	}
	e.expect(e.upload(a.Token, "POST", "/me/photos", jpegBytes(3000, 2000)), 201).json(t, &p1)
	if p1.Width != 1280 || p1.Position != 0 {
		t.Fatalf("expected downscaled main photo, got %+v", p1)
	}
	e.expect(e.upload(a.Token, "POST", "/me/photos", pngBytes(400, 400)), 201).json(t, &p2)
	if p2.Position != 1 {
		t.Fatalf("expected position 1, got %d", p2.Position)
	}

	// The stored file is a re-encoded JPEG, served only to authorised viewers.
	served := e.expect(e.do("GET", "/photos/"+p1.ID, a.Token, nil), 200)
	if _, err := jpeg.Decode(bytes.NewReader(served.Body)); err != nil {
		t.Fatalf("served photo is not a JPEG: %v", err)
	}
	e.expect(e.do("GET", "/photos/"+p1.ID, "", nil), 401)
	// b has no relation to a (a has no profile, so not discoverable): forbidden.
	e.expect(e.do("GET", "/photos/"+p1.ID, b.Token, nil), 404)
	e.expect(e.do("GET", "/photos/not-a-uuid", a.Token, nil), 404)

	// Reorder: p2 becomes the main photo.
	e.expect(e.do("PUT", "/me/photos/order", a.Token, map[string]any{"ids": []string{p2.ID, p1.ID}}), 200)
	e.expect(e.do("PUT", "/me/photos/order", a.Token, map[string]any{"ids": []string{p2.ID}}), 400)
	e.expect(e.do("PUT", "/me/photos/order", b.Token, map[string]any{"ids": []string{p2.ID, p1.ID}}), 400)
	var list struct {
		Photos []struct {
			ID string `json:"id"`
		} `json:"photos"`
	}
	e.expect(e.do("GET", "/me/photos", a.Token, nil), 200).json(t, &list)
	if len(list.Photos) != 2 || list.Photos[0].ID != p2.ID {
		t.Fatalf("unexpected order: %+v", list)
	}

	// Only the owner can delete or replace.
	e.expect(e.do("DELETE", "/me/photos/"+p2.ID, b.Token, nil), 404)
	e.expect(e.upload(b.Token, "PUT", "/me/photos/"+p2.ID, jpegBytes(300, 300)), 404)
	e.expect(e.upload(a.Token, "PUT", "/me/photos/"+p2.ID, jpegBytes(300, 300)), 200)
	e.expect(e.do("DELETE", "/me/photos/"+p2.ID, a.Token, nil), 204)
	e.expect(e.do("GET", "/me/photos", a.Token, nil), 200).json(t, &list)
	if len(list.Photos) != 1 || list.Photos[0].ID != p1.ID {
		t.Fatalf("expected one remaining photo: %+v", list)
	}

	// Six photos maximum.
	for i := 0; i < 5; i++ {
		e.expect(e.upload(a.Token, "POST", "/me/photos", jpegBytes(300, 300)), 201)
	}
	e.expect(e.upload(a.Token, "POST", "/me/photos", jpegBytes(300, 300)), 409)
}

func TestDiscoverySwipeMatchChatFlow(t *testing.T) {
	e := newEnv(t)
	alice, bob, carol, dave := e.register("alice"), e.register("bob"), e.register("carol"), e.register("dave")
	e.onboard(alice, "Alice", "woman", []string{"man"}, 48.85, 2.35)
	e.onboard(bob, "Bob", "man", []string{"woman"}, 48.86, 2.34)
	e.onboard(carol, "Carol", "woman", []string{"man"}, 48.87, 2.30) // not Alice's type
	// Dave lives far away (Marseille): outside the 100 km radius.
	e.onboard(dave, "Dave", "man", []string{"woman"}, 43.30, 5.37)

	type card struct {
		ID         string `json:"id"`
		FirstName  string `json:"firstName"`
		DistanceKm *int   `json:"distanceKm"`
		Photos     []struct {
			URL string `json:"url"`
		} `json:"photos"`
		Interests []struct {
			Slug string `json:"slug"`
		} `json:"interests"`
	}
	var feed struct {
		Profiles []card `json:"profiles"`
	}
	e.expect(e.do("GET", "/discover", alice.Token, nil), 200).json(t, &feed)
	if len(feed.Profiles) != 1 || feed.Profiles[0].ID != bob.ID {
		t.Fatalf("alice should only see bob, got %+v", feed.Profiles)
	}
	c := feed.Profiles[0]
	if c.DistanceKm == nil || *c.DistanceKm%5 != 0 || len(c.Photos) != 1 || len(c.Interests) != 2 {
		t.Fatalf("unexpected card: %+v", c)
	}
	raw := e.do("GET", "/discover", alice.Token, nil).Body
	for _, secret := range []string{"latitude", "longitude", "lat\"", "birthDate", "email"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("discovery leaks %q: %s", secret, raw)
		}
	}
	// Preferences are mutual: carol (woman seeking men) does not see alice, only bob.
	e.expect(e.do("GET", "/discover", carol.Token, nil), 200).json(t, &feed)
	if len(feed.Profiles) != 1 || feed.Profiles[0].ID != bob.ID {
		t.Fatalf("carol should only see bob, got %+v", feed.Profiles)
	}

	// Alice likes Bob: no match yet. Repeating is idempotent.
	var sw struct {
		Matched        bool   `json:"matched"`
		Duplicate      bool   `json:"duplicate"`
		ConversationID string `json:"conversationId"`
	}
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), 200).json(t, &sw)
	if sw.Matched || sw.Duplicate {
		t.Fatalf("unexpected: %+v", sw)
	}
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), 200).json(t, &sw)
	if sw.Matched || !sw.Duplicate {
		t.Fatalf("duplicate like must be flagged: %+v", sw)
	}
	// Already-processed profiles never come back.
	e.expect(e.do("GET", "/discover", alice.Token, nil), 200).json(t, &feed)
	if len(feed.Profiles) != 0 {
		t.Fatalf("processed profile reappeared: %+v", feed.Profiles)
	}
	// No conversation exists before the match.
	e.expect(e.do("POST", "/conversations/00000000-0000-0000-0000-000000000000/messages", alice.Token, map[string]string{"body": "hi"}), 404)

	// Bob sees Alice, likes back: match for both.
	e.expect(e.do("GET", "/discover", bob.Token, nil), 200).json(t, &feed)
	// Alice (who already liked Bob) is ranked first, ahead of Carol.
	if len(feed.Profiles) != 2 || feed.Profiles[0].ID != alice.ID || feed.Profiles[1].ID != carol.ID {
		t.Fatalf("bob should see alice then carol: %+v", feed.Profiles)
	}
	e.expect(e.do("POST", "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), 200).json(t, &sw)
	if !sw.Matched || sw.Duplicate || sw.ConversationID == "" {
		t.Fatalf("expected a match: %+v", sw)
	}
	convID := sw.ConversationID
	e.expect(e.do("POST", "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), 200).json(t, &sw)
	if !sw.Matched || !sw.Duplicate {
		t.Fatalf("repeat must not create a second match: %+v", sw)
	}
	var matchCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches`).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match, got %d", matchCount)
	}

	// Invalid swipes.
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": alice.ID, "action": "like"}), 400)
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "superlike"}), 400)
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": "nope", "action": "like"}), 404)

	// Both got a match notification.
	var notifs struct {
		Notifications []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do("GET", "/notifications", alice.Token, nil), 200).json(t, &notifs)
	if len(notifs.Notifications) != 1 || notifs.Notifications[0].Type != "match" || notifs.UnreadCount != 1 {
		t.Fatalf("unexpected notifications for alice: %+v", notifs)
	}
	e.expect(e.do("POST", "/notifications/"+notifs.Notifications[0].ID+"/read", bob.Token, nil), 404) // not bob's
	e.expect(e.do("POST", "/notifications/"+notifs.Notifications[0].ID+"/read", alice.Token, nil), 204)
	e.expect(e.do("GET", "/notifications", alice.Token, nil), 200).json(t, &notifs)
	if notifs.UnreadCount != 0 {
		t.Fatalf("expected no unread, got %d", notifs.UnreadCount)
	}
	e.expect(e.do("POST", "/notifications/read-all", bob.Token, nil), 204)

	// Chat: permissions first.
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", carol.Token, nil), 404)
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", carol.Token, map[string]string{"body": "intruder"}), 404)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", carol.Token, nil), 404)
	e.expect(e.do("DELETE", "/conversations/"+convID, carol.Token, nil), 404)
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]string{"body": "   "}), 400)

	var msg struct {
		ID, SenderID, Body string
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]string{"body": "  Salut Bob !  "}), 201).json(t, &msg)
	if msg.Body != "Salut Bob !" || msg.SenderID != alice.ID {
		t.Fatalf("unexpected message: %+v", msg)
	}

	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
			Peer struct {
				ID, FirstName string
			} `json:"peer"`
		} `json:"conversations"`
		UnreadTotal int `json:"unreadTotal"`
	}
	e.expect(e.do("GET", "/conversations", bob.Token, nil), 200).json(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.UnreadTotal != 1 ||
		convs.Conversations[0].LastMessage == nil || convs.Conversations[0].Peer.FirstName != "Alice" {
		t.Fatalf("unexpected bob list: %+v", convs)
	}
	e.expect(e.do("GET", "/conversations", alice.Token, nil), 200).json(t, &convs)
	if convs.UnreadTotal != 0 {
		t.Fatalf("sender has nothing unread: %+v", convs)
	}
	// Bob got a message notification.
	e.expect(e.do("GET", "/notifications", bob.Token, nil), 200).json(t, &notifs)
	if notifs.UnreadCount != 1 || notifs.Notifications[0].Type != "message" {
		t.Fatalf("expected message notification: %+v", notifs)
	}

	// Read state.
	var msgs struct {
		Messages       []struct{ Body string } `json:"messages"`
		PeerLastReadAt time.Time               `json:"peerLastReadAt"`
	}
	time.Sleep(5 * time.Millisecond)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", bob.Token, nil), 204)
	e.expect(e.do("GET", "/conversations", bob.Token, nil), 200).json(t, &convs)
	if convs.UnreadTotal != 0 {
		t.Fatalf("expected read, got %+v", convs)
	}
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", alice.Token, nil), 200).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.PeerLastReadAt.IsZero() {
		t.Fatalf("unexpected messages: %+v", msgs)
	}

	// Local deletion hides the conversation until a new message arrives.
	e.expect(e.do("DELETE", "/conversations/"+convID, bob.Token, nil), 204)
	e.expect(e.do("GET", "/conversations", bob.Token, nil), 200).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatalf("hidden conversation still listed: %+v", convs)
	}
	e.expect(e.do("GET", "/conversations", alice.Token, nil), 200).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("alice must keep her conversation")
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]string{"body": "Tu es là ?"}), 201)
	e.expect(e.do("GET", "/conversations", bob.Token, nil), 200).json(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 {
		t.Fatalf("conversation should resurface: %+v", convs)
	}

	// Blocking dissolves the match, the chat and mutual visibility.
	e.expect(e.do("POST", "/reports", bob.Token, map[string]any{"userId": alice.ID, "reason": "spam", "details": "test", "block": true}), 202)
	var evidence string
	_ = e.pool.QueryRow(context.Background(), `SELECT evidence::text FROM reports WHERE reporter_id = $1`, bob.ID).Scan(&evidence)
	if !strings.Contains(evidence, "Tu es là ?") {
		t.Fatalf("report evidence missing: %s", evidence)
	}
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", alice.Token, nil), 404)
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), 404)
	e.expect(e.do("GET", "/profiles/"+bob.ID, alice.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+alice.ID, bob.Token, nil), 404)
	var blocks struct {
		Blocks []struct{ UserID string } `json:"blocks"`
	}
	e.expect(e.do("GET", "/blocks", bob.Token, nil), 200).json(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != alice.ID {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
	e.expect(e.do("POST", "/blocks", bob.Token, map[string]string{"userId": bob.ID}), 400)
	e.expect(e.do("POST", "/reports", bob.Token, map[string]any{"userId": alice.ID, "reason": "bogus"}), 400)
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	a, b := e.register("racea"), e.register("raceb")
	e.onboard(a, "Ana", "woman", []string{"man"}, 48.85, 2.35)
	e.onboard(b, "Ben", "man", []string{"woman"}, 48.85, 2.35)

	var wg sync.WaitGroup
	results := make(chan bool, 2)
	for _, pair := range [][2]user{{a, b}, {b, a}} {
		wg.Add(1)
		go func(actor, target user) {
			defer wg.Done()
			var sw struct {
				Matched   bool `json:"matched"`
				Duplicate bool `json:"duplicate"`
			}
			r := e.do("POST", "/swipes", actor.Token, map[string]string{"targetId": target.ID, "action": "like"})
			r.json(t, &sw)
			results <- sw.Matched && !sw.Duplicate
		}(pair[0], pair[1])
	}
	wg.Wait()
	close(results)
	announced := 0
	for r := range results {
		if r {
			announced++
		}
	}
	var matches, convs int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches`).Scan(&matches)
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM conversations`).Scan(&convs)
	if matches != 1 || convs != 1 || announced != 1 {
		t.Fatalf("expected exactly one match announced once, got matches=%d conversations=%d announced=%d", matches, convs, announced)
	}
}

func TestPrivacyUnmatchAndAccountDeletion(t *testing.T) {
	e := newEnv(t)
	a, b := e.register("dela"), e.register("delb")
	e.onboard(a, "Ana", "woman", []string{"man"}, 48.85, 2.35)
	e.onboard(b, "Ben", "man", []string{"woman"}, 48.85, 2.35)

	// Hiding the profile removes it from discovery and from direct viewing.
	var feed struct {
		Profiles []struct{ ID string } `json:"profiles"`
	}
	e.expect(e.do("PUT", "/me/privacy", b.Token, map[string]bool{"discoverable": false, "showDistance": true}), 200)
	e.expect(e.do("GET", "/discover", a.Token, nil), 200).json(t, &feed)
	if len(feed.Profiles) != 0 {
		t.Fatal("hidden profile must not be discoverable")
	}
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 404)
	e.expect(e.do("PUT", "/me/privacy", b.Token, map[string]bool{"discoverable": true, "showDistance": false}), 200)
	e.expect(e.do("GET", "/discover", a.Token, nil), 200).json(t, &feed)
	if len(feed.Profiles) != 1 {
		t.Fatal("profile should be back")
	}
	var prof struct {
		DistanceKm *int `json:"distanceKm"`
	}
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 200).json(t, &prof)
	if prof.DistanceKm != nil {
		t.Fatal("distance must be hidden when showDistance is false")
	}

	// Pass hides a profile; resetting restores it.
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "pass"}), 200)
	e.expect(e.do("GET", "/discover", a.Token, nil), 200).json(t, &feed)
	if len(feed.Profiles) != 0 {
		t.Fatal("passed profile reappeared")
	}
	e.expect(e.do("DELETE", "/discover/passes", a.Token, nil), 200)
	e.expect(e.do("GET", "/discover", a.Token, nil), 200).json(t, &feed)
	if len(feed.Profiles) != 1 {
		t.Fatal("passes were not reset")
	}

	// Match, then unmatch.
	var sw struct {
		MatchID string `json:"matchId"`
	}
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"}), 200)
	e.expect(e.do("POST", "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"}), 200).json(t, &sw)
	e.expect(e.do("DELETE", "/matches/"+sw.MatchID, a.Token, nil), 204)
	e.expect(e.do("DELETE", "/matches/"+sw.MatchID, a.Token, nil), 404)

	// Account deletion needs the password and erases everything.
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong-password"}), 403)
	var photoCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos WHERE user_id = $1`, a.ID).Scan(&photoCount)
	if photoCount != 1 {
		t.Fatalf("expected a photo before deletion, got %d", photoCount)
	}
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "Password123"}), 204)
	e.expect(e.do("GET", "/me", a.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Password123"}), 401)
	for _, table := range []string{"profiles", "preferences", "photos", "user_interests", "swipes", "matches", "sessions"} {
		col := "user_id"
		switch table {
		case "swipes":
			col = "actor_id"
		case "matches":
			col = "user_a"
		}
		var n int
		_ = e.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, table, col), a.ID).Scan(&n)
		if n != 0 {
			t.Fatalf("table %s still holds %d rows for the deleted user", table, n)
		}
	}
	// The other person never sees the deleted user again.
	e.expect(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 404)
}

func TestRealtimeTicketAndWebSocketAuth(t *testing.T) {
	e := newEnv(t)
	u := e.register("ws")
	var tk struct {
		Ticket string `json:"ticket"`
	}
	e.expect(e.do("POST", "/ws/ticket", "", nil), 401)
	e.expect(e.do("POST", "/ws/ticket", u.Token, nil), 200).json(t, &tk)
	if len(tk.Ticket) < 32 {
		t.Fatal("ticket too short")
	}
	// Without a valid ticket the upgrade is refused.
	e.expect(e.do("GET", "/ws?ticket=nope", "", nil), 401)
}

func TestRealtimeMessageDelivery(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.register("rta"), e.register("rtb")
	e.onboard(alice, "Alice", "woman", []string{"man"}, 48.85, 2.35)
	e.onboard(bob, "Bob", "man", []string{"woman"}, 48.85, 2.35)
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), 200)
	var sw struct {
		ConversationID string `json:"conversationId"`
	}

	srv := httptest.NewServer(e.router)
	defer srv.Close()
	var tk struct {
		Ticket string `json:"ticket"`
	}
	e.expect(e.do("POST", "/ws/ticket", alice.Token, nil), 200).json(t, &tk)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket="+tk.Ticket, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Tickets are single use.
	if _, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket="+tk.Ticket, nil); err == nil {
		t.Fatal("ticket must be single use")
	}

	read := func() map[string]any {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var ev map[string]any
		_ = json.Unmarshal(data, &ev)
		return ev
	}

	// Bob likes back: Alice (connected) receives match.new and a notification.
	e.expect(e.do("POST", "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), 200).json(t, &sw)
	seen := map[string]bool{}
	for len(seen) < 2 {
		ev := read()
		seen[ev["type"].(string)] = true
	}
	if !seen["match.new"] || !seen["notification.new"] {
		t.Fatalf("expected match.new and notification.new, got %v", seen)
	}

	e.expect(e.do("POST", "/conversations/"+sw.ConversationID+"/messages", bob.Token, map[string]string{"body": "Hello Alice"}), 201)
	got := false
	for i := 0; i < 3 && !got; i++ {
		ev := read()
		if ev["type"] == "message.new" {
			data := ev["data"].(map[string]any)
			got = data["body"] == "Hello Alice" && data["senderId"] == bob.ID
		}
	}
	if !got {
		t.Fatal("message.new not delivered")
	}
}
