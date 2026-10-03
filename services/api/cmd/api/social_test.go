package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type pair struct {
	a, b    testUser
	matchID string
}

// matchedPair creates two compatible users and matches them.
func (e *env) matchedPair() pair {
	e.t.Helper()
	lat, lng := scenarioOrigin()
	a := e.completeUser("pair-a", profileSpec{Name: "Ada", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	b := e.completeUser("pair-b", profileSpec{Name: "Bo", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	e.expect(e.do(http.MethodPost, "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"}), http.StatusOK)
	var out struct {
		MatchID string `json:"matchId"`
	}
	e.expect(e.do(http.MethodPost, "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"}), http.StatusOK).json(e.t, &out)
	if out.MatchID == "" {
		e.t.Fatal("expected a match")
	}
	return pair{a: a, b: b, matchID: out.MatchID}
}

func TestPhotoLifecycleAndValidation(t *testing.T) {
	e := newEnv(t, false)
	lat, lng := scenarioOrigin()
	u := e.register("photos")

	// A profile is needed before photos.
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "a.jpg", jpegBytes(40, 40)), http.StatusConflict)
	e.makeProfile(u, profileSpec{Name: "Pho", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})

	// Content sniffing, not file names or headers, decides.
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "evil.jpg", []byte("<?php echo 1; ?>")), http.StatusUnsupportedMediaType)
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "evil.png", []byte("GIF89a....")), http.StatusUnsupportedMediaType)
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "trunc.jpg", jpegBytes(80, 80)[:200]), http.StatusBadRequest)
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "big.jpg", append(jpegBytes(40, 40), make([]byte, 9<<20)...)), http.StatusRequestEntityTooLarge)
	e.expect(e.do(http.MethodPost, "/me/photos", u.Token, []byte("{}")), http.StatusBadRequest)

	// A large JPEG is downscaled and re-encoded.
	var big struct {
		ID string `json:"id"`
	}
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "big.jpg", jpegBytes(2600, 1800)), http.StatusCreated).json(t, &big)
	raw := e.expect(e.do(http.MethodGet, "/photos/"+big.ID, u.Token, nil), http.StatusOK)
	if raw.Header.Get("Content-Type") != "image/jpeg" || raw.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unsafe headers: %v", raw.Header)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw.Body))
	if err != nil || format != "jpeg" || cfg.Width != 1280 || cfg.Height != 886 {
		t.Fatalf("expected a 1280x886 jpeg, got %v %s %v", cfg, format, err)
	}

	// PNG (with alpha) is accepted and converted.
	var png struct {
		ID string `json:"id"`
	}
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "t.png", pngBytes(50, 20)), http.StatusCreated).json(t, &png)
	raw = e.do(http.MethodGet, "/photos/"+png.ID, u.Token, nil)
	if _, format, err := image.DecodeConfig(bytes.NewReader(raw.Body)); err != nil || format != "jpeg" {
		t.Fatalf("png must be stored as jpeg: %s %v", format, err)
	}

	ids := []string{big.ID, png.ID}
	for i := 0; i < 4; i++ {
		var p struct {
			ID string `json:"id"`
		}
		e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "x.jpg", jpegBytes(30, 30)), http.StatusCreated).json(t, &p)
		ids = append(ids, p.ID)
	}
	e.expect(e.upload(http.MethodPost, "/me/photos", u.Token, "x.jpg", jpegBytes(30, 30)), http.StatusConflict)

	type photoList struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	check := func(want []string) {
		t.Helper()
		var l photoList
		e.expect(e.do(http.MethodGet, "/me/photos", u.Token, nil), http.StatusOK).json(t, &l)
		if len(l.Photos) != len(want) {
			t.Fatalf("expected %d photos, got %d", len(want), len(l.Photos))
		}
		for i, p := range l.Photos {
			if p.ID != want[i] || p.Position != i {
				t.Fatalf("position %d: expected %s got %s/%d", i, want[i], p.ID, p.Position)
			}
		}
	}
	check(ids)

	// Reorder: the new first photo becomes the main one. Partial or foreign lists are refused.
	reordered := []string{ids[3], ids[0], ids[1], ids[2], ids[5], ids[4]}
	e.expect(e.do(http.MethodPut, "/me/photos/order", u.Token, map[string]any{"ids": reordered}), http.StatusOK)
	check(reordered)
	e.expect(e.do(http.MethodPut, "/me/photos/order", u.Token, map[string]any{"ids": reordered[:3]}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPut, "/me/photos/order", u.Token, map[string]any{"ids": append(append([]string{}, reordered[:5]...), reordered[0])}), http.StatusBadRequest)

	// Another user can neither delete, replace nor reorder it.
	other := e.completeUser("photos-other", profileSpec{Name: "Oth", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	e.expect(e.do(http.MethodDelete, "/me/photos/"+reordered[0], other.Token, nil), http.StatusNotFound)
	e.expect(e.upload(http.MethodPut, "/me/photos/"+reordered[0], other.Token, "x.jpg", jpegBytes(30, 30)), http.StatusNotFound)

	// Replacement keeps the slot and gets a new id; the old one disappears.
	var repl struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
	}
	e.expect(e.upload(http.MethodPut, "/me/photos/"+reordered[1], u.Token, "new.jpg", jpegBytes(30, 30)), http.StatusCreated).json(t, &repl)
	if repl.Position != 1 || repl.ID == reordered[1] {
		t.Fatalf("unexpected replacement: %+v", repl)
	}
	e.expect(e.do(http.MethodGet, "/photos/"+reordered[1], u.Token, nil), http.StatusNotFound)
	reordered[1] = repl.ID

	// Deleting closes the gap.
	e.expect(e.do(http.MethodDelete, "/me/photos/"+reordered[0], u.Token, nil), http.StatusNoContent)
	check(reordered[1:])
	e.expect(e.do(http.MethodDelete, "/me/photos/"+reordered[0], u.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodDelete, "/me/photos/not-a-uuid", u.Token, nil), http.StatusNotFound)
}

