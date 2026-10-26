package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type cast struct{ a, b, far, wrongGender, hidden, outsideAge user }

func setupCast(e *env) cast {
	return cast{
		a:           e.newUser(userSpec{Name: "Alex", Gender: "man", InterestedIn: []string{"woman"}, Lat: parisLat, Lng: parisLng, AgeYears: 30, Interests: []string{"travel", "music"}}),
		b:           e.newUser(userSpec{Name: "Bea", Gender: "woman", InterestedIn: []string{"man"}, Lat: nearLat, Lng: nearLng, AgeYears: 28, Interests: []string{"music", "wine"}}),
		far:         e.newUser(userSpec{Name: "Faraway", Gender: "woman", InterestedIn: []string{"man"}, Lat: lyonLat, Lng: lyonLng, AgeYears: 28}),
		wrongGender: e.newUser(userSpec{Name: "Wrong", Gender: "man", InterestedIn: []string{"woman"}, Lat: nearLat, Lng: nearLng, AgeYears: 28}),
		hidden:      e.newUser(userSpec{Name: "Hidden", Gender: "woman", InterestedIn: []string{"man"}, Lat: nearLat, Lng: nearLng, AgeYears: 28}),
		outsideAge:  e.newUser(userSpec{Name: "Elder", Gender: "woman", InterestedIn: []string{"man"}, Lat: nearLat, Lng: nearLng, AgeYears: 62}),
	}
}

func TestDiscoveryAppliesPreferencesPrivacyAndPagination(t *testing.T) {
	e := newEnv(t)
	c := setupCast(e)
	e.expect(e.do("PUT", "/me/privacy", c.hidden.Token, map[string]bool{"isVisible": false, "showDistance": true}), 200, "hide profile")

	ids := e.discoverIDs(c.a)
	if !contains(ids, c.b.ID) {
		t.Fatalf("Bea should be proposed, got %v", ids)
	}
	for name, id := range map[string]string{"far": c.far.ID, "wrong gender": c.wrongGender.ID, "hidden": c.hidden.ID, "outside age": c.outsideAge.ID, "self": c.a.ID} {
		if contains(ids, id) {
			t.Fatalf("%s must not be proposed", name)
		}
	}

	// the card exposes only public data and an approximate distance
	r := e.expect(e.do("GET", "/discover", c.a.Token, nil), 200, "discover")
	card := r.list("profiles")[0]
	for _, forbidden := range []string{"email", "birthDate", "latitude", "longitude"} {
		if _, ok := card[forbidden]; ok {
			t.Fatalf("card leaks %s", forbidden)
		}
	}
	if d, ok := card["distanceKm"].(float64); !ok || int(d)%5 != 0 || d < 5 {
		t.Fatalf("distance must be rounded up to 5km steps, got %v", card["distanceKm"])
	}
	if ps, _ := card["photos"].([]any); len(ps) == 0 {
		t.Fatal("card must include photos")
	}
	if is, _ := card["interests"].([]any); len(is) != 2 {
		t.Fatalf("card must include interests: %v", card)
	}

	// hiding the distance removes it from the card
	e.expect(e.do("PUT", "/me/privacy", c.b.Token, map[string]bool{"isVisible": true, "showDistance": false}), 200, "hide distance")
	r = e.expect(e.do("GET", "/discover", c.a.Token, nil), 200, "discover")
	if r.list("profiles")[0]["distanceKm"] != nil {
		t.Fatal("distance must be hidden when the owner opts out")
	}

	// incomplete profiles cannot discover
	fresh := e.register(uniqueEmail("fresh"))
	e.expect(e.do("GET", "/discover", fresh.Token, nil), 409, "incomplete viewer")
	e.expect(e.do("GET", "/discover?limit=abc", c.a.Token, nil), 400, "bad limit")
	e.expect(e.do("GET", "/discover", "", nil), 401, "unauthenticated")

	// batch limit is respected and capped
	if got := e.discoverIDs(c.a); len(got) > 20 {
		t.Fatalf("batch must be capped, got %d", len(got))
	}
}

