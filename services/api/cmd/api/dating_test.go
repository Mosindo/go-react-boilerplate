package main

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestProfileValidation(t *testing.T) {
	r := newRouter()
	u := register(t, r)

	put := func(body map[string]any) int {
		return do(t, r, http.MethodPut, "/profile/me", body, u.Token).Status
	}
	if s := put(map[string]any{"firstName": "Kid", "birthDate": "2015-01-01", "gender": "man"}); s != http.StatusForbidden {
		t.Errorf("underage must be refused (403), got %d", s)
	}
	if s := put(map[string]any{"firstName": "X", "birthDate": "not-a-date", "gender": "man"}); s != 400 {
		t.Errorf("bad date: %d", s)
	}
	if s := put(map[string]any{"firstName": "X", "birthDate": "1990-01-01", "gender": "robot"}); s != 400 {
		t.Errorf("bad gender: %d", s)
	}
	if s := put(map[string]any{"firstName": "", "birthDate": "1990-01-01", "gender": "man"}); s != 400 {
		t.Errorf("empty name: %d", s)
	}
	if s := put(map[string]any{"firstName": "Ok", "birthDate": "1990-01-01", "gender": "man", "interests": []string{"does-not-exist"}}); s != 400 {
		t.Errorf("unknown interest: %d", s)
	}
	if s := put(map[string]any{"firstName": "Ok", "birthDate": "1990-01-01", "gender": "man", "bio": strings.Repeat("a", 501)}); s != 400 {
		t.Errorf("long bio: %d", s)
	}
	// The failed unknown-interest call above already created the profile; the birth date is now immutable.
	if s := put(map[string]any{"firstName": "Ok", "birthDate": "1980-01-01", "gender": "man"}); s != 400 {
		t.Errorf("birth date change must be refused: %d", s)
	}
	if res := do(t, r, http.MethodPut, "/profile/location", map[string]any{"latitude": 120, "longitude": 0}, u.Token); res.Status != 400 {
		t.Errorf("bad latitude: %d", res.Status)
	}
	if res := do(t, r, http.MethodPut, "/profile/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30, "maxDistanceKm": 10}, u.Token); res.Status != 400 {
		t.Errorf("min age 16 must be refused: %d", res.Status)
	}

	var me struct {
		IsComplete bool     `json:"isComplete"`
		Missing    []string `json:"missing"`
	}
	do(t, r, http.MethodGet, "/profile/me", nil, u.Token).json(t, &me)
	if me.IsComplete || len(me.Missing) != 2 {
		t.Errorf("expected incomplete profile (preferences, photos): %+v", me)
	}
}

func TestLocationIsCoarsenedAndNeverExposed(t *testing.T) {
	r := newRouter()
	a := onboard(t, r, "Alice", "woman", []string{"man"}, 0, 0)
	b := onboard(t, r, "Bob", "man", []string{"woman"}, 0, 0)
	do(t, r, http.MethodPut, "/profile/location", map[string]any{"latitude": 48.856614, "longitude": 2.352222, "city": "Paris"}, a.Token)
	do(t, r, http.MethodPut, "/profile/location", map[string]any{"latitude": 48.9012345, "longitude": 2.4012345, "city": "Saint-Denis"}, b.Token)

	var lat, lng float64
	if err := testPool.QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("stored coordinates must be rounded to 2 decimals, got %v,%v", lat, lng)
	}

	res := do(t, r, http.MethodGet, "/profiles/"+b.ID, nil, a.Token)
	if res.Status != 200 {
		t.Fatalf("view profile: %d %s", res.Status, res.Body)
	}
	body := string(res.Body)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email", "48.9", "2.40"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("public profile leaks %q: %s", forbidden, body)
		}
	}
	var p deckProfile
	res.json(t, &p)
	if p.DistanceKm == nil || *p.DistanceKm%5 != 0 {
		t.Errorf("distance must be bucketed to 5 km, got %v", p.DistanceKm)
	}
}

