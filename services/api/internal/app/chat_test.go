package app_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func pair(e *env) (a, b user, conv string) {
	a = e.newUser(spec{Name: "Ann", Gender: "woman", InterestedIn: []string{"man"}})
	b = e.newUser(spec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	return a, b, e.match(a, b)
}

func TestChatPermissionsAndFlow(t *testing.T) {
	e := newEnv(t)
	a, b, conv := pair(e)
	outsider := e.newUser(spec{Name: "Eve", Gender: "woman", InterestedIn: []string{"man"}})

	// Conversation exists right after the match, for both participants only.
	for _, u := range []user{a, b} {
		list := e.want(e.do("GET", "/conversations", u.Token, nil), 200).json()["conversations"].([]any)
		if len(list) != 1 {
			t.Fatalf("expected 1 conversation, got %d", len(list))
		}
	}
	if l := e.want(e.do("GET", "/conversations", outsider.Token, nil), 200).json()["conversations"].([]any); len(l) != 0 {
		t.Fatal("outsider must see no conversation")
	}

	// Outsiders get 404 (not 403) on every conversation route: nothing leaks.
	e.want(e.do("GET", "/conversations/"+conv+"/messages", outsider.Token, nil), 404)
	e.want(e.do("POST", "/conversations/"+conv+"/messages", outsider.Token, map[string]string{"body": "hi"}), 404)
	e.want(e.do("POST", "/conversations/"+conv+"/read", outsider.Token, nil), 404)
	e.want(e.do("DELETE", "/conversations/"+conv, outsider.Token, nil), 404)
	e.want(e.do("GET", "/conversations/"+conv+"/messages", "", nil), 401)
	e.want(e.do("GET", "/conversations/not-a-uuid/messages", a.Token, nil), 404)

	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "   "}), 400)
	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": strings.Repeat("é", 2001)}), 400)
	m := e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "  Salut Bob !  "}), 201).json()
	if m["body"] != "Salut Bob !" || m["readAt"] != nil || m["senderId"] != a.ID {
		t.Fatalf("message: %v", m)
	}
	if _, err := time.Parse(time.RFC3339Nano, m["createdAt"].(string)); err != nil {
		t.Fatalf("createdAt: %v", err)
	}
	e.want(e.do("POST", "/conversations/"+conv+"/messages", b.Token, map[string]string{"body": "Salut Ann"}), 201)
	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "Comment vas-tu ?"}), 201)

	// Unread counts are per recipient.
	cb := e.want(e.do("GET", "/conversations", b.Token, nil), 200).json()["conversations"].([]any)[0].(map[string]any)
	if cb["unreadCount"].(float64) != 2 || cb["lastMessage"].(map[string]any)["body"] != "Comment vas-tu ?" {
		t.Fatalf("bob view: %v", cb)
	}
	ca := e.want(e.do("GET", "/conversations", a.Token, nil), 200).json()["conversations"].([]any)[0].(map[string]any)
	if ca["unreadCount"].(float64) != 1 {
		t.Fatalf("ann view: %v", ca)
	}

	// Reading marks only the other side's messages.
	if n := e.want(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 200).json()["read"].(float64); n != 2 {
		t.Fatalf("read count %v", n)
	}
	msgs := e.want(e.do("GET", "/conversations/"+conv+"/messages", a.Token, nil), 200).json()["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("messages: %d", len(msgs))
	}
	// Newest first; Ann's messages are read, Bob's is not yet.
	if msgs[0].(map[string]any)["readAt"] == nil || msgs[1].(map[string]any)["readAt"] != nil {
		t.Fatalf("read receipts wrong: %v", msgs)
	}

	// Pagination by cursor.
	oldest := msgs[2].(map[string]any)["createdAt"].(string)
	older := e.want(e.do("GET", "/conversations/"+conv+"/messages?before="+oldest, a.Token, nil), 200).json()["messages"].([]any)
	if len(older) != 0 {
		t.Fatalf("nothing older than the oldest: %v", older)
	}
	page := e.want(e.do("GET", "/conversations/"+conv+"/messages?limit=1", a.Token, nil), 200).json()["messages"].([]any)
	if len(page) != 1 {
		t.Fatalf("limit: %d", len(page))
	}
	e.want(e.do("GET", "/conversations/"+conv+"/messages?before=yesterday", a.Token, nil), 400)

	// Local deletion hides it for Bob only, until a new message arrives; old history stays hidden.
	e.want(e.do("DELETE", "/conversations/"+conv, b.Token, nil), 204)
	if l := e.want(e.do("GET", "/conversations", b.Token, nil), 200).json()["conversations"].([]any); len(l) != 0 {
		t.Fatal("deleted conversation must be hidden")
	}
	if l := e.want(e.do("GET", "/conversations", a.Token, nil), 200).json()["conversations"].([]any); len(l) != 1 {
		t.Fatal("Ann keeps her conversation")
	}
	time.Sleep(5 * time.Millisecond)
	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "Tu es là ?"}), 201)
	back := e.want(e.do("GET", "/conversations", b.Token, nil), 200).json()["conversations"].([]any)
	if len(back) != 1 {
		t.Fatal("new message must resurface the conversation")
	}
	hist := e.want(e.do("GET", "/conversations/"+conv+"/messages", b.Token, nil), 200).json()["messages"].([]any)
	if len(hist) != 1 {
		t.Fatalf("history before deletion must stay hidden, got %d", len(hist))
	}
}

