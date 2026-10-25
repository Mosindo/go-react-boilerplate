package app_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type msg struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type messagesPage struct {
	Items      []msg   `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

type conversationsPage struct {
	Items []struct {
		ID          string `json:"id"`
		UnreadCount int    `json:"unreadCount"`
		LastMessage *msg   `json:"lastMessage"`
		OtherUser   struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"otherUser"`
	} `json:"items"`
}

func pair(t *testing.T) (*env, *testUser, *testUser, string) {
	e := newEnv(t)
	a := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	b := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	return e, a, b, matchWith(t, a, b)
}

func TestChatFlow(t *testing.T) {
	e, alice, bob, conv := pair(t)

	// Nothing to list until somebody writes.
	var convs conversationsPage
	e.expect(alice.do("GET", "/conversations", nil), 200).decode(t, &convs)
	if len(convs.Items) != 0 {
		t.Fatal("conversations without messages should not be listed")
	}

	// Validation.
	for _, body := range []string{"", "   ", strings.Repeat("a", 2001), "bad\x00char"} {
		if r := alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": body}); r.Status != 400 {
			t.Fatalf("body %q: expected 400, got %d", body, r.Status)
		}
	}
	var first msg
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "  Salut Bob  "}), 201).decode(t, &first)
	if first.Body != "Salut Bob" || first.SenderID != alice.ID || first.ReadAt != nil {
		t.Fatalf("unexpected message %+v", first)
	}
	for i := 0; i < 4; i++ {
		e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "message " + string(rune('A'+i))}), 201)
	}

	// Bob's view: one conversation, 5 unread, preview of the last message.
	e.expect(bob.do("GET", "/conversations", nil), 200).decode(t, &convs)
	if len(convs.Items) != 1 || convs.Items[0].UnreadCount != 5 || convs.Items[0].LastMessage.Body != "message D" || convs.Items[0].OtherUser.FirstName != "Alice" {
		t.Fatalf("unexpected conversations %+v", convs)
	}
	var unread struct{ UnreadCount int }
	e.expect(bob.do("GET", "/conversations/unread-count", nil), 200).decode(t, &unread)
	if unread.UnreadCount != 5 {
		t.Fatalf("unread=%d", unread.UnreadCount)
	}
	e.expect(alice.do("GET", "/conversations/unread-count", nil), 200).decode(t, &unread)
	if unread.UnreadCount != 0 {
		t.Fatal("sender has nothing unread")
	}

	// Pagination goes back in time, newest first, without gaps or duplicates.
	var page1, page2 messagesPage
	e.expect(bob.do("GET", "/conversations/"+conv+"/messages?limit=3", nil), 200).decode(t, &page1)
	if len(page1.Items) != 3 || page1.NextCursor == nil || page1.Items[0].Body != "message D" {
		t.Fatalf("page1: %+v", page1)
	}
	e.expect(bob.do("GET", "/conversations/"+conv+"/messages?limit=3&cursor="+url.QueryEscape(*page1.NextCursor), nil), 200).decode(t, &page2)
	if len(page2.Items) != 2 || page2.NextCursor != nil || page2.Items[1].ID != first.ID {
		t.Fatalf("page2: %+v", page2)
	}
	e.expect(bob.do("GET", "/conversations/"+conv+"/messages?cursor=garbage", nil), 400)

	// Reading marks messages read for the sender's view too.
	e.expect(bob.do("POST", "/conversations/"+conv+"/read", nil), 204)
	e.expect(bob.do("GET", "/conversations/unread-count", nil), 200).decode(t, &unread)
	if unread.UnreadCount != 0 {
		t.Fatal("expected all read")
	}
	var after messagesPage
	e.expect(alice.do("GET", "/conversations/"+conv+"/messages", nil), 200).decode(t, &after)
	for _, m := range after.Items {
		if m.ReadAt == nil {
			t.Fatalf("message %s should be read", m.Body)
		}
	}

	// Local deletion hides the history for Bob only, and a new message brings the thread back.
	e.expect(bob.do("DELETE", "/conversations/"+conv, nil), 204)
	e.expect(bob.do("GET", "/conversations", nil), 200).decode(t, &convs)
	if len(convs.Items) != 0 {
		t.Fatal("hidden conversation should be gone from Bob's list")
	}
	e.expect(bob.do("GET", "/conversations/"+conv+"/messages", nil), 200).decode(t, &after)
	if len(after.Items) != 0 {
		t.Fatal("Bob's cleared history should be empty")
	}
	e.expect(alice.do("GET", "/conversations/"+conv+"/messages", nil), 200).decode(t, &after)
	if len(after.Items) != 5 {
		t.Fatalf("Alice keeps her history, got %d", len(after.Items))
	}
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "tu es là ?"}), 201)
	e.expect(bob.do("GET", "/conversations", nil), 200).decode(t, &convs)
	if len(convs.Items) != 1 || convs.Items[0].UnreadCount != 1 {
		t.Fatalf("conversation should reappear with the new message: %+v", convs)
	}
}

