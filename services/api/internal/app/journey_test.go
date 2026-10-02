package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

// Register -> profile -> discover -> like -> match -> chat -> read -> notifications.
func TestCriticalJourney(t *testing.T) {
	e := newEnv(t)
	alice := e.register("alice")
	bob := e.register("bob")
	carol := e.register("carol")
	e.onboard(alice, profileOpts{name: "Alice", gender: "woman", interestedIn: []string{"man"}, interests: []string{"travel", "music"}})
	e.onboard(bob, profileOpts{name: "Bob", gender: "man", interestedIn: []string{"woman"}, dLat: 0.05, interests: []string{"music", "yoga"}})
	e.onboard(carol, profileOpts{name: "Carol", gender: "woman", interestedIn: []string{"man"}, dLat: 0.02})

	// Orientation is enforced in both directions: Alice sees Bob only, Carol never sees Alice.
	if ids := e.discoverIDs(alice); !has(ids, bob.ID) || has(ids, carol.ID) || has(ids, alice.ID) {
		t.Fatalf("alice discover unexpected: %v", ids)
	}
	if ids := e.discoverIDs(carol); has(ids, alice.ID) || !has(ids, bob.ID) {
		t.Fatalf("carol discover unexpected: %v", ids)
	}

	// Cards never leak coordinates or birth date; distance is bucketed.
	r := e.expect(e.do("GET", "/discover?limit=5", alice.Token, nil), 200)
	raw := string(r.Raw)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("discover leaks %q: %s", forbidden, raw)
		}
	}
	card := r.JSON["profiles"].([]any)[0].(map[string]any)
	if d := card["distanceKm"].(float64); int(d)%5 != 0 || d < 5 {
		t.Fatalf("distance should be bucketed by 5 km, got %v", d)
	}

	// Alice likes Bob: no match yet. Repeating is idempotent.
	for i := 0; i < 2; i++ {
		s := e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), 200)
		if s.JSON["matched"] != false {
			t.Fatalf("must not match on one-sided like: %s", s.Raw)
		}
	}
	// Already processed profiles never come back.
	if has(e.discoverIDs(alice), bob.ID) {
		t.Fatal("liked profile must disappear from discovery")
	}
	// Changing your mind after the fact is ignored: first decision wins (no flip-flopping into matches).
	s := e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "pass"}), 200)
	if s.str("action") != "like" {
		t.Fatalf("repeat swipe must return the original decision, got %s", s.Raw)
	}

	// Bob sees Alice already (she liked him), likes back -> match.
	m := e.expect(e.do("POST", "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), 200)
	if m.JSON["matched"] != true || m.str("matchId") == "" || m.str("user", "firstName") != "Alice" {
		t.Fatalf("expected match with Alice: %s", m.Raw)
	}
	matchID := m.str("matchId")
	// Repeating returns the same match, creates no duplicate.
	m2 := e.expect(e.do("POST", "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), 200)
	if m2.str("matchId") != matchID {
		t.Fatal("duplicate match created")
	}
	var matchRows int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, alice.ID, bob.ID).Scan(&matchRows)
	if matchRows != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchRows)
	}

	// Chat: participants only.
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", alice.Token, map[string]string{"body": "  Hello Bob  "}), 201)
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", alice.Token, map[string]string{"body": "   "}), 400)
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", alice.Token, map[string]string{"body": strings.Repeat("x", 2001)}), 400)
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", carol.Token, map[string]string{"body": "intruder"}), 404)
	e.expect(e.do("GET", "/matches/"+matchID+"/messages", carol.Token, nil), 404)
	e.expect(e.do("POST", "/matches/"+matchID+"/read", carol.Token, nil), 404)
	e.expect(e.do("DELETE", "/matches/"+matchID, carol.Token, nil), 404)

	list := e.expect(e.do("GET", "/matches", bob.Token, nil), 200)
	item := list.JSON["matches"].([]any)[0].(map[string]any)
	if item["unreadCount"].(float64) != 1 || item["lastMessage"].(map[string]any)["body"] != "Hello Bob" {
		t.Fatalf("unexpected match list item: %v", item)
	}
	sum := e.expect(e.do("GET", "/notifications/summary", bob.Token, nil), 200)
	if sum.JSON["unreadMessages"].(float64) != 1 || sum.JSON["unreadNotifications"].(float64) != 2 {
		t.Fatalf("unexpected summary (match + message notif): %s", sum.Raw)
	}
	// Ten more messages do not create ten notifications.
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/matches/"+matchID+"/messages", alice.Token, map[string]string{"body": "again"}), 201)
	}
	sum = e.expect(e.do("GET", "/notifications/summary", bob.Token, nil), 200)
	if sum.JSON["unreadNotifications"].(float64) != 2 {
		t.Fatalf("message notifications must be coalesced: %s", sum.Raw)
	}

	msgs := e.expect(e.do("GET", "/matches/"+matchID+"/messages?limit=3", bob.Token, nil), 200)
	if got := len(msgs.JSON["messages"].([]any)); got != 3 || msgs.JSON["hasMore"] != true {
		t.Fatalf("pagination: %s", msgs.Raw)
	}
	first := msgs.JSON["messages"].([]any)[0].(map[string]any)
	older := e.expect(e.do("GET", "/matches/"+matchID+"/messages?limit=10&before="+jsonNum(first["id"]), bob.Token, nil), 200)
	if len(older.JSON["messages"].([]any)) != 3 {
		t.Fatalf("older page: %s", older.Raw)
	}

	read := e.expect(e.do("POST", "/matches/"+matchID+"/read", bob.Token, nil), 200)
	if read.JSON["updated"].(float64) != 6 {
		t.Fatalf("read: %s", read.Raw)
	}
	sum = e.expect(e.do("GET", "/notifications/summary", bob.Token, nil), 200)
	if sum.JSON["unreadMessages"].(float64) != 0 || sum.JSON["unreadNotifications"].(float64) != 1 {
		t.Fatalf("after read: %s", sum.Raw)
	}
	n := e.expect(e.do("GET", "/notifications", alice.Token, nil), 200)
	if len(n.JSON["notifications"].([]any)) != 1 {
		t.Fatalf("alice notifications: %s", n.Raw)
	}
	e.expect(e.do("POST", "/notifications/read", alice.Token, nil), 204)
	n = e.expect(e.do("GET", "/notifications", alice.Token, nil), 200)
	if n.JSON["unread"].(float64) != 0 {
		t.Fatal("notifications should all be read")
	}

	// Local deletion hides history for one side only.
	e.expect(e.do("DELETE", "/matches/"+matchID+"/messages", bob.Token, nil), 204)
	if got := len(e.expect(e.do("GET", "/matches/"+matchID+"/messages", bob.Token, nil), 200).JSON["messages"].([]any)); got != 0 {
		t.Fatalf("bob history should be cleared, got %d", got)
	}
	if got := len(e.expect(e.do("GET", "/matches/"+matchID+"/messages", alice.Token, nil), 200).JSON["messages"].([]any)); got != 6 {
		t.Fatalf("alice history must stay, got %d", got)
	}
}

