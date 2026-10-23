package main

import (
	"net/http"
	"strings"
	"testing"
)

func send(e *env, from user, matchID, content string) resp {
	return e.do(http.MethodPost, "/matches/"+matchID+"/messages", map[string]string{"content": content}, from.Token)
}

func listMessages(e *env, u user, matchID, query string) map[string]any {
	e.t.Helper()
	return e.expect(e.do(http.MethodGet, "/matches/"+matchID+"/messages"+query, nil, u.Token), http.StatusOK).JSON(e.t)
}

func matchSummary(e *env, u user, matchID string) map[string]any {
	e.t.Helper()
	for _, m := range e.expect(e.do(http.MethodGet, "/matches", nil, u.Token), http.StatusOK).JSON(e.t)["matches"].([]any) {
		if mm := m.(map[string]any); mm["id"] == matchID {
			return mm
		}
	}
	return nil
}

func TestChatSendReadAndUnread(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)

	msg := e.expect(send(e, a, matchID, "  Salut Berta !  "), http.StatusCreated).JSON(t)
	if msg["content"] != "Salut Berta !" || msg["senderId"] != a.ID || msg["readAt"] != nil {
		t.Fatalf("unexpected message %v", msg)
	}
	e.expect(send(e, a, matchID, "Tu es là ?"), http.StatusCreated)

	s := matchSummary(e, b, matchID)
	if s["unreadCount"].(float64) != 2 || s["lastMessage"].(map[string]any)["content"] != "Tu es là ?" {
		t.Fatalf("unexpected summary for recipient: %v", s)
	}
	if matchSummary(e, a, matchID)["unreadCount"].(float64) != 0 {
		t.Fatal("the sender has no unread messages")
	}

	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/read", nil, b.Token), http.StatusNoContent)
	if matchSummary(e, b, matchID)["unreadCount"].(float64) != 0 {
		t.Fatal("unread count must drop after reading")
	}
	for _, m := range listMessages(e, a, matchID, "")["messages"].([]any) {
		if m.(map[string]any)["readAt"] == nil {
			t.Fatal("sender must see read receipts")
		}
	}
	// Reading your own messages does nothing.
	e.expect(send(e, b, matchID, "Oui !"), http.StatusCreated)
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/read", nil, b.Token), http.StatusNoContent)
	if matchSummary(e, a, matchID)["unreadCount"].(float64) != 1 {
		t.Fatal("a's unread count must still be 1")
	}
}

func TestChatPaginationNewestFirst(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)
	for i := 0; i < 7; i++ {
		e.expect(send(e, a, matchID, "message "+string(rune('A'+i))), http.StatusCreated)
	}
	page1 := listMessages(e, b, matchID, "?limit=3")
	msgs := page1["messages"].([]any)
	if len(msgs) != 3 || msgs[0].(map[string]any)["content"] != "message G" || page1["nextCursor"] == nil {
		t.Fatalf("unexpected first page %v", page1)
	}
	var seen []string
	cursor := ""
	for {
		q := "?limit=3"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		page := listMessages(e, b, matchID, q)
		for _, m := range page["messages"].([]any) {
			seen = append(seen, m.(map[string]any)["content"].(string))
		}
		if page["nextCursor"] == nil {
			break
		}
		cursor = page["nextCursor"].(string)
	}
	if strings.Join(seen, "") != "message Gmessage Fmessage Emessage Dmessage Cmessage Bmessage A" {
		t.Fatalf("pagination lost or duplicated messages: %v", seen)
	}
	e.expect(e.do(http.MethodGet, "/matches/"+matchID+"/messages?cursor=@@@", nil, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodGet, "/matches/"+matchID+"/messages?limit=1000", nil, a.Token), http.StatusBadRequest)
}

func TestChatIsPrivateToParticipants(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)
	e.expect(send(e, a, matchID, "secret"), http.StatusCreated)

	outsider := e.register("outsider")
	e.onboard(outsider, "Olivier", profileOpts{gender: "man", interestedIn: []string{"woman"}})
	e.expect(e.do(http.MethodGet, "/matches/"+matchID+"/messages", nil, outsider.Token), http.StatusNotFound)
	e.expect(send(e, outsider, matchID, "intrusion"), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/read", nil, outsider.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodDelete, "/matches/"+matchID, nil, outsider.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/hide", nil, outsider.Token), http.StatusNotFound)
	if matchSummary(e, outsider, matchID) != nil {
		t.Fatal("outsider must not see the match")
	}

	// Non-matched users cannot message each other even if they know ids.
	c := e.register("nomatch")
	e.onboard(c, "Cora", profileOpts{gender: "woman", interestedIn: []string{"man"}})
	e.expect(e.do(http.MethodGet, "/matches/123e4567-e89b-12d3-a456-426614174000/messages", nil, c.Token), http.StatusNotFound)
}