func TestMatchingFlow(t *testing.T) {
	r := newRouter()
	alice := onboard(t, r, "Alice", "woman", []string{"man"}, 48.85, 2.35)
	bob := onboard(t, r, "Bob", "man", []string{"woman"}, 48.87, 2.36)
	// Carol is a woman interested in women: she must never show up for Bob, nor Bob for her.
	carol := onboard(t, r, "Carol", "woman", []string{"woman"}, 48.86, 2.34)
	// Far away: outside the 100 km radius.
	lyon := onboard(t, r, "Lyon", "man", []string{"woman"}, 45.76, 4.83)

	deckA := deck(t, r, alice)
	if !hasProfile(deckA, bob.ID) {
		t.Fatal("alice should see bob")
	}
	if hasProfile(deckA, carol.ID) || hasProfile(deckA, lyon.ID) || hasProfile(deckA, alice.ID) {
		t.Fatalf("deck contains ineligible profiles: %+v", deckA)
	}
	if hasProfile(deck(t, r, bob), carol.ID) {
		t.Fatal("bob must not see carol (orientation mismatch)")
	}

	// Cannot swipe yourself, with an invalid action, or on a stranger id.
	if s, _ := swipe(t, r, alice, alice.ID, "like"); s != 400 {
		t.Errorf("self swipe: %d", s)
	}
	if s, _ := swipe(t, r, alice, bob.ID, "superlike"); s != 400 {
		t.Errorf("bad action: %d", s)
	}
	if s, _ := swipe(t, r, alice, "00000000-0000-0000-0000-000000000000", "like"); s != 404 {
		t.Errorf("unknown target: %d", s)
	}

	// A likes B: no match yet.
	s, res := swipe(t, r, alice, bob.ID, "like")
	if s != 200 || res.Result != "liked" || res.Match != nil || res.AlreadyProcessed {
		t.Fatalf("first like: %d %+v", s, res)
	}
	// Repeating it is idempotent and creates nothing.
	if _, again := swipe(t, r, alice, bob.ID, "like"); !again.AlreadyProcessed || again.Result != "liked" {
		t.Fatalf("repeat like: %+v", again)
	}
	// Alice cannot change her mind through a later pass either.
	if _, again := swipe(t, r, alice, bob.ID, "pass"); !again.AlreadyProcessed {
		t.Fatalf("pass after like should be ignored: %+v", again)
	}
	if hasProfile(deck(t, r, alice), bob.ID) {
		t.Fatal("judged profile must not reappear")
	}

	// B sees A in "likes received" and at the top of the deck.
	var likes struct{ Profiles []deckProfile }
	do(t, r, http.MethodGet, "/likes/received", nil, bob.Token).json(t, &likes)
	if !hasProfile(likes.Profiles, alice.ID) {
		t.Fatal("bob should see alice in likes received")
	}
	// Bob has a notification about the like.
	var notifs struct {
		Notifications []struct{ Type string }
		Unread        int
	}
	do(t, r, http.MethodGet, "/notifications", nil, bob.Token).json(t, &notifs)
	if notifs.Unread != 1 || notifs.Notifications[0].Type != "like" {
		t.Fatalf("expected one like notification: %+v", notifs)
	}

	// B likes A back: match for both.
	_, matched := swipe(t, r, bob, alice.ID, "like")
	if matched.Result != "matched" || matched.Match == nil || matched.Match.ConversationID == "" {
		t.Fatalf("expected match: %+v", matched)
	}
	// Replaying returns the same match, without duplicates.
	_, replay := swipe(t, r, bob, alice.ID, "like")
	if replay.Match == nil || replay.Match.MatchID != matched.Match.MatchID || !replay.AlreadyProcessed {
		t.Fatalf("replay should return the same match: %+v", replay)
	}
	var matchCount int
	_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a_id IN ($1,$2) AND user_b_id IN ($1,$2)`, alice.ID, bob.ID).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchCount)
	}
	// Both are notified of the match.
	for _, u := range []testUser{alice, bob} {
		var n struct{ Notifications []struct{ Type string } }
		do(t, r, http.MethodGet, "/notifications", nil, u.Token).json(t, &n)
		found := false
		for _, x := range n.Notifications {
			found = found || x.Type == "match"
		}
		if !found {
			t.Errorf("%s missing match notification", u.Email)
		}
	}

	// Pass hides a profile permanently without creating a match.
	if _, p := swipe(t, r, alice, lyon.ID, "pass"); p.Result != "" && p.Result != "passed" {
		t.Fatalf("pass: %+v", p)
	}
}

func TestSwipeRequiresCompleteProfile(t *testing.T) {
	r := newRouter()
	bob := onboard(t, r, "Bob", "man", []string{"woman"}, 0, 0)
	rookie := register(t, r)
	if s, _ := swipe(t, r, rookie, bob.ID, "like"); s != http.StatusForbidden {
		t.Fatalf("incomplete profile must not swipe: %d", s)
	}
	if got := deck(t, r, rookie); len(got) != 0 {
		t.Fatalf("incomplete profile must get an empty deck: %+v", got)
	}
}

func TestConcurrentOppositeLikesCreateExactlyOneMatch(t *testing.T) {
	r := newRouter()
	for round := 0; round < 5; round++ {
		a := onboard(t, r, "A", "woman", []string{"man"}, 0, 0)
		b := onboard(t, r, "B", "man", []string{"woman"}, 0, 0)
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]testUser{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, results[i] = swipe(t, r, pair[0], pair[1].ID, "like")
			}()
		}
		wg.Wait()
		matches := 0
		for _, res := range results {
			if res.Result == "matched" {
				matches++
			}
		}
		if matches != 1 {
			t.Fatalf("round %d: exactly one of the two simultaneous likes must report the match, got %+v", round, results)
		}
		var rows int
		_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a_id IN ($1,$2) AND user_b_id IN ($1,$2)`, a.ID, b.ID).Scan(&rows)
		if rows != 1 {
			t.Fatalf("round %d: %d match rows", round, rows)
		}
	}
}

