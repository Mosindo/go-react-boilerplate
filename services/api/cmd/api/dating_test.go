package main

import (
	"net/http"
	"sync"
	"testing"
)

func TestDiscoveryAppliesMutualPreferencesAndExclusions(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)

	// Wrong gender preference, outside the age window, too far away, not discoverable.
	wantsWomen := e.onboard("wants-women", profileSpec{name: "Walt", gender: "man", interestedIn: []string{"man"}, at: at})
	tooYoungForAlice := e.onboard("young", profileSpec{name: "Yann", gender: "man", birth: "2004-01-01", interestedIn: []string{"woman"}, at: at})
	aliceTooOld := e.onboard("picky", profileSpec{name: "Paul", gender: "man", interestedIn: []string{"woman"}, minAge: 18, maxAge: 25, at: at})
	far := e.onboard("far", profileSpec{name: "Far", gender: "man", interestedIn: []string{"woman"}, at: geo{at.lat + 3, at.lon}})
	hidden := e.onboard("hidden", profileSpec{name: "Hugo", gender: "man", interestedIn: []string{"woman"}, at: at})
	e.expect(e.do("PUT", "/profile", map[string]any{"displayName": "Hugo", "birthDate": "1992-04-10", "gender": "man", "isDiscoverable": false}, hidden.Token), http.StatusOK)

	// Alice only wants men aged 30-40 within 50 km.
	e.expect(e.do("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 40, "maxDistanceKm": 50}, a.Token), http.StatusOK)

	seen := e.discover(a)
	if !containsProfile(seen, b.ID) {
		t.Fatalf("Alice should see Bob: %+v", seen)
	}
	for name, u := range map[string]user{"wrong orientation": wantsWomen, "too young": tooYoungForAlice, "too far": far, "hidden": hidden} {
		if containsProfile(seen, u.ID) {
			t.Errorf("%s must not be discoverable", name)
		}
	}
	// Mutual: Paul only accepts 18-25 so Alice (born 1992) must not see him either way.
	if containsProfile(seen, aliceTooOld.ID) || containsProfile(e.discover(aliceTooOld), a.ID) {
		t.Error("age preferences must be mutual")
	}

	// Public cards never expose exact location or birth date.
	for _, p := range seen {
		if p.UserID == b.ID {
			if p.DistanceKm == nil || *p.DistanceKm != 1 {
				t.Fatalf("same-spot distance should be rounded up to 1 km, got %v", p.DistanceKm)
			}
			if len(p.Photos) != 1 || p.Photos[0].URL == "" || p.Age < 30 {
				t.Fatalf("unexpected card %+v", p)
			}
		}
	}
	card := e.expect(e.do("GET", "/profiles/"+b.ID, nil, a.Token), http.StatusOK)
	for _, forbidden := range []string{"birthDate", "latitude", "longitude", "email"} {
		if containsStr(string(card.Body), forbidden) {
			t.Fatalf("public profile leaks %s: %s", forbidden, card.Body)
		}
	}
	e.expect(e.do("GET", "/profiles/"+hidden.ID, nil, a.Token), http.StatusNotFound)
}

func TestIncompleteProfileCannotDiscoverOrSwipe(t *testing.T) {
	e := newEnv(t)
	u := e.register("incomplete")
	e.expect(e.do("GET", "/discover", nil, u.Token), http.StatusConflict)
	e.expect(e.do("POST", "/swipes", map[string]string{"targetUserId": "11111111-1111-1111-1111-111111111111", "action": "like"}, u.Token), http.StatusConflict)
}

func TestLikeMatchAndConversationJourney(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)

	// Alice likes Bob: no match yet, repeat is idempotent, Bob stays out of her stack.
	first := e.swipe(a, b, "like")
	if first.Matched || first.Duplicate {
		t.Fatalf("unexpected first swipe %+v", first)
	}
	if again := e.swipe(a, b, "like"); !again.Duplicate || again.Matched {
		t.Fatalf("repeat like must be a no-op duplicate: %+v", again)
	}
	if again := e.swipe(a, b, "pass"); !again.Duplicate || again.Action != "like" {
		t.Fatalf("a decision cannot be flipped by re-swiping: %+v", again)
	}
	if containsProfile(e.discover(a), b.ID) {
		t.Fatal("swiped profiles must not reappear")
	}
	// Bob sees Alice first because she already liked him.
	if !containsProfile(e.discover(b), a.ID) {
		t.Fatal("Bob should see Alice")
	}

	second := e.swipe(b, a, "like")
	if !second.Matched || second.Match == nil || second.Match.ConversationID == "" {
		t.Fatalf("expected a match: %+v", second)
	}
	convID := second.Match.ConversationID
	if replay := e.swipe(a, b, "like"); !replay.Matched || replay.Match.MatchID != second.Match.MatchID {
		t.Fatalf("replay should report the same match: %+v", replay)
	}

	var matches int
	if err := e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&matches); err != nil || matches != 1 {
		t.Fatalf("exactly one match row expected, got %d err=%v", matches, err)
	}

	// Both get a match notification.
	for _, u := range []user{a, b} {
		var n struct {
			Notifications []struct {
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		e.expect(e.do("GET", "/notifications", nil, u.Token), http.StatusOK).decode(t, &n)
		if len(n.Notifications) != 1 || n.Notifications[0].Type != "match" || n.UnreadCount != 1 {
			t.Fatalf("expected one unread match notification, got %+v", n)
		}
	}

	// Conversation exists for both, no messages yet.
	type convList struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			User        struct {
				UserID      string `json:"userId"`
				DisplayName string `json:"displayName"`
				PhotoURL    string `json:"photoUrl"`
			} `json:"user"`
			LastMessage *struct {
				Body        string `json:"body"`
				ReadByOther bool   `json:"readByOther"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	var al, bl convList
	e.expect(e.do("GET", "/conversations", nil, a.Token), http.StatusOK).decode(t, &al)
	e.expect(e.do("GET", "/conversations", nil, b.Token), http.StatusOK).decode(t, &bl)
	if len(al.Conversations) != 1 || al.Conversations[0].User.UserID != b.ID || al.Conversations[0].User.PhotoURL == "" || al.Conversations[0].LastMessage != nil {
		t.Fatalf("unexpected Alice conversations %+v", al)
	}
	if len(bl.Conversations) != 1 || bl.Conversations[0].User.DisplayName != "Alice" {
		t.Fatalf("unexpected Bob conversations %+v", bl)
	}

	// Realtime: Bob waits for events, Alice sends, Bob is woken up.
	cursor := struct {
		Cursor int64 `json:"cursor"`
	}{}
	e.expect(e.do("GET", "/events", nil, b.Token), http.StatusOK).decode(t, &cursor)
	done := make(chan resp, 1)
	go func() { done <- e.do("GET", "/events?after="+itoa(cursor.Cursor), nil, b.Token) }()
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]string{"body": "  Salut Bob !  "}, a.Token), http.StatusCreated)
	var ev struct {
		Events []struct {
			Type           string `json:"type"`
			ConversationID string `json:"conversationId"`
		} `json:"events"`
	}
	(<-done).decode(t, &ev)
	if len(ev.Events) == 0 || ev.Events[0].Type != "message" || ev.Events[0].ConversationID != convID {
		t.Fatalf("Bob should be notified of the message: %+v", ev)
	}

	// Unread/read state.
	e.expect(e.do("GET", "/conversations", nil, b.Token), http.StatusOK).decode(t, &bl)
	if bl.Conversations[0].UnreadCount != 1 || bl.Conversations[0].LastMessage == nil || bl.Conversations[0].LastMessage.Body != "Salut Bob !" {
		t.Fatalf("Bob should have 1 unread: %+v", bl.Conversations[0])
	}
	e.expect(e.do("GET", "/conversations", nil, a.Token), http.StatusOK).decode(t, &al)
	if al.Conversations[0].UnreadCount != 0 {
		t.Fatalf("sender has nothing unread: %+v", al.Conversations[0])
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/read", nil, b.Token), http.StatusNoContent)
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]string{"body": "Salut Alice"}, b.Token), http.StatusCreated)

	var msgs struct {
		Messages []struct {
			SenderID    string `json:"senderId"`
			Body        string `json:"body"`
			ReadByOther bool   `json:"readByOther"`
		} `json:"messages"`
	}
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", nil, a.Token), http.StatusOK).decode(t, &msgs)
	if len(msgs.Messages) != 2 || msgs.Messages[0].Body != "Salut Bob !" || !msgs.Messages[0].ReadByOther || msgs.Messages[1].ReadByOther {
		t.Fatalf("unexpected history (chronological, with read receipts): %+v", msgs.Messages)
	}

	// Pagination.
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]string{"body": "msg " + itoa(int64(i))}, a.Token), http.StatusCreated)
	}
	var page struct {
		Messages   []struct{ Body string } `json:"messages"`
		NextBefore *string                 `json:"nextBefore"`
	}
	e.expect(e.do("GET", "/conversations/"+convID+"/messages?limit=3", nil, a.Token), http.StatusOK).decode(t, &page)
	if len(page.Messages) != 3 || page.NextBefore == nil || page.Messages[2].Body != "msg 4" {
		t.Fatalf("unexpected first page %+v", page)
	}

	// Invalid messages.
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]string{"body": "   "}, a.Token), http.StatusBadRequest)

	// Local deletion hides it for one side only, until a new message arrives.
	e.expect(e.do("DELETE", "/conversations/"+convID, nil, a.Token), http.StatusNoContent)
	e.expect(e.do("GET", "/conversations", nil, a.Token), http.StatusOK).decode(t, &al)
	e.expect(e.do("GET", "/conversations", nil, b.Token), http.StatusOK).decode(t, &bl)
	if len(al.Conversations) != 0 || len(bl.Conversations) != 1 {
		t.Fatalf("hide must be per-user: alice=%d bob=%d", len(al.Conversations), len(bl.Conversations))
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]string{"body": "tu es là ?"}, b.Token), http.StatusCreated)
	e.expect(e.do("GET", "/conversations", nil, a.Token), http.StatusOK).decode(t, &al)
	if len(al.Conversations) != 1 {
		t.Fatal("a new message must resurface the conversation")
	}
}

func TestConversationPermissions(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)
	intruder := e.onboard("eve", profileSpec{name: "Eve", gender: "woman", interestedIn: []string{"man"}, at: at})
	convID := e.forceMatch(a, b)
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]string{"body": "secret"}, a.Token), http.StatusCreated)

	e.expect(e.do("GET", "/conversations/"+convID+"/messages", nil, intruder.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]string{"body": "hi"}, intruder.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", nil, intruder.Token), http.StatusNotFound)
	e.expect(e.do("DELETE", "/conversations/"+convID, nil, intruder.Token), http.StatusNotFound)
	e.expect(e.do("DELETE", "/matches/"+uuidOf(t, e, convID), nil, intruder.Token), http.StatusNotFound)
	e.expect(e.do("GET", "/conversations/not-a-uuid/messages", nil, a.Token), http.StatusBadRequest)

	var il struct {
		Conversations []any `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", nil, intruder.Token), http.StatusOK).decode(t, &il)
	if len(il.Conversations) != 0 {
		t.Fatal("intruder must not see foreign conversations")
	}
	// Eve cannot swipe on Bob out of thin air either unless he is a valid candidate;
	// she is, but she cannot bypass filters for an ineligible target.
	hidden := e.onboard("ghost", profileSpec{name: "Ghost", gender: "man", interestedIn: []string{"woman"}, at: at})
	e.expect(e.do("PUT", "/profile", map[string]any{"displayName": "Ghost", "birthDate": "1992-04-10", "gender": "man", "isDiscoverable": false}, hidden.Token), http.StatusOK)
	e.expect(e.do("POST", "/swipes", map[string]string{"targetUserId": hidden.ID, "action": "like"}, intruder.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/swipes", map[string]string{"targetUserId": "11111111-1111-1111-1111-111111111111", "action": "like"}, intruder.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/swipes", map[string]string{"targetUserId": intruder.ID, "action": "like"}, intruder.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/swipes", map[string]string{"targetUserId": b.ID, "action": "superlike"}, intruder.Token), http.StatusBadRequest)
}

func uuidOf(t *testing.T, e *env, conversationID string) string {
	t.Helper()
	var id string
	if err := e.pool.QueryRow(t.Context(), `SELECT match_id::text FROM conversations WHERE id = $1`, conversationID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSimultaneousLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)

	var wg sync.WaitGroup
	results := make([]swipeOut, 2)
	for i, pr := range [][2]user{{a, b}, {b, a}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = e.swipe(pr[0], pr[1], "like")
		}()
	}
	wg.Wait()

	if !results[0].Matched && !results[1].Matched {
		t.Fatalf("at least one side must observe the match: %+v", results)
	}
	var matches, convs int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&matches)
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a = LEAST($1::uuid,$2::uuid) AND m.user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&convs)
	if matches != 1 || convs != 1 {
		t.Fatalf("expected one match and one conversation, got %d and %d", matches, convs)
	}
}

func TestPassNeverMatchesAndUnmatchRemovesConversation(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)

	e.swipe(a, b, "pass")
	if out := e.swipe(b, a, "like"); out.Matched {
		t.Fatal("a pass must never produce a match")
	}

	c := e.onboard("carol", profileSpec{name: "Carol", gender: "woman", interestedIn: []string{"man"}, at: at})
	conv := e.forceMatch(c, b)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hello"}, c.Token), http.StatusCreated)
	e.expect(e.do("DELETE", "/matches/"+uuidOf(t, e, conv), nil, b.Token), http.StatusNoContent)
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, c.Token), http.StatusNotFound)
	if containsProfile(e.discover(c), b.ID) {
		t.Fatal("unmatched users must not be proposed again")
	}
}
