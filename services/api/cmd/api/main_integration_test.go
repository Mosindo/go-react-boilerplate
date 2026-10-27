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
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/testutil"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

type env struct {
	t       *testing.T
	pool    *pgxpool.Pool
	server  *httptest.Server
	uploads string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testutil.DB(t)
	dir := t.TempDir()
	store, err := storage.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{
		dbPool:    pool,
		jwtSecret: []byte("integration-secret-integration-secret"),
		store:     store,
		mailer:    mailer.DiscardMailer{},
	}
	srv := httptest.NewServer(setupRouter(a))
	t.Cleanup(srv.Close)
	return &env{t: t, pool: pool, server: srv, uploads: dir}
}

type user struct {
	e       *env
	ID      string
	Email   string
	Token   string
	Refresh string
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) json(v any) {
	if err := json.Unmarshal(r.Body, v); err != nil {
		panic(fmt.Sprintf("decode %s: %v", r.Body, err))
	}
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
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
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Body: data}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected HTTP %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func (e *env) register(prefix string) *user {
	e.t.Helper()
	email := fmt.Sprintf("%s_%d@e2e.test", prefix, time.Now().UnixNano())
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"}), 201)
	var out struct {
		AccessToken  string              `json:"accessToken"`
		RefreshToken string              `json:"refreshToken"`
		User         struct{ ID string } `json:"user"`
	}
	r.json(&out)
	u := &user{e: e, ID: out.User.ID, Email: email, Token: out.AccessToken, Refresh: out.RefreshToken}
	e.t.Cleanup(func() { e.cleanupUser(u.ID) })
	return u
}

func (e *env) cleanupUser(id string) {
	_, _ = e.pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
}

func (u *user) do(method, path string, body any) resp { return u.e.do(method, path, u.Token, body) }

func pngBytes() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 400, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func (u *user) upload(method, path string, data []byte, filename string) resp {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("photo", filename)
	_, _ = fw.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(method, u.e.server.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+u.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		u.e.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Body: b}
}

// onboard creates profile + photo for a user located near Paris.
func (u *user) onboard(name, gender string, interestedIn []string, lat, lng float64) {
	u.e.t.Helper()
	body := map[string]any{
		"firstName": name, "birthDate": "1993-04-12", "gender": gender, "bio": "Hello, I like hiking.",
		"city": "Paris", "latitude": lat, "longitude": lng, "interests": []string{"hiking", "music"},
	}
	u.e.expect(u.do("PUT", "/profile/me", body), 200)
	u.e.expect(u.do("PUT", "/profile/preferences", map[string]any{
		"interestedIn": interestedIn, "ageMin": 18, "ageMax": 60, "maxDistanceKm": 100,
	}), 200)
	u.e.expect(u.upload("POST", "/photos", pngBytes(), "a.png"), 201)
}

func TestHealthAndAuthGuards(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
	for _, p := range []string{"/me", "/profile/me", "/discover", "/matches", "/conversations", "/notifications", "/blocks"} {
		e.expect(e.do("GET", p, "", nil), 401)
		e.expect(e.do("GET", p, "garbage", nil), 401)
	}
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "bad", "password": "Password123"}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "a@e2e.test", "password": "short"}), 400)
}