func TestPhotoAccessControl(t *testing.T) {
	e := newEnv(t, false)
	p := e.matchedPair()
	lat, lng := scenarioOrigin()
	stranger := e.completeUser("stranger", profileSpec{Name: "Str", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})

	var photos struct {
		Photos []struct {
			ID string `json:"id"`
		} `json:"photos"`
	}
	e.expect(e.do(http.MethodGet, "/me/photos", p.b.Token, nil), http.StatusOK).json(t, &photos)
	bobPhoto := "/photos/" + photos.Photos[0].ID

	e.expect(e.do(http.MethodGet, bobPhoto, "", nil), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, bobPhoto, stranger.Token, nil), http.StatusOK) // visible profile

	// A hidden profile's pictures are private, except for its matches.
	e.expect(e.do(http.MethodPut, "/me/privacy", p.b.Token, map[string]any{"isVisible": false, "showDistance": true}), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, bobPhoto, stranger.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, bobPhoto, p.a.Token, nil), http.StatusOK)
	e.expect(e.do(http.MethodGet, bobPhoto, p.b.Token, nil), http.StatusOK)

	// Blocking cuts access in both directions.
	e.expect(e.do(http.MethodPost, "/blocks", p.a.Token, map[string]string{"userId": p.b.ID}), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, bobPhoto, p.a.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPut, "/me/privacy", p.b.Token, map[string]any{"isVisible": true, "showDistance": true}), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, bobPhoto, p.a.Token, nil), http.StatusNotFound)
}