func TestChatMessageValidation(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)
	e.expect(send(e, a, matchID, "   "), http.StatusBadRequest)
	e.expect(send(e, a, matchID, strings.Repeat("x", 2001)), http.StatusBadRequest)
	e.expect(send(e, a, matchID, strings.Repeat("é", 2000)), http.StatusCreated)
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/messages", map[string]int{"content": 5}, a.Token), http.StatusBadRequest)
}

func TestHideConversationAffectsOnlyMe(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)
	e.expect(send(e, a, matchID, "avant"), http.StatusCreated)

	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/hide", nil, b.Token), http.StatusNoContent)
	if matchSummary(e, b, matchID) != nil {
		t.Fatal("hidden conversation must leave b's list")
	}
	if matchSummary(e, a, matchID) == nil {
		t.Fatal("a's list is unaffected")
	}
	if n := len(listMessages(e, b, matchID, "")["messages"].([]any)); n != 0 {
		t.Fatalf("b must not see history before hiding, got %d", n)
	}
	if n := len(listMessages(e, a, matchID, "")["messages"].([]any)); n != 1 {
		t.Fatalf("a keeps history, got %d", n)
	}

	// A new message brings it back, with only the new history.
	e.expect(send(e, a, matchID, "après"), http.StatusCreated)
	s := matchSummary(e, b, matchID)
	if s == nil || s["unreadCount"].(float64) != 1 {
		t.Fatalf("conversation should resurface with 1 unread: %v", s)
	}
	if n := len(listMessages(e, b, matchID, "")["messages"].([]any)); n != 1 {
		t.Fatalf("b sees only messages after hiding, got %d", n)
	}
}

func TestUnmatchEndsConversationForBoth(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)
	e.expect(send(e, a, matchID, "hello"), http.StatusCreated)

	e.expect(e.do(http.MethodDelete, "/matches/"+matchID, nil, b.Token), http.StatusNoContent)
	e.expect(send(e, a, matchID, "still there?"), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/matches/"+matchID+"/messages", nil, b.Token), http.StatusNotFound)
	if matchSummary(e, a, matchID) != nil || matchSummary(e, b, matchID) != nil {
		t.Fatal("unmatched conversation must disappear for both")
	}
	e.expect(e.do(http.MethodDelete, "/matches/"+matchID, nil, a.Token), http.StatusNotFound)
	// They cannot re-match: both decisions are final and neither is shown again.
	if contains(e.discoverIDs(a), b.ID) || contains(e.discoverIDs(b), a.ID) {
		t.Fatal("unmatched users must not reappear in discovery")
	}
}

func TestMatchListPagination(t *testing.T) {
	e := newEnv(t)
	me := e.register("many")
	e.onboard(me, "Max", profileOpts{gender: "man", interestedIn: []string{"woman"}})
	var matchIDs []string
	for i := 0; i < 3; i++ {
		other := e.register("manyw")
		e.onboard(other, "Wanda", profileOpts{gender: "woman", interestedIn: []string{"man"}})
		matchIDs = append(matchIDs, e.match(me, other))
	}
	page1 := e.expect(e.do(http.MethodGet, "/matches?limit=2", nil, me.Token), http.StatusOK).JSON(t)
	if len(page1["matches"].([]any)) != 2 || page1["nextCursor"] == nil {
		t.Fatalf("unexpected first page %v", page1)
	}
	page2 := e.expect(e.do(http.MethodGet, "/matches?limit=2&cursor="+page1["nextCursor"].(string), nil, me.Token), http.StatusOK).JSON(t)
	if len(page2["matches"].([]any)) != 1 || page2["nextCursor"] != nil {
		t.Fatalf("unexpected second page %v", page2)
	}
	// Activity reorders: the oldest match jumps to the top after a message.
	e.expect(send(e, me, matchIDs[0], "bump"), http.StatusCreated)
	top := e.expect(e.do(http.MethodGet, "/matches?limit=1", nil, me.Token), http.StatusOK).JSON(t)["matches"].([]any)[0].(map[string]any)
	if top["id"] != matchIDs[0] {
		t.Fatalf("most recent activity must come first: %v", top["id"])
	}
}