func TestDatingFlowEndToEnd(t *testing.T) {
	e := newEnv(t)
	alice := e.register("alice")
	bob := e.register("bob")
	carol := e.register("carol")
	mallory := e.register("mallory")

	// Profile is required before discovery or swipes.
	e.expect(alice.do("GET", "/profile/me", nil), 404)
	e.expect(alice.do("POST", "/photos", nil), 400)

	// Validation: underage, bad gender, bad interests.
	bad := map[string]any{"firstName": "Al", "birthDate": time.Now().AddDate(-17, 0, 0).Format("2006-01-02"), "gender": "woman", "interests": []string{}}
	e.expect(alice.do("PUT", "/profile/me", bad), 400)
	bad["birthDate"], bad["gender"] = "1990-01-01", "alien"
	e.expect(alice.do("PUT", "/profile/me", bad), 400)
	bad["gender"], bad["interests"] = "woman", []string{"does_not_exist"}
	e.expect(alice.do("PUT", "/profile/me", bad), 400)

	alice.onboard("Alice", "woman", []string{"man"}, 48.8566, 2.3522)
	bob.onboard("Bob", "man", []string{"woman"}, 48.9, 2.4)
	carol.onboard("Carol", "woman", []string{"man"}, 48.85, 2.35) // not interested in by Alice's preference
	mallory.onboard("Mallory", "man", []string{"woman"}, 48.86, 2.34)

	// Photos: bad uploads are rejected, good ones are re-encoded and access-controlled.
	e.expect(alice.upload("POST", "/photos", []byte("<?php system($_GET['c']); ?>"), "evil.png"), 400)
	var me struct {
		Photos      []struct{ ID, URL string } `json:"photos"`
		Complete    bool                       `json:"complete"`
		Preferences struct {
			InterestedIn []string `json:"interestedIn"`
		} `json:"preferences"`
	}
	e.expect(alice.do("GET", "/profile/me", nil), 200).json(&me)
	if len(me.Photos) != 1 || !me.Complete {
		t.Fatalf("expected 1 photo and complete profile: %+v", me)
	}
	photo := e.expect(alice.do("GET", me.Photos[0].URL, nil), 200)
	if !bytes.HasPrefix(photo.Body, []byte{0xFF, 0xD8}) {
		t.Fatal("served photo must be a JPEG")
	}
	e.expect(e.do("GET", me.Photos[0].URL, "", nil), 401)

	// Discovery: Alice (interested in men) sees Bob and Mallory, never herself or Carol.
	var feed struct {
		Profiles []struct {
			UserID     string                 `json:"userId"`
			FirstName  string                 `json:"firstName"`
			DistanceKM *int                   `json:"distanceKm"`
			Photos     []struct{ URL string } `json:"photos"`
			Interests  []string               `json:"interests"`
		} `json:"profiles"`
	}
	e.expect(alice.do("GET", "/discover?limit=10", nil), 200).json(&feed)
	ids := map[string]bool{}
	for _, p := range feed.Profiles {
		ids[p.UserID] = true
		if p.DistanceKM == nil || *p.DistanceKM%5 != 0 {
			t.Fatalf("distance must be a rounded bucket, got %v", p.DistanceKM)
		}
		if len(p.Photos) == 0 || len(p.Interests) == 0 {
			t.Fatal("cards must include photos and interests")
		}
	}
	if !ids[bob.ID] || !ids[mallory.ID] || ids[alice.ID] || ids[carol.ID] {
		t.Fatalf("unexpected feed: %v", ids)
	}
	if strings.Contains(string(alice.do("GET", "/discover", nil).Body), "latitude") || strings.Contains(string(alice.do("GET", "/discover", nil).Body), "birthDate") {
		t.Fatal("public profiles must not leak coordinates or birth date")
	}

	// Swipe validation and idempotence.
	e.expect(alice.do("POST", "/swipes", map[string]string{"targetId": alice.ID, "action": "like"}), 400)
	e.expect(alice.do("POST", "/swipes", map[string]string{"targetId": bob.ID, "action": "super"}), 400)
	e.expect(alice.do("POST", "/swipes", map[string]string{"targetId": "not-a-uuid", "action": "like"}), 400)
	e.expect(alice.do("POST", "/swipes", map[string]string{"targetId": "00000000-0000-0000-0000-000000000000", "action": "like"}), 404)

	var res struct {
		Matched        bool   `json:"matched"`
		ConversationID string `json:"conversationId"`
		MatchID        string `json:"matchId"`
	}
	e.expect(alice.do("POST", "/swipes", map[string]string{"targetId": bob.ID, "action": "like"}), 200).json(&res)
	if res.Matched {
		t.Fatal("a one-sided like must not match")
	}
	e.expect(alice.do("POST", "/swipes", map[string]string{"targetId": bob.ID, "action": "like"}), 409) // duplicate
	e.expect(alice.do("POST", "/swipes", map[string]string{"targetId": mallory.ID, "action": "pass"}), 200)

	// Handled profiles never come back.
	feed.Profiles = nil
	e.expect(alice.do("GET", "/discover", nil), 200).json(&feed)
	if len(feed.Profiles) != 0 {
		t.Fatalf("handled profiles must not reappear: %+v", feed.Profiles)
	}

	// Realtime: Bob connects before the match is made.
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
	bobWS, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bobWS.Close()
	_ = bobWS.WriteJSON(map[string]string{"type": "auth", "token": bob.Token})
	readEvent := func() map[string]any {
		_ = bobWS.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev map[string]any
		if err := bobWS.ReadJSON(&ev); err != nil {
			t.Fatalf("websocket read: %v", err)
		}
		return ev
	}
	if ev := readEvent(); ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v", ev)
	}

	// Bob likes Alice back: mutual -> match, notified in realtime.
	e.expect(bob.do("POST", "/swipes", map[string]string{"targetId": alice.ID, "action": "like"}), 200).json(&res)
	if !res.Matched || res.ConversationID == "" || res.MatchID == "" {
		t.Fatalf("expected a match: %+v", res)
	}
	if ev := readEvent(); ev["type"] != "match" {
		t.Fatalf("expected match event, got %v", ev)
	}
	var count int
	pool := testutil.DB(t)
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE (user_a = $1 AND user_b = $2) OR (user_a = $2 AND user_b = $1)`, alice.ID, bob.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("exactly one match row expected, got %d", count)
	}

	var matches struct {
		Matches []struct {
			ConversationID string                             `json:"conversationId"`
			User           struct{ UserID, FirstName string } `json:"user"`
		} `json:"matches"`
	}
	e.expect(alice.do("GET", "/matches", nil), 200).json(&matches)
	if len(matches.Matches) != 1 || matches.Matches[0].User.UserID != bob.ID || matches.Matches[0].ConversationID != res.ConversationID {
		t.Fatalf("unexpected matches: %+v", matches)
	}

	// Chat: only participants can read or write.
	conv := res.ConversationID
	e.expect(carol.do("GET", "/conversations/"+conv+"/messages", nil), 404)
	e.expect(carol.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "intruder"}), 404)
	e.expect(carol.do("POST", "/conversations/"+conv+"/read", nil), 404)
	e.expect(carol.do("DELETE", "/conversations/"+conv, nil), 404)
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "   "}), 400)
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": strings.Repeat("x", 2001)}), 400)
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "Hi Bob!"}), 201)
	if ev := readEvent(); ev["type"] != "message" {
		t.Fatalf("expected message event, got %v", ev)
	}

	var convs struct {
		Conversations []struct {
			ID          string                `json:"id"`
			UnreadCount int                   `json:"unreadCount"`
			LastMessage struct{ Body string } `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.expect(bob.do("GET", "/conversations", nil), 200).json(&convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].LastMessage.Body != "Hi Bob!" {
		t.Fatalf("unexpected conversations for bob: %+v", convs)
	}
	e.expect(bob.do("POST", "/conversations/"+conv+"/read", nil), 200)
	convs.Conversations = nil
	e.expect(bob.do("GET", "/conversations", nil), 200).json(&convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread count must drop to 0 after read")
	}
	e.expect(bob.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "Hey Alice"}), 201)

	var msgs struct {
		Messages []struct {
			ID   int64  `json:"id"`
			Body string `json:"body"`
		} `json:"messages"`
		OtherLastReadID int64 `json:"otherLastReadId"`
	}
	e.expect(alice.do("GET", "/conversations/"+conv+"/messages", nil), 200).json(&msgs)
	if len(msgs.Messages) != 2 || msgs.Messages[0].Body != "Hey Alice" {
		t.Fatalf("messages must be newest first: %+v", msgs)
	}
	if msgs.OtherLastReadID == 0 {
		t.Fatal("read receipt expected")
	}

	// Notifications
	var notif struct {
		Notifications []struct{ ID, Type string } `json:"notifications"`
		UnreadCount   int                         `json:"unreadCount"`
	}
	e.expect(alice.do("GET", "/notifications", nil), 200).json(&notif)
	if notif.UnreadCount < 2 {
		t.Fatalf("alice should have match + message notifications: %+v", notif)
	}
	e.expect(alice.do("POST", "/notifications/"+notif.Notifications[0].ID+"/read", nil), 204)
	e.expect(carol.do("POST", "/notifications/"+notif.Notifications[0].ID+"/read", nil), 404)
	e.expect(alice.do("POST", "/notifications/read", nil), 204)
	e.expect(alice.do("GET", "/notifications", nil), 200).json(&notif)
	if notif.UnreadCount != 0 {
		t.Fatal("all notifications must be read")
	}

	// Local deletion hides history for one side only.
	e.expect(bob.do("DELETE", "/conversations/"+conv, nil), 204)
	convs.Conversations = nil
	e.expect(bob.do("GET", "/conversations", nil), 200).json(&convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("cleared conversation must disappear for bob")
	}
	convs.Conversations = nil
	e.expect(alice.do("GET", "/conversations", nil), 200).json(&convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("alice must keep her history")
	}

	// Privacy: hiding the profile removes it from discovery.
	e.expect(mallory.do("PUT", "/profile/privacy", map[string]any{"discoverable": false}), 200)
	e.expect(carol.do("PUT", "/profile/preferences", map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 60, "maxDistanceKm": 100}), 200)
	feed.Profiles = nil
	e.expect(carol.do("GET", "/discover", nil), 200).json(&feed)
	for _, p := range feed.Profiles {
		if p.UserID == mallory.ID {
			t.Fatal("non discoverable profile leaked into the feed")
		}
	}

	// Report validation + block: blocking removes the match and the conversation.
	e.expect(alice.do("POST", "/reports", map[string]any{"userId": bob.ID, "reason": "nonsense"}), 400)
	e.expect(alice.do("POST", "/reports", map[string]any{"userId": alice.ID, "reason": "spam"}), 400)
	e.expect(alice.do("POST", "/reports", map[string]any{"userId": bob.ID, "reason": "harassment", "details": "rude", "block": true}), 201)
	e.expect(alice.do("GET", "/conversations/"+conv+"/messages", nil), 404)
	e.expect(bob.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "still there?"}), 404)
	matches.Matches = nil
	e.expect(alice.do("GET", "/matches", nil), 200).json(&matches)
	if len(matches.Matches) != 0 {
		t.Fatal("block must remove the match")
	}
	e.expect(bob.do("GET", me.Photos[0].URL, nil), 404)                                                 // blocked users cannot fetch the blocker's photos
	e.expect(bob.do("POST", "/swipes", map[string]string{"targetId": alice.ID, "action": "like"}), 404) // blocked profiles are unavailable
	e.expect(alice.do("DELETE", "/blocks/"+bob.ID, nil), 204)
	var blocked struct {
		Blocks []struct{ UserID string } `json:"blocks"`
	}
	e.expect(alice.do("GET", "/blocks", nil), 200).json(&blocked)
	if len(blocked.Blocks) != 0 {
		t.Fatal("unblock must clear the list")
	}

	// Account deletion: password required, everything is removed, token stops working.
	e.expect(bob.do("DELETE", "/me", map[string]string{"password": "wrong-password"}), 403)
	e.expect(bob.do("DELETE", "/me", map[string]string{"password": "Password123"}), 204)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": bob.Email, "password": "Password123"}), 401)
	_ = pool.QueryRow(context.Background(), `SELECT (SELECT COUNT(*) FROM profiles WHERE user_id = $1) + (SELECT COUNT(*) FROM photos WHERE user_id = $1) + (SELECT COUNT(*) FROM messages WHERE sender_id = $1) + (SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1)`, bob.ID).Scan(&count)
	if count != 0 {
		t.Fatalf("account deletion must cascade, %d rows remain", count)
	}
	entries, _ := os.ReadDir(e.uploads)
	if len(entries) != 3 { // alice, carol, mallory keep their photo; bob's file is gone
		t.Fatalf("expected 3 stored files after bob's deletion, got %d", len(entries))
	}
}

