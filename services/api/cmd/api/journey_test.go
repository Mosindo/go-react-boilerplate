package main

import (
	"net/http"
	"strings"
	"testing"
)

// TestJourney walks the critical product path:
// register → profile → photo → discover → like → match → realtime chat → read receipts → notifications.
func TestJourney(t *testing.T) {
	e := newEnv(t)

	alice := e.onboard(userSpec{Name: "Alice", Gender: "woman", Age: 29, InterestedIn: []string{"man"}, HasLocation: true, Lat: 48.8566, Lon: 2.3522, Interests: []int{1, 2}})
	bob := e.onboard(userSpec{Name: "Bob", Gender: "man", Age: 31, InterestedIn: []string{"woman"}, HasLocation: true, Lat: 48.8800, Lon: 2.3600, Interests: []int{2, 3}})
	e.onboard(userSpec{Name: "Carl", Gender: "man", Age: 40, InterestedIn: []string{"man"}}) // not compatible with Alice

	// /me reflects a complete profile.
	me := e.expect(e.call("GET", "/me", alice.Token, nil), 200)
	if me.Body["profileComplete"] != true {
		t.Fatalf("profileComplete should be true: %s", me.Raw)
	}

	// Alice only sees Bob (gender/preference filtering), with an approximate distance and no coordinates.
	disc := e.expect(e.call("GET", "/discover?limit=10", alice.Token, nil), 200)
	profiles := disc.list("profiles")
	if len(profiles) != 1 {
		t.Fatalf("alice should see exactly 1 candidate, got %d: %s", len(profiles), disc.Raw)
	}
	first := profiles[0].(map[string]any)
	if first["userId"] != bob.ID || first["firstName"] != "Bob" {
		t.Fatalf("unexpected candidate: %v", first)
	}
	if _, leaked := first["latitude"]; leaked {
		t.Fatal("coordinates must never be exposed")
	}
	if strings.Contains(string(disc.Raw), "birthDate") {
		t.Fatal("birth date must not be exposed on public profiles")
	}
	if d, ok := first["distanceKm"].(float64); !ok || d < 1 || int(d)%5 != 0 && d != 1 {
		t.Fatalf("distance should be approximated, got %v", first["distanceKm"])
	}

	// Photo of a visible profile is downloadable; it is a JPEG without metadata.
	photoURL := first["photos"].([]any)[0].(map[string]any)["url"].(string)
	req, _ := http.NewRequest("GET", e.srv.URL+photoURL, nil)
	req.Header.Set("Authorization", "Bearer "+alice.Token)
	img := e.do(req)
	if img.Status != 200 || img.Header.Get("Content-Type") != "image/jpeg" || len(img.Raw) < 1000 {
		t.Fatalf("photo download failed: %d %s", img.Status, img.Header.Get("Content-Type"))
	}
	// Photos require authentication.
	anon, _ := http.NewRequest("GET", e.srv.URL+photoURL, nil)
	if r := e.do(anon); r.Status != 401 {
		t.Fatalf("anonymous photo access should be 401, got %d", r.Status)
	}

	_, aliceEvents := e.ws(alice.Token)
	_, bobEvents := e.ws(bob.Token)

	// Alice likes Bob: no match yet. Repeating is idempotent.
	sw := e.expect(e.call("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"}), 200)
	if sw.Body["matched"] != false {
		t.Fatalf("no match expected yet: %s", sw.Raw)
	}
	e.expect(e.call("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"}), 200)

	// Bob already liked back would match; he now sees Alice first and likes her.
	bobDisc := e.expect(e.call("GET", "/discover", bob.Token, nil), 200)
	if len(bobDisc.list("profiles")) != 1 {
		t.Fatalf("bob should see alice: %s", bobDisc.Raw)
	}
	// Already-swiped profiles do not come back for Alice.
	if again := e.expect(e.call("GET", "/discover", alice.Token, nil), 200); len(again.list("profiles")) != 0 {
		t.Fatalf("swiped profile reappeared: %s", again.Raw)
	}

	bobSwipe := e.expect(e.call("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}), 200)
	if bobSwipe.Body["matched"] != true {
		t.Fatalf("expected a match: %s", bobSwipe.Raw)
	}
	convID := bobSwipe.str("conversation", "id")
	if convID == "" {
		t.Fatal("match response must carry the conversation")
	}
	// Both sides get the match live.
	waitEvent(t, aliceEvents, "match")
	waitEvent(t, bobEvents, "match")
	// Repeat after the match returns the stored result with the same conversation.
	rep := e.expect(e.call("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}), 200)
	if rep.str("conversation", "id") != convID {
		t.Fatal("repeated swipe must return the same conversation")
	}

	// Matched users no longer appear in discovery.
	if d := e.expect(e.call("GET", "/discover", bob.Token, nil), 200); len(d.list("profiles")) != 0 {
		t.Fatal("matched profile must leave the queue")
	}

	// Conversation lists show the match for both.
	list := e.expect(e.call("GET", "/conversations", alice.Token, nil), 200)
	if len(list.list("conversations")) != 1 {
		t.Fatalf("alice should have 1 conversation: %s", list.Raw)
	}

	// Chat: Alice writes, Bob receives live, unread count shows, read receipt flows back.
	msg := e.expect(e.call("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "  Salut Bob !  "}), 201)
	if msg.str("body") != "Salut Bob !" {
		t.Fatalf("body should be trimmed: %s", msg.Raw)
	}
	ev := waitEvent(t, bobEvents, "message")
	if ev["data"].(map[string]any)["body"] != "Salut Bob !" {
		t.Fatalf("unexpected ws message: %v", ev)
	}
	waitEvent(t, bobEvents, "notification")

	bobList := e.expect(e.call("GET", "/conversations", bob.Token, nil), 200)
	conv0 := bobList.list("conversations")[0].(map[string]any)
	if conv0["unreadCount"].(float64) != 1 || conv0["lastMessage"] == nil {
		t.Fatalf("bob should have 1 unread: %v", conv0)
	}
	// A second message does not create a second unread notification.
	e.expect(e.call("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "Ça va ?"}), 201)
	notifs := e.expect(e.call("GET", "/notifications", bob.Token, nil), 200)
	if notifs.Body["unreadCount"].(float64) != 2 { // match + one deduplicated message notification
		t.Fatalf("expected 2 unread notifications (match+message), got %v", notifs.Body["unreadCount"])
	}

	e.expect(e.call("POST", "/conversations/"+convID+"/read", bob.Token, nil), 204)
	receipt := waitEvent(t, aliceEvents, "read")
	if receipt["data"].(map[string]any)["readerId"] != bob.ID {
		t.Fatalf("unexpected receipt: %v", receipt)
	}
	msgs := e.expect(e.call("GET", "/conversations/"+convID+"/messages?limit=1", alice.Token, nil), 200)
	if len(msgs.list("messages")) != 1 || msgs.Body["nextCursor"] == nil {
		t.Fatalf("pagination expected: %s", msgs.Raw)
	}
	if msgs.list("messages")[0].(map[string]any)["readAt"] == nil {
		t.Fatal("message should be marked read")
	}
	page2 := e.expect(e.call("GET", "/conversations/"+convID+"/messages?limit=1&cursor="+msgs.Body["nextCursor"].(string), alice.Token, nil), 200)
	if len(page2.list("messages")) != 1 || page2.list("messages")[0].(map[string]any)["body"] != "Salut Bob !" {
		t.Fatalf("second page wrong: %s", page2.Raw)
	}
	afterRead := e.expect(e.call("GET", "/notifications", bob.Token, nil), 200)
	if afterRead.Body["unreadCount"].(float64) != 1 {
		t.Fatalf("reading the conversation should clear its message notification: %s", afterRead.Raw)
	}
	e.expect(e.call("POST", "/notifications/read-all", bob.Token, nil), 204)

	// Local deletion hides history for Alice only; Bob still sees everything.
	e.expect(e.call("DELETE", "/conversations/"+convID, alice.Token, nil), 204)
	hidden := e.expect(e.call("GET", "/conversations", alice.Token, nil), 200)
	if len(hidden.list("conversations")) != 0 {
		t.Fatal("cleared conversation should be hidden for alice")
	}
	if b := e.expect(e.call("GET", "/conversations/"+convID+"/messages", bob.Token, nil), 200); len(b.list("messages")) != 2 {
		t.Fatal("bob must keep the history")
	}
	e.expect(e.call("POST", "/conversations/"+convID+"/messages", bob.Token, map[string]any{"body": "Toujours là ?"}), 201)
	back := e.expect(e.call("GET", "/conversations", alice.Token, nil), 200)
	if len(back.list("conversations")) != 1 {
		t.Fatal("a new message should bring the conversation back")
	}
	if m := e.expect(e.call("GET", "/conversations/"+convID+"/messages", alice.Token, nil), 200); len(m.list("messages")) != 1 {
		t.Fatal("alice should only see messages after she cleared")
	}

	// Unmatch removes the conversation for both; they never see each other again.
	e.expect(e.call("POST", "/conversations/"+convID+"/unmatch", alice.Token, nil), 204)
	e.expect(e.call("GET", "/conversations/"+convID+"/messages", bob.Token, nil), 404)
	if d := e.expect(e.call("GET", "/discover", alice.Token, nil), 200); len(d.list("profiles")) != 0 {
		t.Fatal("unmatched users must not reappear")
	}
}