func TestConversationPermissions(t *testing.T) {
	e := newEnv(t, false)
	p := e.matchedPair()
	lat, lng := scenarioOrigin()
	intruder := e.completeUser("intruder", profileSpec{Name: "Int", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})

	base := "/conversations/" + p.matchID
	e.expect(e.do(http.MethodPost, base+"/messages", p.a.Token, map[string]string{"body": "secret"}), http.StatusCreated)

	// Non-participants get 404 everywhere: they cannot read, write, mark read, hide or unmatch.
	e.expect(e.do(http.MethodGet, base+"/messages", intruder.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, base+"/messages", intruder.Token, map[string]string{"body": "hi"}), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, base+"/read", intruder.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodDelete, base, intruder.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodDelete, "/matches/"+p.matchID, intruder.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/conversations/not-a-uuid/messages", p.a.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, base+"/messages", "", nil), http.StatusUnauthorized)
	var list struct {
		Conversations []struct{} `json:"conversations"`
	}
	e.expect(e.do(http.MethodGet, "/conversations", intruder.Token, nil), http.StatusOK).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("intruder must not see foreign conversations")
	}

	// Message validation.
	for _, body := range []string{"", "   ", strings.Repeat("x", 2001), "bad\x00char"} {
		if r := e.do(http.MethodPost, base+"/messages", p.a.Token, map[string]string{"body": body}); r.Status != http.StatusBadRequest {
			t.Fatalf("body %q: expected 400, got %d", body[:min(len(body), 10)], r.Status)
		}
	}
	e.expect(e.do(http.MethodGet, base+"/messages?limit=0", p.a.Token, nil), http.StatusBadRequest)
	e.expect(e.do(http.MethodGet, base+"/messages?before=yesterday", p.a.Token, nil), http.StatusBadRequest)
}