func TestSwipeMatchFlow(t *testing.T) {
	e := newEnv(t)
	c := setupCast(e)

	// swiping on a profile the server would not propose is refused
	e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.far.ID, "action": "like"}), 404, "out of range")
	e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.wrongGender.ID, "action": "like"}), 404, "not in preferences")
	e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.a.ID, "action": "like"}), 404, "self swipe")
	e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "action": "like"}), 404, "unknown user")
	e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": "nope", "action": "like"}), 400, "invalid id")
	e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.b.ID, "action": "superlike"}), 400, "invalid action")

	// A likes B: no match yet
	r := e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.b.ID, "action": "like"}), 200, "A likes B")
	if r.JSON["matched"] != false {
		t.Fatalf("one-sided like must not match: %s", r.Raw)
	}
	if contains(e.discoverIDs(c.a), c.b.ID) {
		t.Fatal("a swiped profile must never reappear")
	}

	// repeated like is idempotent
	r = e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.b.ID, "action": "like"}), 200, "A likes B again")
	if r.JSON["alreadySwiped"] != true || r.JSON["matched"] != false {
		t.Fatalf("repeat like must be flagged, not re-processed: %s", r.Raw)
	}
	if n := e.count(`SELECT count(*) FROM swipes WHERE from_user_id = $1`, c.a.ID); n != 1 {
		t.Fatalf("duplicate swipes stored: %d", n)
	}

	// B sees A first (who already liked them), then likes back -> match
	if ids := e.discoverIDs(c.b); len(ids) == 0 || ids[0] != c.a.ID {
		t.Fatalf("users who liked you are ranked first, got %v", ids)
	}
	r = e.expect(e.do("POST", "/discover/swipes", c.b.Token, map[string]string{"userId": c.a.ID, "action": "like"}), 200, "B likes A")
	if r.JSON["matched"] != true {
		t.Fatalf("mutual like must match: %s", r.Raw)
	}
	matchID := r.str("match", "id")
	if matchID == "" || r.str("match", "profile", "firstName") != "Alex" {
		t.Fatalf("match payload incomplete: %s", r.Raw)
	}

	// both sides get exactly one match notification, one match row
	if n := e.count(`SELECT count(*) FROM matches`); n != 1 {
		t.Fatalf("expected a single match, got %d", n)
	}
	for _, u := range []user{c.a, c.b} {
		nr := e.expect(e.do("GET", "/notifications", u.Token, nil), 200, "notifications")
		items := nr.list("notifications")
		if len(items) != 1 || items[0]["type"] != "match" {
			t.Fatalf("expected one match notification: %s", nr.Raw)
		}
	}

	// re-swiping after the match does not duplicate anything
	r = e.expect(e.do("POST", "/discover/swipes", c.b.Token, map[string]string{"userId": c.a.ID, "action": "like"}), 200, "re-like")
	if r.JSON["matched"] != false || r.JSON["alreadySwiped"] != true {
		t.Fatalf("unexpected: %s", r.Raw)
	}
	if n := e.count(`SELECT count(*) FROM matches`); n != 1 {
		t.Fatalf("duplicate match created: %d", n)
	}

	// a pass never creates a match even if the other side liked
	e.expect(e.do("POST", "/discover/swipes", c.hidden.Token, map[string]string{"userId": c.a.ID, "action": "like"}), 200, "hidden likes A")
	r = e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.hidden.ID, "action": "pass"}), 200, "A passes")
	if r.JSON["matched"] != false {
		t.Fatal("pass must not match")
	}
	if n := e.count(`SELECT count(*) FROM matches`); n != 1 {
		t.Fatalf("pass created a match: %d", n)
	}
}

func TestSimultaneousReciprocalLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		a := e.newUser(userSpec{Name: "Ra", Gender: "man", InterestedIn: []string{"woman"}, Lat: parisLat, Lng: parisLng, AgeYears: 30})
		b := e.newUser(userSpec{Name: "Rb", Gender: "woman", InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng, AgeYears: 30})
		var wg sync.WaitGroup
		results := make([]resp, 2)
		for i, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to user) {
				defer wg.Done()
				results[i] = e.do("POST", "/discover/swipes", from.Token, map[string]string{"userId": to.ID, "action": "like"})
			}(i, pair[0], pair[1])
		}
		wg.Wait()
		matched := 0
		for _, r := range results {
			if r.Status != 200 {
				t.Fatalf("swipe failed: %s", r.Raw)
			}
			if r.JSON["matched"] == true {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("round %d: exactly one side must observe the match, got %d", round, matched)
		}
		lo, hi := a.ID, b.ID
		if hi < lo {
			lo, hi = hi, lo
		}
		if n := e.count(`SELECT count(*) FROM matches WHERE user_a = $1 AND user_b = $2`, lo, hi); n != 1 {
			t.Fatalf("round %d: %d match rows", round, n)
		}
	}
}

func match(e *env, a, b user) string {
	e.t.Helper()
	e.expect(e.do("POST", "/discover/swipes", a.Token, map[string]string{"userId": b.ID, "action": "like"}), 200, "like")
	r := e.expect(e.do("POST", "/discover/swipes", b.Token, map[string]string{"userId": a.ID, "action": "like"}), 200, "like back")
	if r.JSON["matched"] != true {
		e.t.Fatalf("expected match: %s", r.Raw)
	}
	return r.str("match", "id")
}