func jsonNum(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestRealtimeDeliversMessages(t *testing.T) {
	e := newEnv(t)
	alice := e.register("rt-alice")
	bob := e.register("rt-bob")
	e.onboard(alice, profileOpts{name: "Alice", gender: "woman", interestedIn: []string{"man"}})
	e.onboard(bob, profileOpts{name: "Bob", gender: "man", interestedIn: []string{"woman"}})
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), 200)

	// Missing / wrong-type tickets are refused (an access token is not a ticket).
	e.expect(e.do("GET", "/realtime/ws", "", nil), 401)
	e.expect(e.do("GET", "/realtime/ws?ticket="+bob.Token, "", nil), 401)

	ticket := e.expect(e.do("POST", "/realtime/ticket", bob.Token, nil), 200).str("ticket")
	url := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/realtime/ws?ticket=" + ticket
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	read := func() map[string]any {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read ws: %v", err)
		}
		var ev map[string]any
		_ = json.Unmarshal(data, &ev)
		return ev
	}

	m := e.expect(e.do("POST", "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), 200)
	if ev := read(); ev["type"] != "match" {
		t.Fatalf("expected match event, got %v", ev)
	}
	e.expect(e.do("POST", "/matches/"+m.str("matchId")+"/messages", alice.Token, map[string]string{"body": "live!"}), 201)
	ev := read()
	if ev["type"] != "message" || ev["payload"].(map[string]any)["body"] != "live!" {
		t.Fatalf("expected message event, got %v", ev)
	}
}

func TestBlockUnmatchReportAndDelete(t *testing.T) {
	e := newEnv(t)
	a := e.register("sa")
	b := e.register("sb")
	e.onboard(a, profileOpts{name: "A", gender: "woman", interestedIn: []string{"man"}})
	e.onboard(b, profileOpts{name: "B", gender: "man", interestedIn: []string{"woman"}})
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"}), 200)
	m := e.expect(e.do("POST", "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"}), 200)
	matchID := m.str("matchId")
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", a.Token, map[string]string{"body": "hi"}), 201)

	// Reports: validated, never against yourself.
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": a.ID, "reason": "spam"}), 400)
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "nope"}), 400)
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}), 404)

	// Report + block in one go: the match, messages and visibility disappear for both sides.
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "rude", "block": true}), 201)
	var reports int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("report not stored")
	}
	e.expect(e.do("GET", "/matches/"+matchID+"/messages", a.Token, nil), 404)
	e.expect(e.do("GET", "/matches/"+matchID+"/messages", b.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 404)
	if has(e.discoverIDs(a), b.ID) || has(e.discoverIDs(b), a.ID) {
		t.Fatal("blocked users must not see each other")
	}
	e.expect(e.do("POST", "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"}), 409)
	if bl := e.expect(e.do("GET", "/blocks", a.Token, nil), 200); len(bl.JSON["blocks"].([]any)) != 1 {
		t.Fatalf("blocks list: %s", bl.Raw)
	}
	// Unblocking brings them back to discovery with a clean slate.
	e.expect(e.do("DELETE", "/blocks/"+b.ID, a.Token, nil), 204)
	if !has(e.discoverIDs(a), b.ID) {
		t.Fatal("unblocked user should be discoverable again")
	}

	// Unmatch path.
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"}), 200)
	m = e.expect(e.do("POST", "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"}), 200)
	e.expect(e.do("DELETE", "/matches/"+m.str("matchId"), b.Token, nil), 204)
	e.expect(e.do("POST", "/matches/"+m.str("matchId")+"/messages", a.Token, map[string]string{"body": "x"}), 404)

	// Account deletion: wrong password refused, then everything is erased and the token dies at once.
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong-password"}), 403)
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "Password123"}), 204)
	e.expect(e.do("GET", "/me", a.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Password123"}), 401)
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "swipes": "swiper_id", "preferences": "user_id", "reports": "reporter_id"} {
		var n int
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM `+table+` WHERE `+col+` = $1`, a.ID).Scan(&n)
		if n != 0 {
			t.Fatalf("%s still holds rows of the deleted user", table)
		}
	}
	// The other user is untouched and no longer sees the deleted one.
	e.expect(e.do("GET", "/me", b.Token, nil), 200)
	if got := e.expect(e.do("GET", "/matches", b.Token, nil), 200).JSON["matches"].([]any); len(got) != 0 {
		t.Fatalf("matches of deleted user should be gone: %v", got)
	}
}

func TestConcurrentOppositeLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	a := e.register("ca")
	b := e.register("cb")
	e.onboard(a, profileOpts{name: "A", gender: "woman", interestedIn: []string{"man"}})
	e.onboard(b, profileOpts{name: "B", gender: "man", interestedIn: []string{"woman"}})

	done := make(chan resp, 2)
	go func() {
		done <- e.do("POST", "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"})
	}()
	go func() {
		done <- e.do("POST", "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"})
	}()
	r1, r2 := <-done, <-done
	if r1.Status != 200 || r2.Status != 200 {
		t.Fatalf("statuses %d %d", r1.Status, r2.Status)
	}
	if !(r1.JSON["matched"] == true || r2.JSON["matched"] == true) {
		t.Fatal("at least one of the simultaneous likes must report the match")
	}
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected exactly 1 match, got %d", n)
	}
}

var _ = httptest.NewRequest
var _ = http.StatusOK