func TestChatPermissions(t *testing.T) {
	e, _, _, conv := pair(t)
	mallory := e.signup("Mallory", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	for _, c := range []struct{ method, path string }{
		{"GET", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/read"},
		{"DELETE", "/conversations/" + conv},
	} {
		if r := mallory.do(c.method, c.path, nil); r.Status != 404 {
			t.Fatalf("%s %s by outsider: expected 404, got %d", c.method, c.path, r.Status)
		}
	}
	if r := mallory.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hello"}); r.Status != 404 {
		t.Fatalf("outsider send: %d", r.Status)
	}
	e.expect(mallory.do("GET", "/conversations/not-a-uuid/messages", nil), 404)
	e.expect(e.do("GET", "/conversations", "", nil), 401)
}

func TestUnmatchBlockAndNotifications(t *testing.T) {
	e, alice, bob, conv := pair(t)

	var notifs struct {
		Items []struct {
			ID     string  `json:"id"`
			Type   string  `json:"type"`
			ReadAt *string `json:"readAt"`
			Actor  *struct {
				FirstName string `json:"firstName"`
			} `json:"actor"`
		} `json:"items"`
		UnreadCount int `json:"unreadCount"`
	}
	// Alice swiped second... Bob liked second: Bob saw the match immediately (read), Alice has an unread one.
	e.expect(alice.do("GET", "/notifications", nil), 200).decode(t, &notifs)
	if notifs.UnreadCount != 1 || len(notifs.Items) != 1 || notifs.Items[0].Type != "match" || notifs.Items[0].Actor.FirstName != "Bob" {
		t.Fatalf("alice notifications: %+v", notifs)
	}
	e.expect(bob.do("GET", "/notifications", nil), 200).decode(t, &notifs)
	if notifs.UnreadCount != 0 || len(notifs.Items) != 1 {
		t.Fatalf("bob notifications: %+v", notifs)
	}

	// Message notifications are coalesced per conversation.
	for i := 0; i < 3; i++ {
		e.expect(bob.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hey"}), 201)
	}
	e.expect(alice.do("GET", "/notifications", nil), 200).decode(t, &notifs)
	if notifs.UnreadCount != 2 || len(notifs.Items) != 2 || notifs.Items[0].Type != "message" {
		t.Fatalf("expected match + one coalesced message notification: %+v", notifs)
	}
	// Opening the conversation clears the message notification.
	e.expect(alice.do("POST", "/conversations/"+conv+"/read", nil), 204)
	e.expect(alice.do("GET", "/notifications", nil), 200).decode(t, &notifs)
	if notifs.UnreadCount != 1 {
		t.Fatalf("message notification should be read: %+v", notifs)
	}
	e.expect(alice.do("POST", "/notifications/"+notifs.Items[1].ID+"/read", nil), 204)
	e.expect(alice.do("POST", "/notifications/read-all", nil), 204)
	e.expect(alice.do("GET", "/notifications", nil), 200).decode(t, &notifs)
	if notifs.UnreadCount != 0 {
		t.Fatal("everything should be read")
	}
	// Notifications belong to their owner.
	e.expect(bob.do("POST", "/notifications/"+notifs.Items[0].ID+"/read", nil), 404)

	// Block: conversation and match disappear for both, nobody can write, reciprocal invisibility.
	e.expect(alice.do("POST", "/blocks", map[string]string{"userId": alice.ID}), 400)
	e.expect(alice.do("POST", "/blocks", map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}), 404)
	e.expect(alice.do("POST", "/blocks", map[string]string{"userId": bob.ID}), 204)
	e.expect(alice.do("POST", "/blocks", map[string]string{"userId": bob.ID}), 204) // idempotent
	for _, u := range []*testUser{alice, bob} {
		e.expect(u.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "x"}), 404)
		var matches struct{ Items []any }
		e.expect(u.do("GET", "/matches", nil), 200).decode(t, &matches)
		if len(matches.Items) != 0 {
			t.Fatal("match should be gone after a block")
		}
	}
	if contains(alice.feedIDs(), bob.ID) || contains(bob.feedIDs(), alice.ID) {
		t.Fatal("blocked users must not see each other")
	}
	// Blocking before any swipe makes the blocker unreachable.
	carol := e.signup("Carol", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	dan := e.signup("Dan", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	e.expect(carol.do("POST", "/blocks", map[string]string{"userId": dan.ID}), 204)
	if st, _ := dan.swipe(carol.ID, "like"); st != 404 {
		t.Fatalf("swiping a blocker: %d", st)
	}
	if st, _ := carol.swipe(dan.ID, "like"); st != 404 {
		t.Fatalf("swiping a blocked user: %d", st)
	}
	var blocked struct {
		Items []struct{ UserID, FirstName string } `json:"items"`
	}
	e.expect(alice.do("GET", "/blocks", nil), 200).decode(t, &blocked)
	if len(blocked.Items) != 1 || blocked.Items[0].UserID != bob.ID || blocked.Items[0].FirstName != "Bob" {
		t.Fatalf("blocked list: %+v", blocked)
	}
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM messages`).Scan(&n)
	if n != 0 {
		t.Fatalf("messages should be deleted with the conversation, %d left", n)
	}

	// Unblock restores visibility but not the match: previous swipes are final.
	e.expect(alice.do("DELETE", "/blocks/"+bob.ID, nil), 204)
	if contains(alice.feedIDs(), bob.ID) {
		t.Fatal("already swiped users stay out of the feed after unblock")
	}
}

func TestUnmatch(t *testing.T) {
	e, alice, bob, conv := pair(t)
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hi"}), 201)
	var matches struct {
		Items []struct {
			MatchID string `json:"matchId"`
		}
	}
	e.expect(alice.do("GET", "/matches", nil), 200).decode(t, &matches)
	id := matches.Items[0].MatchID
	e.expect(bob.do("DELETE", "/matches/"+id, nil), 204)
	e.expect(bob.do("DELETE", "/matches/"+id, nil), 404)
	e.expect(alice.do("GET", "/conversations/"+conv+"/messages", nil), 404)
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM conversations`).Scan(&n)
	if n != 0 {
		t.Fatal("conversation should be removed")
	}
}