func TestMessagePaginationHideAndUnmatch(t *testing.T) {
	e := newEnv(t, false)
	p := e.matchedPair()
	base := "/conversations/" + p.matchID

	for i := 0; i < 5; i++ {
		e.expect(e.do(http.MethodPost, base+"/messages", p.a.Token, map[string]string{"body": strings.Repeat("m", i+1)}), http.StatusCreated)
		time.Sleep(2 * time.Millisecond)
	}
	var page struct {
		Messages []struct {
			Body      string    `json:"body"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"messages"`
	}
	e.expect(e.do(http.MethodGet, base+"/messages?limit=2", p.b.Token, nil), http.StatusOK).json(t, &page)
	if len(page.Messages) != 2 || page.Messages[0].Body != "mmmmm" {
		t.Fatalf("expected newest first: %+v", page)
	}
	before := page.Messages[1].CreatedAt.Format(time.RFC3339Nano)
	e.expect(e.do(http.MethodGet, base+"/messages?limit=10&before="+before, p.b.Token, nil), http.StatusOK).json(t, &page)
	if len(page.Messages) != 3 || page.Messages[0].Body != "mmm" {
		t.Fatalf("expected the three older messages: %+v", page)
	}

	// Hiding removes the conversation only for the hider, and only until a new message arrives.
	e.expect(e.do(http.MethodDelete, base, p.b.Token, nil), http.StatusNoContent)
	var list struct {
		Conversations []struct {
			UnreadCount int `json:"unreadCount"`
		} `json:"conversations"`
	}
	e.expect(e.do(http.MethodGet, "/conversations", p.b.Token, nil), http.StatusOK).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("hidden conversation must disappear from the hider's list")
	}
	e.expect(e.do(http.MethodGet, "/conversations", p.a.Token, nil), http.StatusOK).json(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("the other participant keeps the conversation")
	}
	time.Sleep(5 * time.Millisecond)
	e.expect(e.do(http.MethodPost, base+"/messages", p.a.Token, map[string]string{"body": "back again"}), http.StatusCreated)
	e.expect(e.do(http.MethodGet, "/conversations", p.b.Token, nil), http.StatusOK).json(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 1 {
		t.Fatalf("a new message must resurface the conversation with only new unread messages: %+v", list)
	}
	e.expect(e.do(http.MethodGet, base+"/messages", p.b.Token, nil), http.StatusOK).json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "back again" {
		t.Fatalf("history before hiding must stay hidden: %+v", page)
	}

	// Unmatching deletes the conversation for both.
	e.expect(e.do(http.MethodDelete, "/matches/"+p.matchID, p.a.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, base+"/messages", p.b.Token, nil), http.StatusNotFound)
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM chat_messages WHERE match_id = $1`, p.matchID).Scan(&n)
	if n != 0 {
		t.Fatal("messages must be removed with the match")
	}
}

func TestBlockAndReport(t *testing.T) {
	e := newEnv(t, false)
	p := e.matchedPair()
	base := "/conversations/" + p.matchID
	e.expect(e.do(http.MethodPost, base+"/messages", p.b.Token, map[string]string{"body": "hello"}), http.StatusCreated)

	e.expect(e.do(http.MethodPost, "/blocks", p.a.Token, map[string]string{"userId": p.a.ID}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/blocks", p.a.Token, map[string]string{"userId": "nope"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/blocks", p.a.Token, map[string]string{"userId": "00000000-0000-4000-8000-000000000000"}), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/blocks", p.a.Token, map[string]string{"userId": p.b.ID}), http.StatusNoContent)
	e.expect(e.do(http.MethodPost, "/blocks", p.a.Token, map[string]string{"userId": p.b.ID}), http.StatusNoContent) // idempotent

	// Both sides lose the conversation, the profile and any way to write.
	for _, u := range []testUser{p.a, p.b} {
		var list struct {
			Conversations []struct{} `json:"conversations"`
		}
		e.expect(e.do(http.MethodGet, "/conversations", u.Token, nil), http.StatusOK).json(t, &list)
		if len(list.Conversations) != 0 {
			t.Fatal("blocked conversation must vanish for both users")
		}
		e.expect(e.do(http.MethodGet, base+"/messages", u.Token, nil), http.StatusNotFound)
		e.expect(e.do(http.MethodPost, base+"/messages", u.Token, map[string]string{"body": "hi"}), http.StatusNotFound)
	}
	e.expect(e.do(http.MethodGet, "/profiles/"+p.b.ID, p.a.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/profiles/"+p.a.ID, p.b.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/swipes", p.b.Token, map[string]string{"targetId": p.a.ID, "action": "like"}), http.StatusOK) // already swiped: idempotent, no leak

	var blocks struct {
		Blocks []struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
		} `json:"blocks"`
	}
	e.expect(e.do(http.MethodGet, "/blocks", p.a.Token, nil), http.StatusOK).json(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].ID != p.b.ID || blocks.Blocks[0].FirstName != "Bo" {
		t.Fatalf("unexpected block list: %+v", blocks)
	}
	e.expect(e.do(http.MethodGet, "/blocks", p.b.Token, nil), http.StatusOK).json(t, &blocks)
	if len(blocks.Blocks) != 0 {
		t.Fatal("the blocked user must not learn they are blocked")
	}

	// Unblocking restores the conversation.
	e.expect(e.do(http.MethodDelete, "/blocks/"+p.b.ID, p.a.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, base+"/messages", p.a.Token, nil), http.StatusOK)

	// Reports.
	e.expect(e.do(http.MethodPost, "/reports", p.a.Token, map[string]string{"userId": p.b.ID, "reason": "harassment", "details": "rude"}), http.StatusCreated)
	e.expect(e.do(http.MethodPost, "/reports", p.a.Token, map[string]string{"userId": p.b.ID, "reason": "bogus"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/reports", p.a.Token, map[string]string{"userId": p.a.ID, "reason": "spam"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/reports", p.a.Token, map[string]string{"userId": p.b.ID, "reason": "spam", "details": strings.Repeat("d", 1001)}), http.StatusBadRequest)
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open'`, p.a.ID, p.b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected one open report, got %d", n)
	}
}