func TestChatLifecycleAndAuthorization(t *testing.T) {
	e := newEnv(t)
	c := setupCast(e)
	matchID := match(e, c.a, c.b)

	// conversation shows up for both, without messages yet
	cr := e.expect(e.do("GET", "/conversations", c.a.Token, nil), 200, "conversations")
	convs := cr.list("conversations")
	if len(convs) != 1 || convs[0]["matchId"] != matchID || convs[0]["lastMessage"] != nil {
		t.Fatalf("unexpected conversations: %s", cr.Raw)
	}

	// validation
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "   "}), 400, "empty message")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": strings.Repeat("x", 2001)}), 400, "too long")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "bad\x00char"}), 400, "control char")

	// send / receive / unread / read receipts
	m1 := e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "  Salut Bea !  "}), 201, "send")
	if m1.str("body") != "Salut Bea !" || m1.str("createdAt") == "" {
		t.Fatalf("message not normalised: %s", m1.Raw)
	}
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "Ça va ?"}), 201, "send 2")

	convs = e.expect(e.do("GET", "/conversations", c.b.Token, nil), 200, "conversations B").list("conversations")
	if convs[0]["unreadCount"].(float64) != 2 || convs[0]["lastMessage"].(map[string]any)["body"] != "Ça va ?" {
		t.Fatalf("unread count/last message wrong: %v", convs[0])
	}
	// unread message notification is coalesced per conversation
	nr := e.expect(e.do("GET", "/notifications", c.b.Token, nil), 200, "notifications B")
	msgNotifs := 0
	for _, n := range nr.list("notifications") {
		if n["type"] == "message" {
			msgNotifs++
		}
	}
	if msgNotifs != 1 || nr.JSON["unreadCount"].(float64) != 2 {
		t.Fatalf("expected 1 coalesced message notification (+1 match): %s", nr.Raw)
	}

	msgs := e.expect(e.do("GET", "/conversations/"+matchID+"/messages", c.b.Token, nil), 200, "messages")
	list := msgs.list("messages")
	if len(list) != 2 || list[0]["body"] != "Ça va ?" || list[0]["readAt"] != nil {
		t.Fatalf("newest first and unread expected: %s", msgs.Raw)
	}
	rd := e.expect(e.do("POST", "/conversations/"+matchID+"/read", c.b.Token, nil), 200, "mark read")
	if rd.JSON["count"].(float64) != 2 {
		t.Fatalf("expected 2 messages read: %s", rd.Raw)
	}
	msgs = e.expect(e.do("GET", "/conversations/"+matchID+"/messages", c.a.Token, nil), 200, "messages A")
	if msgs.list("messages")[0]["readAt"] == nil {
		t.Fatal("sender must see the read state")
	}
	if got := e.expect(e.do("GET", "/notifications/unread-count", c.b.Token, nil), 200, "unread").JSON["unreadCount"].(float64); got != 1 {
		t.Fatalf("only the match notification should remain unread, got %v", got)
	}

	// pagination
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.b.Token, map[string]string{"body": "msg " + string(rune('a'+i))}), 201, "send")
	}
	page1 := e.expect(e.do("GET", "/conversations/"+matchID+"/messages?limit=3", c.a.Token, nil), 200, "page1")
	if len(page1.list("messages")) != 3 || page1.JSON["hasMore"] != true {
		t.Fatalf("page 1 wrong: %s", page1.Raw)
	}
	oldest := page1.list("messages")[2]["id"].(float64)
	page2 := e.expect(e.do("GET", "/conversations/"+matchID+"/messages?limit=10&before="+strconv.FormatInt(int64(oldest), 10), c.a.Token, nil), 200, "page2")
	if len(page2.list("messages")) != 4 || page2.JSON["hasMore"] != false {
		t.Fatalf("page 2 wrong: %s", page2.Raw)
	}

	// authorization: outsiders and unmatched users get 404 and nothing leaks
	for _, outsider := range []user{c.far, c.wrongGender} {
		e.expect(e.do("GET", "/conversations/"+matchID+"/messages", outsider.Token, nil), 404, "outsider reads")
		e.expect(e.do("POST", "/conversations/"+matchID+"/messages", outsider.Token, map[string]string{"body": "hi"}), 404, "outsider writes")
		e.expect(e.do("POST", "/conversations/"+matchID+"/read", outsider.Token, nil), 404, "outsider marks read")
		e.expect(e.do("DELETE", "/conversations/"+matchID, outsider.Token, nil), 404, "outsider clears")
		e.expect(e.do("GET", "/matches/"+matchID, outsider.Token, nil), 404, "outsider views match")
		e.expect(e.do("DELETE", "/matches/"+matchID, outsider.Token, nil), 404, "outsider unmatches")
		if len(e.expect(e.do("GET", "/conversations", outsider.Token, nil), 200, "list").list("conversations")) != 0 {
			t.Fatal("outsider sees conversations")
		}
	}
	e.expect(e.do("GET", "/conversations/not-a-uuid/messages", c.a.Token, nil), 404, "bad id")

	// local deletion hides history for one side only
	e.expect(e.do("DELETE", "/conversations/"+matchID, c.a.Token, nil), 204, "clear")
	if len(e.expect(e.do("GET", "/conversations", c.a.Token, nil), 200, "list").list("conversations")) != 0 {
		t.Fatal("cleared conversation must leave the list")
	}
	if len(e.expect(e.do("GET", "/conversations/"+matchID+"/messages", c.a.Token, nil), 200, "msgs").list("messages")) != 0 {
		t.Fatal("history must be hidden for the requester")
	}
	if len(e.expect(e.do("GET", "/conversations/"+matchID+"/messages", c.b.Token, nil), 200, "msgs").list("messages")) != 7 {
		t.Fatal("the other participant keeps the history")
	}
	time.Sleep(5 * time.Millisecond)
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.b.Token, map[string]string{"body": "back again"}), 201, "new message resurrects")
	convs = e.expect(e.do("GET", "/conversations", c.a.Token, nil), 200, "list").list("conversations")
	if len(convs) != 1 || convs[0]["lastMessage"].(map[string]any)["body"] != "back again" {
		t.Fatalf("conversation should reappear with the new message only: %v", convs)
	}

	// unmatch closes the conversation for both
	e.expect(e.do("DELETE", "/matches/"+matchID, c.a.Token, nil), 204, "unmatch")
	e.expect(e.do("GET", "/conversations/"+matchID+"/messages", c.b.Token, nil), 404, "closed for B")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "hello?"}), 404, "closed for A")
	if contains(e.discoverIDs(c.a), c.b.ID) {
		t.Fatal("unmatched users must not be proposed again")
	}
}