func TestBlockStopsChatAndMatch(t *testing.T) {
	e := newEnv(t)
	a, b, conv := pair(e)
	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "hello"}), 201)
	e.want(e.do("POST", "/blocks", b.Token, map[string]any{"userId": a.ID}), 204)
	e.want(e.do("POST", "/blocks", b.Token, map[string]any{"userId": a.ID}), 204) // idempotent
	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "still there?"}), 404)
	e.want(e.do("GET", "/conversations/"+conv+"/messages", b.Token, nil), 404)
	for _, u := range []user{a, b} {
		if l := e.want(e.do("GET", "/matches", u.Token, nil), 200).json()["matches"].([]any); len(l) != 0 {
			t.Fatal("block must remove the match")
		}
		if l := e.want(e.do("GET", "/conversations", u.Token, nil), 200).json()["conversations"].([]any); len(l) != 0 {
			t.Fatal("block must remove the conversation")
		}
	}
	if contains(e.feedIDs(a), b.ID) || contains(e.feedIDs(b), a.ID) {
		t.Fatal("blocked users must not see each other")
	}
	e.want(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 404)
	e.want(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 404)
	c := e.newUser(spec{Name: "Cat", Gender: "man", InterestedIn: []string{"woman"}})
	e.want(e.do("POST", "/blocks", a.Token, map[string]any{"userId": c.ID}), 204)
	e.want(e.swipe(a, c, "like"), 404)
	e.want(e.swipe(c, a, "like"), 404)

	bl := e.want(e.do("GET", "/blocks", b.Token, nil), 200).json()["blocks"].([]any)
	if len(bl) != 1 || bl[0].(map[string]any)["userId"] != a.ID {
		t.Fatalf("blocks list: %v", bl)
	}
	for _, item := range e.want(e.do("GET", "/blocks", a.Token, nil), 200).json()["blocks"].([]any) {
		if item.(map[string]any)["userId"] == b.ID {
			t.Fatal("the blocked user must not learn about the block")
		}
	}
	e.want(e.do("POST", "/blocks", b.Token, map[string]any{"userId": b.ID}), 400)
	e.want(e.do("POST", "/blocks", b.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000"}), 404)
	e.want(e.do("DELETE", "/blocks/"+a.ID, b.Token, nil), 204)
	e.want(e.do("DELETE", "/blocks/"+a.ID, b.Token, nil), 404)
	// After unblocking they can see each other again (swipes were kept, so no new match yet).
	e.want(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 200)
}

func TestNotifications(t *testing.T) {
	e := newEnv(t)
	a, b, conv := pair(e)
	for _, u := range []user{a, b} {
		r := e.want(e.do("GET", "/notifications", u.Token, nil), 200).json()
		list := r["notifications"].([]any)
		if len(list) != 1 || list[0].(map[string]any)["type"] != "match" || r["unreadCount"].(float64) != 1 {
			t.Fatalf("match notification missing: %v", r)
		}
	}
	// Message notifications coalesce per conversation.
	for i := 0; i < 3; i++ {
		e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "hey"}), 201)
	}
	r := e.want(e.do("GET", "/notifications", b.Token, nil), 200).json()
	if r["unreadCount"].(float64) != 2 {
		t.Fatalf("expected match + 1 coalesced message notification: %v", r)
	}
	// Another user cannot mark someone else's notification.
	id := r["notifications"].([]any)[0].(map[string]any)["id"].(string)
	e.want(e.do("POST", "/notifications/"+id+"/read", a.Token, nil), 404)
	e.want(e.do("POST", "/notifications/"+id+"/read", b.Token, nil), 204)
	if n := e.want(e.do("GET", "/notifications/unread-count", b.Token, nil), 200).json()["unreadCount"].(float64); n != 1 {
		t.Fatalf("unread after one read: %v", n)
	}
	// Reading the conversation clears its notification.
	e.want(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 200)
	e.want(e.do("POST", "/notifications/read-all", b.Token, nil), 204)
	if n := e.want(e.do("GET", "/notifications/unread-count", b.Token, nil), 200).json()["unreadCount"].(float64); n != 0 {
		t.Fatalf("read-all: %v", n)
	}
	e.want(e.do("GET", "/notifications", "", nil), 401)
}