func TestBlockedUsersDisappearFromDiscovery(t *testing.T) {
	e := newEnv(t, false)
	lat, lng := scenarioOrigin()
	a := e.completeUser("blk-a", profileSpec{Name: "A", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	b := e.completeUser("blk-b", profileSpec{Name: "B", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	if ids := idsOf(e.do(http.MethodGet, "/discover", a.Token, nil), t); !contains(ids, b.ID) {
		t.Fatal("setup: b should be discoverable")
	}
	// B blocks A: A must stop seeing B, and B must stop seeing A.
	e.expect(e.do(http.MethodPost, "/blocks", b.Token, map[string]string{"userId": a.ID}), http.StatusNoContent)
	if ids := idsOf(e.do(http.MethodGet, "/discover", a.Token, nil), t); contains(ids, b.ID) {
		t.Fatal("blocker still shown to the blocked user")
	}
	if ids := idsOf(e.do(http.MethodGet, "/discover", b.Token, nil), t); contains(ids, a.ID) {
		t.Fatal("blocked user still shown to the blocker")
	}
	e.expect(e.do(http.MethodPost, "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"}), http.StatusNotFound)
}

func TestNotifications(t *testing.T) {
	e := newEnv(t, false)
	p := e.matchedPair()

	var list struct {
		Notifications []struct {
			ID     string            `json:"id"`
			Type   string            `json:"type"`
			Data   map[string]string `json:"data"`
			IsRead bool              `json:"isRead"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do(http.MethodGet, "/notifications", p.a.Token, nil), http.StatusOK).json(t, &list)
	if len(list.Notifications) != 1 || list.Notifications[0].Data["matchId"] != p.matchID {
		t.Fatalf("expected a match notification carrying the match id: %+v", list)
	}

	// Offline recipients get one "message" notification, not one per message.
	for i := 0; i < 3; i++ {
		e.expect(e.do(http.MethodPost, "/conversations/"+p.matchID+"/messages", p.b.Token, map[string]string{"body": "ping"}), http.StatusCreated)
	}
	e.expect(e.do(http.MethodGet, "/notifications", p.a.Token, nil), http.StatusOK).json(t, &list)
	if list.UnreadCount != 2 {
		t.Fatalf("expected match + one collapsed message notification, got %+v", list)
	}

	// Users cannot read or touch each other's notifications, nor create any.
	foreign := list.Notifications[0].ID
	e.expect(e.do(http.MethodPost, "/notifications/"+foreign+"/read", p.b.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/notifications", p.a.Token, map[string]string{"type": "x", "title": "x", "body": "x"}), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/notifications/"+foreign+"/read", p.a.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, "/notifications", p.a.Token, nil), http.StatusOK).json(t, &list)
	if list.UnreadCount != 1 {
		t.Fatalf("expected 1 unread, got %d", list.UnreadCount)
	}
	e.expect(e.do(http.MethodPost, "/notifications/read-all", p.a.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, "/notifications", p.a.Token, nil), http.StatusOK).json(t, &list)
	if list.UnreadCount != 0 {
		t.Fatalf("expected 0 unread, got %d", list.UnreadCount)
	}
	e.expect(e.do(http.MethodGet, "/notifications?limit=500", p.a.Token, nil), http.StatusBadRequest)
}

func TestRealtimeMessagesOverWebSocket(t *testing.T) {
	e := newEnv(t, false)
	p := e.matchedPair()
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	dial := func(token string) *websocket.Conn {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		_ = c.WriteJSON(map[string]string{"type": "auth", "token": token})
		return c
	}
	next := func(c *websocket.Conn) map[string]any {
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		var m map[string]any
		if err := c.ReadJSON(&m); err != nil {
			t.Fatalf("read: %v", err)
		}
		return m
	}

	// A bad token is closed right away.
	bad := dial("not-a-token")
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("expected the connection to be closed")
	}
	bad.Close()

	bob := dial(p.b.Token)
	defer bob.Close()
	if m := next(bob); m["type"] != "ready" {
		t.Fatalf("expected ready, got %v", m)
	}

	e.expect(e.do(http.MethodPost, "/conversations/"+p.matchID+"/messages", p.a.Token, map[string]string{"body": "live!"}), http.StatusCreated)
	m := next(bob)
	raw, _ := json.Marshal(m["data"])
	if m["type"] != "message.new" || !strings.Contains(string(raw), "live!") {
		t.Fatalf("expected message.new, got %v", m)
	}

	// Connected users are not also spammed with an offline notification.
	var list struct {
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do(http.MethodGet, "/notifications", p.b.Token, nil), http.StatusOK).json(t, &list)
	if list.UnreadCount != 1 { // only the match notification
		t.Fatalf("expected only the match notification, got %d", list.UnreadCount)
	}

	// Read receipts reach the sender.
	ada := dial(p.a.Token)
	defer ada.Close()
	next(ada) // ready
	e.expect(e.do(http.MethodPost, "/conversations/"+p.matchID+"/read", p.b.Token, nil), http.StatusNoContent)
	if m := next(ada); m["type"] != "message.read" {
		t.Fatalf("expected message.read, got %v", m)
	}
}