func (e *env) ws(token string) *websocket.Conn {
	e.t.Helper()
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		e.t.Fatalf("dial ws: %v", err)
	}
	e.t.Cleanup(func() { _ = conn.Close() })
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		e.t.Fatal(err)
	}
	return conn
}

func readEvent(t *testing.T, c *websocket.Conn) map[string]any {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ev map[string]any
	if err := c.ReadJSON(&ev); err != nil {
		t.Fatalf("read ws event: %v", err)
	}
	return ev
}

func TestRealtimeDelivery(t *testing.T) {
	e := newEnv(t)
	c := setupCast(e)

	wsB := e.ws(c.b.Token)
	if ev := readEvent(t, wsB); ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v", ev)
	}
	wsA := e.ws(c.a.Token)
	readEvent(t, wsA)

	// B is notified of the match in real time
	e.expect(e.do("POST", "/discover/swipes", c.b.Token, map[string]string{"userId": c.a.ID, "action": "like"}), 200, "B likes")
	e.expect(e.do("POST", "/discover/swipes", c.a.Token, map[string]string{"userId": c.b.ID, "action": "like"}), 200, "A likes back")
	ev := readEvent(t, wsB)
	if ev["type"] != "match" {
		t.Fatalf("expected match event, got %v", ev)
	}
	matchID := ev["data"].(map[string]any)["id"].(string)

	// messages are pushed to the recipient, read receipts to the sender
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "live!"}), 201, "send")
	ev = readEvent(t, wsB)
	if ev["type"] != "message" || ev["data"].(map[string]any)["body"] != "live!" {
		t.Fatalf("expected message event, got %v", ev)
	}
	readEvent(t, wsA) // echo of own message for multi-device sync
	e.expect(e.do("POST", "/conversations/"+matchID+"/read", c.b.Token, nil), 200, "read")
	ev = readEvent(t, wsA)
	if ev["type"] != "read" {
		t.Fatalf("expected read event, got %v", ev)
	}

	// outsiders receive nothing
	wsOut := e.ws(c.far.Token)
	readEvent(t, wsOut)
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "secret"}), 201, "send 2")
	_ = wsOut.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var leaked map[string]any
	if err := wsOut.ReadJSON(&leaked); err == nil {
		t.Fatalf("outsider received %v", leaked)
	}

	// unmatching notifies the other side
	e.expect(e.do("DELETE", "/matches/"+matchID, c.a.Token, nil), 204, "unmatch")
	for {
		ev = readEvent(t, wsB)
		if ev["type"] == "match_ended" {
			break
		}
	}
}

func TestWebSocketRejectsBadAuth(t *testing.T) {
	e := newEnv(t)
	conn := e.ws("not-a-token")
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("bad token must close the socket")
	}
	// a socket that never authenticates is closed
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	idle, _, err := websocket.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	defer idle.Close()
	_ = idle.SetReadDeadline(time.Now().Add(7 * time.Second))
	if _, _, err := idle.ReadMessage(); err == nil {
		t.Fatal("idle socket must be closed after the auth timeout")
	}
}

