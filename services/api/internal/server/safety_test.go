package server_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBlockDissolvesMatchAndHidesBothWays(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)
	conv := e.match(a, b)
	a.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "salut"})

	var photoID string
	_ = e.pool.QueryRow(t.Context(), `SELECT id FROM photos WHERE user_id=$1`, b.ID).Scan(&photoID)

	a.expect(400, "POST", "/blocks", map[string]string{"userId": a.ID})
	a.expect(404, "POST", "/blocks", map[string]string{"userId": "00000000-0000-0000-0000-000000000000"})
	a.expect(204, "POST", "/blocks", map[string]string{"userId": b.ID})
	a.expect(204, "POST", "/blocks", map[string]string{"userId": b.ID}) // idempotent

	// Conversation, messages and match are gone for both sides.
	for _, u := range []*user{a, b} {
		u.expect(404, "GET", "/conversations/"+conv+"/messages", nil)
		u.expect(404, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "x"})
	}
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM matches)+(SELECT count(*) FROM messages)`).Scan(&n)
	if n != 0 {
		t.Fatalf("match and messages must be removed, %d rows left", n)
	}

	// Mutual invisibility.
	if contains(a.discover(), b.ID) || contains(b.discover(), a.ID) {
		t.Fatal("blocked users must not see each other in discover")
	}
	a.expect(404, "GET", "/profiles/"+b.ID, nil)
	b.expect(404, "GET", "/profiles/"+a.ID, nil)
	a.expect(404, "POST", "/swipes", map[string]string{"userId": b.ID, "action": "like"})
	b.expect(404, "POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"})
	a.expect(404, "GET", "/photos/"+photoID, nil)

	var blocks struct {
		Blocks []struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"blocks"`
	}
	a.expect(200, "GET", "/blocks", nil).decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != b.ID || blocks.Blocks[0].FirstName != "Beth" {
		t.Fatalf("block list: %+v", blocks)
	}
	b.expect(200, "GET", "/blocks", nil).decode(t, &blocks)
	if len(blocks.Blocks) != 0 {
		t.Fatal("the blocked user must not learn they were blocked via /blocks")
	}

	a.expect(204, "DELETE", "/blocks/"+b.ID, nil)
	a.expect(204, "DELETE", "/blocks/"+b.ID, nil)
	a.expect(400, "DELETE", "/blocks/not-a-uuid", nil)
	a.expect(200, "GET", "/profiles/"+b.ID, nil)
}

func TestReports(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)
	conv := e.match(a, b)
	b.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "message offensant"})

	a.expect(400, "POST", "/reports", map[string]string{"userId": b.ID, "reason": "because"})
	a.expect(400, "POST", "/reports", map[string]string{"userId": a.ID, "reason": "spam"})
	a.expect(404, "POST", "/reports", map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"})
	a.expect(400, "POST", "/reports", map[string]string{"userId": b.ID, "reason": "spam", "details": string(make([]rune, 1001))})
	a.expect(204, "POST", "/reports", map[string]string{"userId": b.ID, "reason": "harassment", "details": " il m'insulte "})
	a.expect(204, "POST", "/reports", map[string]string{"userId": b.ID, "reason": "harassment"}) // dedup within 24h
	a.expect(204, "POST", "/reports", map[string]string{"userId": b.ID, "reason": "spam"})

	var count int
	var details, ctx string
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM reports`).Scan(&count)
	_ = e.pool.QueryRow(t.Context(), `SELECT details, context::text FROM reports WHERE reason='harassment'`).Scan(&details, &ctx)
	if count != 2 {
		t.Fatalf("expected 2 reports (one deduplicated), got %d", count)
	}
	if details != "il m'insulte" || !containsStr(ctx, "message offensant") {
		t.Fatalf("report must keep trimmed details and a message snapshot: %q %q", details, ctx)
	}

	// Evidence survives the reporter blocking (which deletes the conversation).
	a.expect(204, "POST", "/blocks", map[string]string{"userId": b.ID})
	_ = e.pool.QueryRow(t.Context(), `SELECT context::text FROM reports WHERE reason='harassment'`).Scan(&ctx)
	if !containsStr(ctx, "message offensant") {
		t.Fatal("snapshot must outlive the conversation")
	}
}

func containsStr(s, sub string) bool { return contains2(s, sub) }

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)
	c := e.onboard("Cleo", woman)
	conv := e.match(a, b)
	a.swipe(c, "like")
	a.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hello"})
	a.expect(204, "POST", "/blocks", map[string]string{"userId": c.ID})
	a.expect(204, "POST", "/reports", map[string]string{"userId": c.ID, "reason": "spam"})

	var key string
	_ = e.pool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE user_id=$1`, a.ID).Scan(&key)
	file := filepath.Join(e.uploads, key[:2], key)
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("photo should exist on disk: %v", err)
	}

	a.expect(400, "POST", "/me/delete", map[string]string{})
	if r := a.do("POST", "/me/delete", map[string]string{"password": "wrong-password"}); r.Status != 403 {
		t.Fatalf("wrong password must be refused, got %d", r.Status)
	}
	if r := e.do("POST", "/me/delete", "", map[string]string{"password": a.Password}); r.Status != 401 {
		t.Fatalf("auth required: %d", r.Status)
	}
	a.expect(204, "POST", "/me/delete", map[string]string{"password": a.Password})

	var left int
	for table, col := range map[string]string{
		"users": "id", "profiles": "user_id", "preferences": "user_id", "photos": "user_id", "user_interests": "user_id",
		"sessions": "user_id", "notifications": "user_id", "blocks": "blocker_id", "reports": "reporter_id",
		"conversation_members": "user_id", "messages": "sender_id", "swipes": "from_user",
	} {
		_ = e.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE "+col+"=$1", a.ID).Scan(&left)
		if left != 0 {
			t.Errorf("%s still has %d row(s) for the deleted user", table, left)
		}
	}
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches`).Scan(&left)
	if left != 0 {
		t.Error("matches involving the deleted user must be removed")
	}
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM swipes WHERE to_user=$1`, a.ID).Scan(&left)
	if left != 0 {
		t.Error("swipes targeting the deleted user must be removed")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("photo files must be deleted from disk")
	}

	if r := e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": a.Password}); r.Status != 401 {
		t.Fatalf("deleted account cannot log in, got %d", r.Status)
	}
	b.expect(404, "GET", "/conversations/"+conv, nil)
	if contains(b.discover(), a.ID) {
		t.Fatal("deleted users vanish from discover")
	}
}