type convList struct {
	Conversations []struct {
		ID          string `json:"id"`
		Unread      int    `json:"unread"`
		LastMessage *struct {
			Body string `json:"body"`
		} `json:"lastMessage"`
		With struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"with"`
	} `json:"conversations"`
}

func matchPair(t *testing.T, r http.Handler) (testUser, testUser, string) {
	t.Helper()
	a := onboard(t, r, "Alice", "woman", []string{"man"}, 0, 0)
	b := onboard(t, r, "Bob", "man", []string{"woman"}, 0, 0)
	swipe(t, r, a, b.ID, "like")
	_, m := swipe(t, r, b, a.ID, "like")
	if m.Match == nil {
		t.Fatal("expected match")
	}
	return a, b, m.Match.ConversationID
}

func TestChatFlowAndPermissions(t *testing.T) {
	r := newRouter()
	a, b, conv := matchPair(t, r)
	stranger := onboard(t, r, "Eve", "woman", []string{"man"}, 0, 0)

	// The new match shows up as a conversation without messages.
	var list convList
	do(t, r, http.MethodGet, "/conversations", nil, a.Token).json(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].With.FirstName != "Bob" || list.Conversations[0].LastMessage != nil {
		t.Fatalf("unexpected conversation list: %+v", list)
	}

	if res := do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "   "}, a.Token); res.Status != 400 {
		t.Errorf("blank message: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": strings.Repeat("x", 2001)}, a.Token); res.Status != 400 {
		t.Errorf("long message: %d", res.Status)
	}
	send := do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "Salut Bob !"}, a.Token)
	if send.Status != 201 {
		t.Fatalf("send: %d %s", send.Status, send.Body)
	}

	// Bob has 1 unread; reading resets it, and Alice sees the read pointer.
	do(t, r, http.MethodGet, "/conversations", nil, b.Token).json(t, &list)
	if list.Conversations[0].Unread != 1 || list.Conversations[0].LastMessage.Body != "Salut Bob !" {
		t.Fatalf("bob list: %+v", list)
	}
	var msgs struct {
		Messages []struct {
			ID, Body, SenderID string
		}
		PeerLastReadID string
	}
	do(t, r, http.MethodGet, "/conversations/"+conv+"/messages", nil, b.Token).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.Messages[0].SenderID != a.ID {
		t.Fatalf("messages: %+v", msgs)
	}
	if res := do(t, r, http.MethodPost, "/conversations/"+conv+"/read", nil, b.Token); res.Status != 204 {
		t.Fatalf("read: %d", res.Status)
	}
	do(t, r, http.MethodGet, "/conversations", nil, b.Token).json(t, &list)
	if list.Conversations[0].Unread != 0 {
		t.Fatalf("unread should be 0: %+v", list)
	}
	do(t, r, http.MethodGet, "/conversations/"+conv+"/messages", nil, a.Token).json(t, &msgs)
	if msgs.PeerLastReadID != msgs.Messages[0].ID {
		t.Fatalf("alice should see bob read her message: %+v", msgs)
	}

	// Permissions: an outsider can neither read, write, mark-read, hide nor unmatch.
	for _, c := range []struct{ method, path string }{
		{"GET", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/read"},
		{"POST", "/conversations/" + conv + "/unmatch"},
		{"DELETE", "/conversations/" + conv},
	} {
		var body any
		if c.method == "POST" && strings.HasSuffix(c.path, "messages") {
			body = map[string]string{"body": "hi"}
		}
		if res := do(t, r, c.method, c.path, body, stranger.Token); res.Status != http.StatusNotFound {
			t.Errorf("%s %s by outsider: expected 404 got %d", c.method, c.path, res.Status)
		}
	}
	var eveList convList
	do(t, r, http.MethodGet, "/conversations", nil, stranger.Token).json(t, &eveList)
	if len(eveList.Conversations) != 0 {
		t.Fatalf("outsider sees conversations: %+v", eveList)
	}
	if res := do(t, r, http.MethodGet, "/conversations/"+conv+"/messages", nil, ""); res.Status != 401 {
		t.Errorf("unauthenticated: %d", res.Status)
	}

	// Local deletion hides the conversation for Alice only; a new message brings it back.
	if res := do(t, r, http.MethodDelete, "/conversations/"+conv, nil, a.Token); res.Status != 204 {
		t.Fatalf("hide: %d", res.Status)
	}
	do(t, r, http.MethodGet, "/conversations", nil, a.Token).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatalf("hidden conversation still listed: %+v", list)
	}
	do(t, r, http.MethodGet, "/conversations", nil, b.Token).json(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("bob's side must be untouched")
	}
	do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "Tu es là ?"}, b.Token)
	do(t, r, http.MethodGet, "/conversations/"+conv+"/messages", nil, a.Token).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.Messages[0].Body != "Tu es là ?" {
		t.Fatalf("alice should only see messages after hiding: %+v", msgs)
	}

	// Pagination of history.
	for i := 0; i < 5; i++ {
		do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "m"}, b.Token)
	}
	var page struct {
		Messages   []struct{ ID string }
		NextBefore string
	}
	do(t, r, http.MethodGet, "/conversations/"+conv+"/messages?limit=2", nil, a.Token).json(t, &page)
	if len(page.Messages) != 2 || page.NextBefore == "" {
		t.Fatalf("page: %+v", page)
	}
	do(t, r, http.MethodGet, "/conversations/"+conv+"/messages?limit=100&before="+page.NextBefore, nil, a.Token).json(t, &msgs)
	if len(msgs.Messages) != 4 {
		t.Fatalf("older page: %d", len(msgs.Messages))
	}

	// Unmatch deletes the conversation for both.
	if res := do(t, r, http.MethodPost, "/conversations/"+conv+"/unmatch", nil, b.Token); res.Status != 204 {
		t.Fatalf("unmatch: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/conversations/"+conv+"/messages", nil, a.Token); res.Status != 404 {
		t.Fatalf("after unmatch: %d", res.Status)
	}
	// ...but they never come back into each other's deck.
	if hasProfile(deck(t, r, a), b.ID) {
		t.Fatal("unmatched profile reappeared")
	}
}

func TestBlockAndReport(t *testing.T) {
	r := newRouter()
	a, b, conv := matchPair(t, r)
	do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "hello"}, b.Token)

	if res := do(t, r, http.MethodPost, "/blocks", map[string]string{"userId": a.ID}, a.Token); res.Status != 400 {
		t.Errorf("self block: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/blocks", map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}, a.Token); res.Status != 404 {
		t.Errorf("unknown block: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token); res.Status != 204 {
		t.Fatalf("block: %d", res.Status)
	}
	// Idempotent.
	if res := do(t, r, http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token); res.Status != 204 {
		t.Fatalf("block twice: %d", res.Status)
	}

	// Block hides everything in both directions.
	var list convList
	for _, u := range []testUser{a, b} {
		do(t, r, http.MethodGet, "/conversations", nil, u.Token).json(t, &list)
		if len(list.Conversations) != 0 {
			t.Errorf("%s still sees blocked conversation", u.Email)
		}
	}
	if res := do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "still there?"}, b.Token); res.Status != 403 {
		t.Errorf("blocked user must not message: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/conversations/"+conv+"/messages", nil, a.Token); res.Status != 404 {
		t.Errorf("blocked history must be hidden: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/profiles/"+a.ID, nil, b.Token); res.Status != 404 {
		t.Errorf("blocker's profile must be hidden: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/profiles/"+b.ID, nil, a.Token); res.Status != 404 {
		t.Errorf("blocked profile must be hidden: %d", res.Status)
	}
	if s, _ := swipe(t, r, b, a.ID, "like"); s != 404 {
		t.Errorf("swipe on blocker: %d", s)
	}
	var blocked struct{ Blocks []struct{ UserID string } }
	do(t, r, http.MethodGet, "/blocks", nil, a.Token).json(t, &blocked)
	if len(blocked.Blocks) != 1 || blocked.Blocks[0].UserID != b.ID {
		t.Fatalf("blocks list: %+v", blocked)
	}

	// Unblock restores visibility.
	if res := do(t, r, http.MethodDelete, "/blocks/"+b.ID, nil, a.Token); res.Status != 204 {
		t.Fatalf("unblock: %d", res.Status)
	}
	do(t, r, http.MethodGet, "/conversations", nil, a.Token).json(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("conversation should be back after unblock")
	}

	// Reports.
	if res := do(t, r, http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "bogus"}, a.Token); res.Status != 400 {
		t.Errorf("bad reason: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, a.Token); res.Status != 400 {
		t.Errorf("self report: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "harassment", "details": "rude", "alsoBlock": true}, a.Token); res.Status != 201 {
		t.Fatalf("report: %d", res.Status)
	}
	do(t, r, http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "harassment"}, a.Token)
	var reports, blocks int
	_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&reports)
	_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, a.ID, b.ID).Scan(&blocks)
	if reports != 1 || blocks != 1 {
		t.Fatalf("expected one open report and a block, got %d/%d", reports, blocks)
	}
}

func TestPhotos(t *testing.T) {
	r := newRouter()
	u := register(t, r)
	other := register(t, r)

	if res := upload(t, r, http.MethodPost, "/photos", u.Token, []byte("<?php echo 1; ?>")); res.Status != http.StatusUnsupportedMediaType {
		t.Errorf("non-image upload: %d", res.Status)
	}
	if res := upload(t, r, http.MethodPost, "/photos", u.Token, pngImage(50, 50)); res.Status != 400 {
		t.Errorf("tiny image: %d", res.Status)
	}

	var ids []string
	for i := 0; i < 6; i++ {
		res := upload(t, r, http.MethodPost, "/photos", u.Token, pngImage(300+i, 400))
		if res.Status != 201 {
			t.Fatalf("upload %d: %d %s", i, res.Status, res.Body)
		}
		var p struct{ ID, URL string }
		res.json(t, &p)
		ids = append(ids, p.ID)
	}
	if res := upload(t, r, http.MethodPost, "/photos", u.Token, pngImage(300, 400)); res.Status != http.StatusConflict {
		t.Errorf("7th photo must be refused: %d", res.Status)
	}

	// Main photo + reorder.
	var out struct {
		Photos []struct {
			ID       string
			Position int
		}
	}
	if res := do(t, r, http.MethodPut, "/photos/"+ids[3]+"/primary", nil, u.Token); res.Status != 200 {
		t.Fatalf("primary: %d %s", res.Status, res.Body)
	} else {
		res.json(t, &out)
	}
	if out.Photos[0].ID != ids[3] {
		t.Fatalf("primary not first: %+v", out)
	}
	if res := do(t, r, http.MethodPut, "/photos/order", map[string]any{"ids": []string{ids[0]}}, u.Token); res.Status != 400 {
		t.Errorf("partial order must be refused: %d", res.Status)
	}

	// Other users cannot touch my photos.
	if res := do(t, r, http.MethodDelete, "/photos/"+ids[0], nil, other.Token); res.Status != 404 {
		t.Errorf("delete foreign photo: %d", res.Status)
	}
	if res := upload(t, r, http.MethodPut, "/photos/"+ids[0], other.Token, pngImage(300, 300)); res.Status != 404 {
		t.Errorf("replace foreign photo: %d", res.Status)
	}

	// Signed URL serves the file; a tampered one does not.
	var list struct{ Photos []struct{ ID, URL string } }
	do(t, r, http.MethodGet, "/photos", nil, u.Token).json(t, &list)
	file := do(t, r, http.MethodGet, list.Photos[0].URL, nil, "")
	if file.Status != 200 || file.Header.Get("Content-Type") != "image/jpeg" || file.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("file: %d %v", file.Status, file.Header)
	}
	bad := strings.Replace(list.Photos[0].URL, "sig=", "sig=00", 1)
	if res := do(t, r, http.MethodGet, bad, nil, ""); res.Status != 404 {
		t.Errorf("tampered signature: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/photos/"+list.Photos[0].ID+"/file", nil, ""); res.Status != 404 {
		t.Errorf("unsigned access: %d", res.Status)
	}

	// Replace keeps the position; delete compacts positions and removes the file.
	if res := upload(t, r, http.MethodPut, "/photos/"+ids[3], u.Token, pngImage(320, 320)); res.Status != 200 {
		t.Fatalf("replace: %d %s", res.Status, res.Body)
	}
	var key string
	_ = testPool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE id = $1`, ids[1]).Scan(&key)
	path, _ := testStore.Path(key)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}
	if res := do(t, r, http.MethodDelete, "/photos/"+ids[1], nil, u.Token); res.Status != 204 {
		t.Fatalf("delete: %d", res.Status)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should be removed from disk: %v", err)
	}
	do(t, r, http.MethodGet, "/photos", nil, u.Token).json(t, &out)
	if len(out.Photos) != 5 {
		t.Fatalf("expected 5 photos: %+v", out)
	}
	for i, p := range out.Photos {
		if p.Position != i {
			t.Errorf("positions must stay contiguous: %+v", out.Photos)
		}
	}
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	r := newRouter()
	a, b, conv := matchPair(t, r)
	do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "bye"}, a.Token)
	do(t, r, http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "spam"}, a.Token)

	var key string
	_ = testPool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID).Scan(&key)
	path, _ := testStore.Path(key)

	if res := do(t, r, http.MethodDelete, "/me", map[string]string{"password": "WrongPass123"}, a.Token); res.Status != http.StatusForbidden {
		t.Fatalf("wrong password must be refused: %d", res.Status)
	}
	if res := do(t, r, http.MethodDelete, "/me", map[string]string{"password": "Passw0rdTest"}, a.Token); res.Status != 204 {
		t.Fatalf("delete account: %d %s", res.Status, res.Body)
	}
	if res := do(t, r, http.MethodGet, "/me", nil, a.Token); res.Status != 401 {
		t.Fatalf("deleted account token still works: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/auth/login", map[string]string{"email": a.Email, "password": "Passw0rdTest"}, ""); res.Status != 401 {
		t.Fatalf("deleted account can still login: %d", res.Status)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("photo file must be deleted: %v", err)
	}
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "swipes": "from_user_id", "messages": "sender_id", "reports": "reporter_id", "notifications": "user_id", "sessions": "user_id"} {
		var n int
		_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM `+table+` WHERE `+col+` = $1`, a.ID).Scan(&n)
		if n != 0 {
			t.Errorf("%s still has %d rows for the deleted user", table, n)
		}
	}
	var convs int
	_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM conversations WHERE id = $1`, conv).Scan(&convs)
	if convs != 0 {
		t.Error("conversation must be deleted with the match")
	}
	// The other person is unaffected and no longer sees the deleted user.
	do(t, r, http.MethodGet, "/conversations", nil, b.Token).json(t, &convList{})
	if res := do(t, r, http.MethodGet, "/me", nil, b.Token); res.Status != 200 {
		t.Fatalf("other user broken: %d", res.Status)
	}
}