func TestBlockReportAndAccountDeletion(t *testing.T) {
	e := newEnv(t)
	c := setupCast(e)
	matchID := match(e, c.a, c.b)
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "hi"}), 201, "send")

	// report validation
	e.expect(e.do("POST", "/reports", c.a.Token, map[string]string{"userId": c.b.ID, "reason": "nonsense"}), 400, "bad reason")
	e.expect(e.do("POST", "/reports", c.a.Token, map[string]string{"userId": c.a.ID, "reason": "spam"}), 400, "self report")
	e.expect(e.do("POST", "/reports", c.a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}), 404, "unknown reported")
	e.expect(e.do("POST", "/reports", c.a.Token, map[string]any{"userId": c.b.ID, "reason": "spam", "details": strings.Repeat("x", 1001)}), 400, "details too long")
	e.expect(e.do("POST", "/reports", c.a.Token, map[string]string{"userId": c.b.ID, "reason": "harassment", "details": "insultes"}), 201, "report")
	if n := e.count(`SELECT count(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open'`, c.a.ID, c.b.ID); n != 1 {
		t.Fatalf("report not stored: %d", n)
	}

	// blocking ends the match and hides both users from each other
	e.expect(e.do("POST", "/blocks", c.a.Token, map[string]string{"userId": c.a.ID}), 400, "self block")
	e.expect(e.do("POST", "/blocks", c.a.Token, map[string]string{"userId": c.b.ID}), 204, "block")
	e.expect(e.do("POST", "/blocks", c.a.Token, map[string]string{"userId": c.b.ID}), 204, "block is idempotent")
	e.expect(e.do("GET", "/conversations/"+matchID+"/messages", c.b.Token, nil), 404, "blocked user loses the chat")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", c.a.Token, map[string]string{"body": "x"}), 404, "blocker cannot write")
	if contains(e.discoverIDs(c.a), c.b.ID) || contains(e.discoverIDs(c.b), c.a.ID) {
		t.Fatal("blocked users must disappear from discovery in both directions")
	}
	bl := e.expect(e.do("GET", "/blocks", c.a.Token, nil), 200, "blocks").list("blocks")
	if len(bl) != 1 || bl[0]["userId"] != c.b.ID || bl[0]["firstName"] != "Bea" {
		t.Fatalf("unexpected block list: %v", bl)
	}
	if len(e.expect(e.do("GET", "/blocks", c.b.Token, nil), 200, "blocks").list("blocks")) != 0 {
		t.Fatal("the blocked user must not see the block")
	}
	// even a direct swipe is refused
	e.expect(e.do("POST", "/discover/swipes", c.b.Token, map[string]string{"userId": c.a.ID, "action": "like"}), 404, "swipe on blocker")

	e.expect(e.do("DELETE", "/blocks/"+c.b.ID, c.a.Token, nil), 204, "unblock")
	// unblocking does not resurrect the match
	e.expect(e.do("GET", "/conversations/"+matchID+"/messages", c.a.Token, nil), 404, "match stays ended")

	// account deletion removes everything linked to the account
	e.expect(e.do("DELETE", "/me", c.b.Token, map[string]string{"password": "Wrong12345"}), 403, "wrong password")
	e.expect(e.do("DELETE", "/me", c.b.Token, map[string]string{"password": testPassword}), 204, "delete account")
	ctx := context.Background()
	for table, col := range map[string]string{
		"users": "id", "profiles": "user_id", "photos": "user_id", "user_preferences": "user_id", "user_interests": "user_id",
		"sessions": "user_id", "notifications": "user_id", "blocks": "blocked_id",
	} {
		var n int
		if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE `+col+` = $1`, c.b.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s still holds %d rows for the deleted user", table, n)
		}
	}
	if n := e.count(`SELECT count(*) FROM swipes WHERE from_user_id = $1 OR to_user_id = $1`, c.b.ID); n != 0 {
		t.Fatal("swipes must be deleted")
	}
	if n := e.count(`SELECT count(*) FROM matches WHERE user_a = $1 OR user_b = $1`, c.b.ID); n != 0 {
		t.Fatal("matches must be deleted")
	}
	if n := e.count(`SELECT count(*) FROM messages WHERE sender_id = $1`, c.b.ID); n != 0 {
		t.Fatal("messages must be deleted")
	}
	// reports are kept for moderation but detached from the deleted account
	if n := e.count(`SELECT count(*) FROM reports WHERE reported_id IS NULL AND reporter_id = $1`, c.a.ID); n != 1 {
		t.Fatal("reports must be kept anonymised")
	}
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": c.b.Email, "password": testPassword}), 401, "login after deletion")
	e.expect(e.do("GET", "/me", c.a.Token, nil), 200, "other users unaffected")
}