func TestReports(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(spec{Name: "Rep", Gender: "woman"})
	b := e.newUser(spec{Name: "Bad", Gender: "man"})
	e.want(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "nonsense"}), 400)
	e.want(e.do("POST", "/reports", a.Token, map[string]any{"userId": a.ID, "reason": "spam"}), 400)
	e.want(e.do("POST", "/reports", a.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}), 404)
	e.want(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "spam", "details": strings.Repeat("x", 1001)}), 400)
	e.want(e.do("POST", "/reports", "", map[string]any{"userId": b.ID, "reason": "spam"}), 401)
	e.want(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "insultes"}), 202)
	e.want(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "insultes"}), 202)
	if n := count(e, `SELECT count(*) FROM reports WHERE reporter_id=$1 AND reported_id=$2 AND status='open'`, a.ID, b.ID); n != 1 {
		t.Fatalf("repeated report must be deduplicated, got %d", n)
	}
	e.want(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "spam", "alsoBlock": true}), 202)
	if n := count(e, `SELECT count(*) FROM blocks WHERE blocker_id=$1 AND blocked_id=$2`, a.ID, b.ID); n != 1 {
		t.Fatal("alsoBlock must block")
	}
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e := newEnv(t)
	a, b, conv := pair(e)
	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "salut"}), 201)
	e.want(e.do("POST", "/reports", b.Token, map[string]any{"userId": a.ID, "reason": "spam"}), 202)
	var keys []string
	rows, _ := e.pool.Query(t.Context(), `SELECT storage_key FROM photos WHERE user_id=$1 UNION ALL SELECT thumb_key FROM photos WHERE user_id=$1`, a.ID)
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) != 2 {
		t.Fatalf("expected 2 stored files, got %d", len(keys))
	}

	e.want(e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong-password"}), 403)
	e.want(e.do("DELETE", "/me", a.Token, nil), 400)
	e.want(e.do("DELETE", "/me", "", map[string]string{"password": a.Password}), 401)
	e.want(e.do("DELETE", "/me", a.Token, map[string]string{"password": a.Password}), 204)

	for _, q := range []string{
		`SELECT count(*) FROM users WHERE id = $1`, `SELECT count(*) FROM profiles WHERE user_id = $1`,
		`SELECT count(*) FROM photos WHERE user_id = $1`, `SELECT count(*) FROM preferences WHERE user_id = $1`,
		`SELECT count(*) FROM sessions WHERE user_id = $1`, `SELECT count(*) FROM swipes WHERE from_user_id = $1 OR to_user_id = $1`,
		`SELECT count(*) FROM messages WHERE sender_id = $1`, `SELECT count(*) FROM notifications WHERE user_id = $1 OR actor_id = $1`,
		`SELECT count(*) FROM matches WHERE user_a_id = $1 OR user_b_id = $1`,
	} {
		if n := count(e, q, a.ID); n != 0 {
			t.Errorf("%s -> %d rows left", q, n)
		}
	}
	for _, k := range keys {
		if _, err := osStat(e.uploads, k); err == nil {
			t.Errorf("file %s must be deleted", k)
		}
	}
	// Reports are kept for moderation but no longer point to the deleted account.
	if n := count(e, `SELECT count(*) FROM reports WHERE reported_id IS NULL AND reporter_id = $1`, b.ID); n != 1 {
		t.Errorf("report should be retained anonymised, got %d", n)
	}
	// The other user is intact and the dead account cannot be used.
	e.want(e.do("GET", "/me", b.Token, nil), 200)
	if l := e.want(e.do("GET", "/matches", b.Token, nil), 200).json()["matches"].([]any); len(l) != 0 {
		t.Fatal("match must disappear")
	}
	e.want(e.do("GET", "/me", a.Token, nil), 401)
	e.want(e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": a.Password}), 401)
	e.want(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": a.Refresh}), 401)
}

func TestRealtimeDelivery(t *testing.T) {
	e := newEnv(t)
	a, b, conv := pair(e)
	srv := httptest.NewServer(e.svc.Router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?ticket="

	e.want(e.do("POST", "/realtime/ticket", "", nil), 401)
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL+"bogus", nil); err == nil || resp == nil || resp.StatusCode != 401 {
		t.Fatalf("invalid ticket must be refused (err=%v)", err)
	}
	ticket := e.want(e.do("POST", "/realtime/ticket", b.Token, nil), 200).json()["ticket"].(string)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+ticket, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	// Tickets are single-use.
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL+ticket, nil); err == nil || resp == nil || resp.StatusCode != 401 {
		t.Fatalf("ticket reuse must be refused (err=%v)", err)
	}

	e.want(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "en direct"}), 201)
	got := map[string]bool{}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for len(got) < 2 {
		var ev struct {
			Type string         `json:"type"`
			Data map[string]any `json:"data"`
		}
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read (got %v): %v", got, err)
		}
		if ev.Type == "message.new" && ev.Data["body"] != "en direct" {
			t.Fatalf("payload: %v", ev.Data)
		}
		got[ev.Type] = true
	}
	if !got["message.new"] || !got["notification.new"] {
		t.Fatalf("events: %v", got)
	}
}

func TestAuthRateLimit(t *testing.T) {
	e := newEnv(t)
	e.fixedIP = "203.0.113.9:5000"
	u := e.register("limit")
	var limited bool
	for i := 0; i < 15; i++ {
		if e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "wrong-password"}).Status == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("repeated failed logins from one IP must be rate limited")
	}
}