func TestChangePasswordKeepsCurrentSessionOnly(t *testing.T) {
	r := newRouter()
	u := register(t, r)
	login := do(t, r, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "Passw0rdTest"}, "")
	var second struct{ AccessToken string }
	login.json(t, &second)

	if res := do(t, r, http.MethodPut, "/me/password", map[string]string{"currentPassword": "bad", "newPassword": "NewPassw0rd"}, u.Token); res.Status != 403 {
		t.Fatalf("wrong current password: %d", res.Status)
	}
	if res := do(t, r, http.MethodPut, "/me/password", map[string]string{"currentPassword": "Passw0rdTest", "newPassword": "weak"}, u.Token); res.Status != 400 {
		t.Fatalf("weak new password: %d", res.Status)
	}
	if res := do(t, r, http.MethodPut, "/me/password", map[string]string{"currentPassword": "Passw0rdTest", "newPassword": "NewPassw0rd"}, u.Token); res.Status != 204 {
		t.Fatalf("change: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/me", nil, u.Token); res.Status != 200 {
		t.Fatalf("current session must survive: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/me", nil, second.AccessToken); res.Status != 401 {
		t.Fatalf("other sessions must be revoked: %d", res.Status)
	}
}

func TestNotificationsReadFlow(t *testing.T) {
	r := newRouter()
	a, b, _ := matchPair(t, r)
	_ = a
	var n struct {
		Notifications []struct {
			ID     string
			IsRead bool
		}
		Unread int
	}
	do(t, r, http.MethodGet, "/notifications?limit=1", nil, b.Token).json(t, &n)
	if n.Unread < 1 || len(n.Notifications) != 1 {
		t.Fatalf("notifications: %+v", n)
	}
	if res := do(t, r, http.MethodPost, "/notifications/"+n.Notifications[0].ID+"/read", nil, a.Token); res.Status != 404 {
		t.Errorf("reading someone else's notification: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/notifications/"+n.Notifications[0].ID+"/read", nil, b.Token); res.Status != 204 {
		t.Fatalf("mark read: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/notifications/read-all", nil, b.Token); res.Status != 204 {
		t.Fatalf("read all: %d", res.Status)
	}
	do(t, r, http.MethodGet, "/notifications", nil, b.Token).json(t, &n)
	if n.Unread != 0 {
		t.Fatalf("unread after read-all: %d", n.Unread)
	}
}