func TestReports(t *testing.T) {
	e, alice, bob, _ := pair(t)
	e.expect(alice.do("POST", "/reports", map[string]string{"userId": bob.ID, "reason": "harassment", "details": "  insultes  "}), 201)
	for name, body := range map[string]map[string]string{
		"self":           {"userId": alice.ID, "reason": "spam"},
		"bad reason":     {"userId": bob.ID, "reason": "dislike"},
		"unknown target": {"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"},
		"long details":   {"userId": bob.ID, "reason": "spam", "details": strings.Repeat("x", 1001)},
		"bad id":         {"userId": "x", "reason": "spam"},
	} {
		if r := alice.do("POST", "/reports", body); r.Status != 400 && r.Status != 404 {
			t.Fatalf("%s: expected 400/404, got %d", name, r.Status)
		}
	}
	var reason, details, status string
	if err := e.pool.QueryRow(t.Context(), `SELECT reason, details, status FROM reports`).Scan(&reason, &details, &status); err != nil {
		t.Fatal(err)
	}
	if reason != "harassment" || details != "insultes" || status != "open" {
		t.Fatalf("stored report: %s %s %s", reason, details, status)
	}
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e, alice, bob, conv := pair(t)
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hi"}), 201)
	e.expect(bob.do("POST", "/reports", map[string]string{"userId": alice.ID, "reason": "spam"}), 201)
	e.expect(bob.do("POST", "/blocks", map[string]string{"userId": alice.ID}), 204)

	e.expect(alice.do("DELETE", "/me", map[string]string{"password": "wrong-password"}), 403)
	e.expect(alice.do("DELETE", "/me", map[string]string{"password": "correct-horse-1"}), 204)

	e.expect(alice.do("GET", "/me", nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": alice.Email, "password": "correct-horse-1"}), 401)
	for _, table := range []string{"profiles", "preferences", "photos", "user_interests", "swipes", "matches", "conversations",
		"conversation_participants", "messages", "blocks", "reports", "notifications", "sessions", "password_resets"} {
		var n int
		col := "user_id"
		switch table {
		case "swipes":
			col = "from_user_id"
		case "blocks":
			col = "blocker_id"
		case "reports":
			col = "reporter_id"
		}
		query := "SELECT COUNT(*) FROM " + table + " WHERE " + col + " = $1"
		if table == "matches" {
			query = "SELECT COUNT(*) FROM matches WHERE user_a = $1 OR user_b = $1"
		}
		if table == "conversations" || table == "messages" {
			query = "SELECT COUNT(*) FROM " + table
		}
		args := []any{alice.ID}
		if table == "conversations" || table == "messages" {
			args = nil
		}
		if err := e.pool.QueryRow(t.Context(), query, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		if n != 0 {
			t.Fatalf("%s still has %d rows for the deleted user", table, n)
		}
	}
	var reported int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM reports WHERE reported_id = $1 OR reporter_id = $1`, alice.ID).Scan(&reported)
	var swipesToHer int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM swipes WHERE to_user_id = $1`, alice.ID).Scan(&swipesToHer)
	if reported != 0 || swipesToHer != 0 {
		t.Fatalf("rows referencing the deleted user remain: reports=%d swipes=%d", reported, swipesToHer)
	}
	// The partner is untouched.
	e.expect(bob.do("GET", "/me", nil), 200)
}

func dialWS(t *testing.T, e *env) *websocket.Conn {
	t.Helper()
	u := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func readEvent(t *testing.T, c *websocket.Conn) (string, json.RawMessage) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	var evt struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := c.ReadJSON(&evt); err != nil {
		t.Fatalf("read event: %v", err)
	}
	return evt.Type, evt.Data
}

func TestWebSocketRealtime(t *testing.T) {
	e := newEnv(t)
	alice := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})

	// Bad token -> closed with 4401.
	bad := dialWS(t, e)
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "nope"})
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err := bad.ReadMessage()
	if ce, ok := err.(*websocket.CloseError); !ok || ce.Code != 4401 {
		t.Fatalf("expected close 4401, got %v", err)
	}
	// Not authenticating as first frame is refused as well.
	silent := dialWS(t, e)
	_ = silent.WriteJSON(map[string]string{"type": "ping"})
	_ = silent.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := silent.ReadMessage(); err == nil {
		t.Fatal("expected the socket to be closed")
	}

	wsA, wsB := dialWS(t, e), dialWS(t, e)
	for _, p := range []struct {
		c *websocket.Conn
		u *testUser
	}{{wsA, alice}, {wsB, bob}} {
		_ = p.c.WriteJSON(map[string]string{"type": "auth", "token": p.u.Token})
		if typ, _ := readEvent(t, p.c); typ != "ready" {
			t.Fatalf("expected ready, got %s", typ)
		}
	}
	_ = wsA.WriteJSON(map[string]string{"type": "ping"})
	if typ, _ := readEvent(t, wsA); typ != "pong" {
		t.Fatalf("expected pong, got %s", typ)
	}

	// Match: Alice likes, Bob likes back -> Alice's socket gets "match" + "notification"? (Bob swiped second.)
	alice.swipe(bob.ID, "like")
	_, res := bob.swipe(alice.ID, "like")
	got := map[string]json.RawMessage{}
	for i := 0; i < 2; i++ {
		typ, data := readEvent(t, wsA)
		got[typ] = data
	}
	if got["match"] == nil || got["notification"] == nil {
		t.Fatalf("alice should receive match + notification, got %v", got)
	}
	conv := res.Match.ConversationID

	// Message delivery to the recipient and echo to the sender.
	e.expect(alice.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "coucou"}), 201)
	gotB := map[string]json.RawMessage{}
	for i := 0; i < 2; i++ {
		typ, data := readEvent(t, wsB)
		gotB[typ] = data
	}
	var delivered msg
	if err := json.Unmarshal(gotB["message"], &delivered); err != nil || delivered.Body != "coucou" || delivered.SenderID != alice.ID {
		t.Fatalf("bob should receive the message: %v %s", err, gotB["message"])
	}
	if gotB["notification"] == nil {
		t.Fatal("bob should receive a message notification")
	}
	if typ, _ := readEvent(t, wsA); typ != "message" {
		t.Fatalf("sender echo expected, got %s", typ)
	}

	// Read receipt reaches the sender.
	e.expect(bob.do("POST", "/conversations/"+conv+"/read", nil), 204)
	typ, data := readEvent(t, wsA)
	var read struct {
		ConversationID string `json:"conversationId"`
		ReaderID       string `json:"readerId"`
	}
	_ = json.Unmarshal(data, &read)
	if typ != "read" || read.ConversationID != conv || read.ReaderID != bob.ID {
		t.Fatalf("read event: %s %s", typ, data)
	}

	// Unmatch -> both clients are told, and deleting the account closes the socket.
	e.expect(alice.do("POST", "/blocks", map[string]string{"userId": bob.ID}), 204)
	for _, c := range []*websocket.Conn{wsA, wsB} {
		if typ, _ := readEvent(t, c); typ != "conversation_removed" {
			t.Fatalf("expected conversation_removed, got %s", typ)
		}
	}
	e.expect(bob.do("DELETE", "/me", map[string]string{"password": "correct-horse-1"}), 204)
	_ = wsB.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := wsB.ReadMessage(); err != nil {
			break
		}
	}
}