func TestNotifications(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)
	conv := e.match(a, b)

	type notifList struct {
		Notifications []struct {
			ID     string  `json:"id"`
			Type   string  `json:"type"`
			ReadAt *string `json:"readAt"`
			Actor  *struct {
				FirstName string `json:"firstName"`
			} `json:"actor"`
		} `json:"notifications"`
		UnreadCount int    `json:"unreadCount"`
		NextCursor  string `json:"nextCursor"`
	}
	var list notifList
	a.expect(200, "GET", "/notifications", nil).decode(t, &list)
	if len(list.Notifications) != 1 || list.Notifications[0].Type != "new_match" || list.Notifications[0].Actor.FirstName != "Beth" || list.UnreadCount != 1 {
		t.Fatalf("match must notify both sides: %+v", list)
	}

	// A burst of messages produces a single unread notification.
	for i := 0; i < 3; i++ {
		b.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "yo"})
	}
	a.expect(200, "GET", "/notifications", nil).decode(t, &list)
	if list.UnreadCount != 2 || len(list.Notifications) != 2 || list.Notifications[0].Type != "new_message" {
		t.Fatalf("messages should be coalesced into one notification: %+v", list)
	}

	// Reading the conversation clears its message notification.
	a.expect(204, "POST", "/conversations/"+conv+"/read", nil)
	a.expect(200, "GET", "/notifications", nil).decode(t, &list)
	if list.UnreadCount != 1 {
		t.Fatalf("expected 1 unread after reading the chat: %+v", list)
	}

	// Mark a single one read; other users cannot touch it.
	matchNotif := list.Notifications[1].ID
	b.expect(404, "POST", "/notifications/"+matchNotif+"/read", nil)
	a.expect(404, "POST", "/notifications/not-a-uuid/read", nil)
	a.expect(204, "POST", "/notifications/"+matchNotif+"/read", nil)
	a.expect(200, "GET", "/notifications", nil).decode(t, &list)
	if list.UnreadCount != 0 {
		t.Fatalf("expected 0 unread: %+v", list)
	}

	b.expect(204, "POST", "/notifications/read-all", nil)
	b.expect(200, "GET", "/notifications", nil).decode(t, &list)
	if list.UnreadCount != 0 {
		t.Fatal("read-all must clear everything")
	}

	// Pagination.
	a.expect(200, "GET", "/notifications?limit=1", nil).decode(t, &list)
	if len(list.Notifications) != 1 || list.NextCursor == "" {
		t.Fatalf("expected a cursor: %+v", list)
	}
	var page2 notifList
	a.expect(200, "GET", "/notifications?limit=5&cursor="+list.NextCursor, nil).decode(t, &page2)
	if len(page2.Notifications) != 1 || page2.Notifications[0].ID == list.Notifications[0].ID {
		t.Fatalf("second page: %+v", page2)
	}
	a.expect(400, "GET", "/notifications?cursor=%25%25", nil)
}

func TestFullUserJourney(t *testing.T) {
	e := newEnv(t)
	// register -> profile -> photos -> discover -> like -> match -> conversation
	a := e.register("Alice")
	a.expect(200, "PUT", "/profile", map[string]any{"firstName": "Alice", "birthDate": "1993-04-12", "gender": "woman", "interests": []int{1, 3}})
	a.expect(200, "PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 25, "maxAge": 45, "showDistance": true})
	a.upload(testJPEG(800, 1000), nil, 201)

	b := e.register("Bruno")
	b.expect(200, "PUT", "/profile", map[string]any{"firstName": "Bruno", "birthDate": "1990-09-30", "gender": "man", "interests": []int{3}})
	b.expect(200, "PUT", "/preferences", map[string]any{"interestedIn": []string{"woman"}, "minAge": 25, "maxAge": 45, "showDistance": true})
	b.upload(testJPEG(800, 1000), nil, 201)

	feed := b.discover()
	if len(feed) != 1 || feed[0].UserID != a.ID {
		t.Fatalf("Bruno should discover Alice: %+v", feed)
	}
	if res := b.swipe(a, "like"); res.Matched {
		t.Fatal("no match yet")
	}
	res := a.swipe(b, "like")
	if !res.Matched {
		t.Fatal("expected match")
	}
	conv := res.Match.ConversationID
	a.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "Bonjour Bruno"})
	var page msgPage
	b.expect(200, "GET", "/conversations/"+conv+"/messages", nil).decode(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Bonjour Bruno" || page.Messages[0].SenderID != a.ID {
		t.Fatalf("journey: %+v", page)
	}
}