func TestRejectsOversizedAndExtraPhotos(t *testing.T) {
	e := newEnv(t)
	u := e.register("photos")
	u.onboard("Pat", "non_binary", []string{"man", "woman", "non_binary"}, 45.0, 4.0)
	for i := 0; i < 5; i++ {
		e.expect(u.upload("POST", "/photos", pngBytes(), "p.png"), 201)
	}
	e.expect(u.upload("POST", "/photos", pngBytes(), "p.png"), 409) // 7th photo
	e.expect(u.upload("POST", "/photos", make([]byte, 9<<20), "big.png"), 413)

	var p struct {
		Photos []struct{ ID string } `json:"photos"`
	}
	e.expect(u.do("GET", "/profile/me", nil), 200).json(&p)
	if len(p.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(p.Photos))
	}
	// Reorder: make the last photo the main one.
	order := []string{p.Photos[5].ID, p.Photos[0].ID, p.Photos[1].ID, p.Photos[2].ID, p.Photos[3].ID, p.Photos[4].ID}
	e.expect(u.do("PUT", "/photos/order", map[string]any{"ids": order}), 200)
	e.expect(u.do("PUT", "/photos/order", map[string]any{"ids": order[:3]}), 400)
	e.expect(u.do("PUT", "/photos/order", map[string]any{"ids": append(order[:5:5], order[0])}), 400)
	e.expect(u.upload("PUT", "/photos/"+order[0], pngBytes(), "r.png"), 200)
	e.expect(u.do("DELETE", "/photos/"+order[0], nil), 204)
	e.expect(u.do("DELETE", "/photos/"+order[0], nil), 404)
	e.expect(u.do("GET", "/profile/me", nil), 200).json(&p)
	if len(p.Photos) != 5 {
		t.Fatalf("expected 5 photos after deletion, got %d", len(p.Photos))
	}
}

// Two people liking each other at the very same instant must still produce exactly one match.
func TestSimultaneousLikesProduceOneMatch(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 5; i++ {
		a := e.register("racea")
		b := e.register("raceb")
		a.onboard("Ana", "woman", []string{"man"}, 48.85, 2.35)
		b.onboard("Ben", "man", []string{"woman"}, 48.85, 2.35)

		results := make(chan bool, 2)
		start := make(chan struct{})
		for _, pair := range [][2]*user{{a, b}, {b, a}} {
			go func(from, to *user) {
				<-start
				r := from.do("POST", "/swipes", map[string]string{"targetId": to.ID, "action": "like"})
				var out struct {
					Matched bool `json:"matched"`
				}
				r.json(&out)
				results <- out.Matched
			}(pair[0], pair[1])
		}
		close(start)
		first, second := <-results, <-results
		if !first && !second {
			t.Fatalf("round %d: neither request reported the match", i)
		}
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: expected exactly 1 match, got %d", i, n)
		}
	}
}
